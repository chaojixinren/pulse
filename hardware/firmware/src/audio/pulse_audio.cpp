/**
 * @file pulse_audio.cpp
 * @brief I2S0 PDM RX 音频采集实现（legacy 驱动）
 */
#include "pulse_audio.h"

#include <Arduino.h>
#include <driver/i2s.h>
#include <esp_heap_caps.h>
#include <freertos/FreeRTOS.h>
#include <freertos/queue.h>

#include <string.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "audio";

#define I2S_PORT I2S_NUM_0

static bool g_ready = false;
static audio_stats_t g_stats;

/* I2S 事件队列：legacy 驱动通过它上报 RX DMA 队列溢出（真正可靠的
 * overrun 信号，比「读满请求量」这种启发式准确得多）。 */
static QueueHandle_t g_i2s_evt_queue = nullptr;

/* ─────────── 直流偏移滤除 ───────────
 * PDM MEMS 麦克风普遍带直流偏移，不处理会：
 *   1. 让 VAD 的能量计算长期偏高，静音也被判成语音
 *   2. 浪费动态范围
 * 用一阶高通（差分 + 泄漏积分），截止频率约 20Hz，对语音无损。 */
static int32_t g_dc_acc = 0;

static inline int16_t dc_filter(int16_t s) {
    /* g_dc_acc 是放大 256 倍的直流估计，避免整数除法丢精度 */
    g_dc_acc += ((int32_t)s * 256 - g_dc_acc) >> 8;
    int32_t out = (int32_t)s - (g_dc_acc >> 8);
    if (out > 32767) out = 32767;
    if (out < -32768) out = -32768;
    return (int16_t)out;
}

/* ─────────── pre-roll 环形缓冲 ───────────
 * VAD 确认语音需要 speech_confirm_ms，这段时间的音频若不保留就会丢首字。
 * 缓冲放 PSRAM：preroll 300ms 只需 9.6KB，但放 PSRAM 可以给内部 RAM
 * 留给 LVGL 与 DMA。这里是 CPU 拷贝，不涉及 DMA，PSRAM 完全够用。 */
static int16_t *g_preroll = nullptr;
static size_t g_preroll_cap = 0;   /* 容量（采样点）*/
static size_t g_preroll_head = 0;  /* 下一个写入位置 */
static size_t g_preroll_len = 0;   /* 已缓存的采样点数 */

static bool preroll_alloc(void) {
    const pulse_config_t *c = config_get();
    size_t samples = (size_t)((uint64_t)c->vad.preroll_ms * AUDIO_SAMPLE_RATE / 1000);
    if (samples == 0) samples = 1;

    g_preroll_cap = samples;
    size_t bytes = samples * sizeof(int16_t);

    /* 优先 PSRAM；失败退内部 RAM（preroll 很小，退化可接受）*/
    g_preroll = (int16_t *)heap_caps_malloc(bytes, MALLOC_CAP_SPIRAM);
    if (!g_preroll) {
        g_preroll = (int16_t *)malloc(bytes);
        if (!g_preroll) {
            log_e("%s: pre-roll 缓冲分配失败 (%u 字节)", TAG, (unsigned)bytes);
            return false;
        }
        log_w("%s: pre-roll 落在内部 RAM (%u 字节)", TAG, (unsigned)bytes);
    }
    g_preroll_head = 0;
    g_preroll_len = 0;
    return true;
}

static void preroll_push(const int16_t *pcm, size_t count) {
    if (!g_preroll || g_preroll_cap == 0) return;
    /* 超过容量时只保留最后 cap 个采样点 */
    if (count >= g_preroll_cap) {
        memcpy(g_preroll, pcm + (count - g_preroll_cap), g_preroll_cap * sizeof(int16_t));
        g_preroll_head = 0;
        g_preroll_len = g_preroll_cap;
        return;
    }
    for (size_t i = 0; i < count; i++) {
        g_preroll[g_preroll_head] = pcm[i];
        g_preroll_head = (g_preroll_head + 1) % g_preroll_cap;
    }
    g_preroll_len += count;
    if (g_preroll_len > g_preroll_cap) g_preroll_len = g_preroll_cap;
}

size_t audio_get_preroll(int16_t *out, size_t max_count) {
    if (!g_preroll || !out || g_preroll_len == 0) return 0;
    size_t n = g_preroll_len < max_count ? g_preroll_len : max_count;
    /* 环形缓冲按时间顺序展开：最旧的数据在 head 之后 n 个位置 */
    size_t start = (g_preroll_head + g_preroll_cap - n) % g_preroll_cap;
    for (size_t i = 0; i < n; i++) {
        out[i] = g_preroll[(start + i) % g_preroll_cap];
    }
    return n;
}

void audio_clear_preroll(void) {
    g_preroll_head = 0;
    g_preroll_len = 0;
}

/* ─────────── I2S 初始化 ─────────── */
bool audio_init(void) {
    const pulse_config_t *c = config_get();

    if (!preroll_alloc()) return false;

    memset(&g_stats, 0, sizeof(g_stats));
    g_dc_acc = 0;

    i2s_config_t cfg = {};
    /* PDM 模式必须同时带 MASTER 与 RX 标志 */
    cfg.mode = (i2s_mode_t)(I2S_MODE_MASTER | I2S_MODE_RX | I2S_MODE_PDM);
    cfg.sample_rate = AUDIO_SAMPLE_RATE;
    cfg.bits_per_sample = I2S_BITS_PER_SAMPLE_16BIT;
    /* MSM261D4030H1CPM 的 L/R 脚决定它在哪个相位输出。
     * 单麦 PDM 常规接法为左声道；若录到全 0，改成 ONLY_RIGHT 试。 */
    cfg.channel_format = I2S_CHANNEL_FMT_ONLY_LEFT;
    cfg.communication_format = I2S_COMM_FORMAT_STAND_I2S;
    cfg.intr_alloc_flags = ESP_INTR_FLAG_LEVEL1;
    cfg.dma_buf_count = c->audio.dma_buf_count;
    cfg.dma_buf_len = c->audio.dma_buf_len;
    cfg.use_apll = false;
    cfg.tx_desc_auto_clear = false;
    cfg.fixed_mclk = 0;

    /* queue_size=4 创建事件队列，用 RX_Q_OVF 事件检测真实 overrun。
     * 若已存在（audio_resume 重入）则先释放旧的。 */
    if (g_i2s_evt_queue) {
        i2s_driver_uninstall(I2S_PORT);
        g_i2s_evt_queue = nullptr;
    }
    esp_err_t err = i2s_driver_install(I2S_PORT, &cfg, 4, &g_i2s_evt_queue);
    if (err != ESP_OK) {
        log_e("%s: i2s_driver_install 失败: %s", TAG, esp_err_to_name(err));
        return false;
    }

    /* PDM RX 的引脚含义与标准 I2S 不同，见头文件说明：
     * ws=CLK, data_in=DATA, bck 必须 -1。 */
    i2s_pin_config_t pins = {};
    pins.mck_io_num = I2S_PIN_NO_CHANGE;
    pins.bck_io_num = I2S_PIN_NO_CHANGE;
    pins.ws_io_num = PIN_MIC_CLK;
    pins.data_out_num = I2S_PIN_NO_CHANGE;
    pins.data_in_num = PIN_MIC_DATA;

    err = i2s_set_pin(I2S_PORT, &pins);
    if (err != ESP_OK) {
        log_e("%s: i2s_set_pin 失败: %s", TAG, esp_err_to_name(err));
        i2s_driver_uninstall(I2S_PORT);
        return false;
    }

    /* 下采样倍率。DSR_8S 对应 PDM 时钟 = 采样率 x 64，
     * 是 16kHz 下的常规选择；DSR_16S 会让 PDM 时钟翻倍，
     * 部分麦克风在低时钟下噪声更大，若录音异常可对调试试。 */
    err = i2s_set_pdm_rx_down_sample(I2S_PORT, I2S_PDM_DSR_8S);
    if (err != ESP_OK) {
        log_w("%s: i2s_set_pdm_rx_down_sample 失败: %s", TAG, esp_err_to_name(err));
    }

    i2s_zero_dma_buffer(I2S_PORT);
    err = i2s_start(I2S_PORT);
    if (err != ESP_OK) {
        log_e("%s: i2s_start 失败: %s", TAG, esp_err_to_name(err));
        i2s_driver_uninstall(I2S_PORT);
        return false;
    }

    g_ready = true;
    log_i("%s: I2S0 PDM RX 就绪 CLK=GPIO%d DATA=GPIO%d %uHz/%ubit/%uch DMA %ux%u", TAG,
          PIN_MIC_CLK, PIN_MIC_DATA, AUDIO_SAMPLE_RATE, AUDIO_BITS_PER_SAMPLE, AUDIO_CHANNELS,
          c->audio.dma_buf_count, c->audio.dma_buf_len);
    return true;
}

bool audio_is_ready(void) {
    return g_ready;
}

void audio_stop(void) {
    if (!g_ready) return;
    i2s_stop(I2S_PORT);
    i2s_driver_uninstall(I2S_PORT);
    g_i2s_evt_queue = nullptr; /* 句柄随 uninstall 失效，防止 audio_resume 重入时重复卸载 */
    g_ready = false;
    log_i("%s: I2S 已停止", TAG);
}

bool audio_resume(void) {
    if (g_ready) return true;
    return audio_init();
}

/* ─────────── 读取 ─────────── */
bool audio_read(int16_t *pcm, size_t max_count, size_t *got, uint32_t timeout_ms) {
    if (got) *got = 0;
    if (!g_ready || !pcm || max_count == 0) return false;

    const pulse_config_t *c = config_get();
    size_t bytes_read = 0;
    esp_err_t err = i2s_read(I2S_PORT, pcm, max_count * sizeof(int16_t), &bytes_read,
                             pdMS_TO_TICKS(timeout_ms));
    if (err != ESP_OK) {
        g_stats.read_errors++;
        return false;
    }

    size_t count = bytes_read / sizeof(int16_t);
    if (got) *got = count;
    if (count == 0) return true;

    /* 增益补偿 + 直流滤除。
     * gain_shift 每 +1 约放大 6dB；PDM 麦原始电平常偏低。 */
    int16_t peak = 0;
    int shift = c->audio.gain_shift;
    for (size_t i = 0; i < count; i++) {
        int32_t s = pcm[i];
        if (shift > 0) {
            s <<= shift;
            if (s > 32767) s = 32767;
            if (s < -32768) s = -32768;
        } else if (shift < 0) {
            s >>= (-shift);
        }
        int16_t v = (int16_t)s;
        if (c->audio.dc_removal) v = dc_filter(v);
        pcm[i] = v;
        int16_t a = v < 0 ? (int16_t)-v : v;
        if (a > peak) peak = a;
    }

    g_stats.total_samples += count;
    g_stats.total_bytes += bytes_read;
    g_stats.last_peak = peak;
    g_stats.dc_offset = g_dc_acc >> 8;
    /* 直流偏移超过满量程 3% 视为异常，通常是硬件或 gain 配置问题 */
    g_stats.dc_offset_warning = (g_stats.dc_offset > 1000 || g_stats.dc_offset < -1000);

    /* 统计真实 overrun：legacy 驱动在 RX DMA 队列溢出时往事件队列投
     * I2S_EVENT_RX_Q_OVF。这里非阻塞 drain 队列累加次数。 */
    if (g_i2s_evt_queue) {
        i2s_event_t evt;
        while (xQueueReceive(g_i2s_evt_queue, &evt, 0) == pdTRUE) {
            if (evt.type == I2S_EVENT_RX_Q_OVF) {
                g_stats.overruns++;
            }
        }
    }
    return true;
}

bool audio_read_with_vad(int16_t *pcm, size_t max_count, size_t *got, vad_result_t *vad_out,
                         uint32_t timeout_ms) {
    size_t count = 0;
    if (!audio_read(pcm, max_count, &count, timeout_ms)) return false;
    if (got) *got = count;
    if (count == 0) return true;

    vad_result_t r;
    vad_process(pcm, count, &r);
    g_stats.last_energy = r.last.energy;
    if (vad_out) *vad_out = r;

    /* 静音期持续填充 pre-roll，供语音起点回溯。
     * 语音期不必填 —— 数据已经被写进段里了。 */
    if (r.state == VAD_SILENCE || r.state == VAD_MAYBE_SPEECH) {
        preroll_push(pcm, count);
    }
    return true;
}

void audio_get_stats(audio_stats_t *out) {
    if (out) *out = g_stats;
}

void audio_reset_stats(void) {
    memset(&g_stats, 0, sizeof(g_stats));
}

void audio_dump(void) {
    const pulse_config_t *c = config_get();
    Serial.println();
    Serial.println("================ 音频采集状态 ================");
    Serial.printf("  驱动       legacy I2S (driver/i2s.h)\n");
    Serial.printf("  引脚       CLK=GPIO%d  DATA=GPIO%d\n", PIN_MIC_CLK, PIN_MIC_DATA);
    Serial.printf("  格式       %u Hz / %u bit / %u 声道 = %u B/s\n", AUDIO_SAMPLE_RATE,
                  AUDIO_BITS_PER_SAMPLE, AUDIO_CHANNELS, AUDIO_BYTES_PER_SEC);
    Serial.printf("  DMA        %u 个 x %u 采样点\n", c->audio.dma_buf_count, c->audio.dma_buf_len);
    Serial.printf("  增益       gain_shift=%d (%+d dB)\n", c->audio.gain_shift,
                  c->audio.gain_shift * 6);
    Serial.printf("  直流滤除   %s\n", c->audio.dc_removal ? "开" : "关");
    Serial.printf("  pre-roll   %u ms (%u 采样点, %s)\n", c->vad.preroll_ms,
                  (unsigned)g_preroll_cap,
                  g_preroll ? (heap_caps_get_allocated_size(g_preroll) ? "已分配" : "已分配")
                            : "未分配");
    Serial.println();
    Serial.printf("  累计采样   %llu 点 (%.2f 秒)\n", g_stats.total_samples,
                  (double)g_stats.total_samples / AUDIO_SAMPLE_RATE);
    Serial.printf("  累计字节   %llu (%.2f MB)\n", g_stats.total_bytes,
                  g_stats.total_bytes / 1048576.0);
    Serial.printf("  读取错误   %u\n", g_stats.read_errors);
    Serial.printf("  DMA 积压   %u 次\n", g_stats.overruns);
    Serial.printf("  最近峰值   %d %s\n", g_stats.last_peak,
                  g_stats.last_peak >= 32000 ? "（削波！减小 gain_shift）" : "");
    Serial.printf("  直流偏移   %d %s\n", g_stats.dc_offset,
                  g_stats.dc_offset_warning ? "（偏大，检查 dc_removal 与硬件）" : "");
    Serial.println("=============================================");
}
