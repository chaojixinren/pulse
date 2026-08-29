/**
 * @file pulse_storage.cpp
 * @brief 存储层实现：SD spool 队列 + PSRAM 降级缓冲
 */
#include "pulse_storage.h"

#include <Arduino.h>
#include <FS.h>
#include <SD_MMC.h>
#include <esp_heap_caps.h>

#include <string.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "stor";

/* WAV 头固定 44 字节。后端 audio.go 会校验 RIFF 魔数，
 * 因此每段必须是自带完整头的 WAV，不能是裸 PCM。 */
#define WAV_HEADER_SIZE 44

/* 段文件名前缀。.part 表示正在写入，关闭时才改名为 .wav，
 * 这样上传任务永远不会读到写了一半的文件。 */
#define SEG_EXT_PARTIAL ".part"
#define SEG_EXT_DONE ".wav"

static storage_backend_t g_backend = STORAGE_BACKEND_NONE;

/* ─────────── PSRAM 降级缓冲 ───────────
 * 简单的单段暂存：无卡时至少能让 M5/M8 联调跑通。
 * 只保留最近 N 段，满了丢最旧的。 */
#define PSRAM_MAX_SEGS 8
typedef struct {
    uint8_t *data;
    size_t size;
    uint32_t recorded_at;
    bool used;
} psram_seg_t;

static psram_seg_t g_psram_segs[PSRAM_MAX_SEGS];
static size_t g_psram_seg_cap = 0; /* 单段容量上限 */

/* ─────────── 段句柄 ─────────── */
struct storage_segment {
    bool is_sd;
    /* SD 模式 */
    File file;
    char path_partial[96];
    char path_done[96];
    /* PSRAM 模式 */
    int psram_idx;
    /* 公共 */
    size_t pcm_bytes;
    uint32_t recorded_at;
};

static storage_segment g_seg; /* 同一时刻只允许一个段在写 */
static bool g_seg_busy = false;

/* ─────────── WAV 头 ───────────
 * 采样参数由后端 stt.go 写死（16kHz/16bit/mono），不可配置。 */
static void wav_header_build(uint8_t *hdr, uint32_t pcm_bytes) {
    const uint32_t sample_rate = AUDIO_SAMPLE_RATE;
    const uint16_t channels = AUDIO_CHANNELS;
    const uint16_t bits = AUDIO_BITS_PER_SAMPLE;
    const uint32_t byte_rate = sample_rate * channels * (bits / 8);
    const uint16_t block_align = (uint16_t)(channels * (bits / 8));

    memcpy(hdr + 0, "RIFF", 4);
    /* RIFF chunk size = 文件总长 - 8 */
    uint32_t riff_size = pcm_bytes + WAV_HEADER_SIZE - 8;
    memcpy(hdr + 4, &riff_size, 4);
    memcpy(hdr + 8, "WAVE", 4);

    memcpy(hdr + 12, "fmt ", 4);
    uint32_t fmt_size = 16;
    memcpy(hdr + 16, &fmt_size, 4);
    uint16_t audio_fmt = 1; /* PCM */
    memcpy(hdr + 20, &audio_fmt, 2);
    memcpy(hdr + 22, &channels, 2);
    memcpy(hdr + 24, &sample_rate, 4);
    memcpy(hdr + 28, &byte_rate, 4);
    memcpy(hdr + 32, &block_align, 2);
    memcpy(hdr + 34, &bits, 2);

    memcpy(hdr + 36, "data", 4);
    memcpy(hdr + 40, &pcm_bytes, 4);
}

/* ─────────── 目录准备 ─────────── */
static bool ensure_dir(const char *path) {
    if (SD_MMC.exists(path)) return true;
    /* 逐级创建：/rec/pending 需要先有 /rec */
    char tmp[96];
    strncpy(tmp, path, sizeof(tmp) - 1);
    tmp[sizeof(tmp) - 1] = '\0';
    for (char *p = tmp + 1; *p; p++) {
        if (*p == '/') {
            *p = '\0';
            if (!SD_MMC.exists(tmp)) SD_MMC.mkdir(tmp);
            *p = '/';
        }
    }
    if (!SD_MMC.exists(tmp)) SD_MMC.mkdir(tmp);
    return SD_MMC.exists(path);
}

/* ─────────── SD 挂载 ─────────── */
static bool try_mount_sd(void) {
    const pulse_config_t *c = config_get();

    if (!SD_MMC.setPins(PIN_SD_CLK, PIN_SD_CMD, PIN_SD_D0, PIN_SD_D1, PIN_SD_D2, PIN_SD_D3)) {
        log_e("%s: SD setPins 失败", TAG);
        return false;
    }

    int freq_khz = c->storage.sd_freq_mhz * 1000;

    /* 先按配置尝试。sd_use_1bit=true 时直接走 1-bit。 */
    if (!c->storage.sd_use_1bit) {
        if (SD_MMC.begin("/sdcard", false, false, freq_khz)) {
            g_backend = STORAGE_BACKEND_SD_4BIT;
            log_i("%s: SD 4-bit @ %dMHz 挂载成功", TAG, c->storage.sd_freq_mhz);
            return true;
        }
        log_w("%s: SD 4-bit 挂载失败，尝试 1-bit", TAG);
        SD_MMC.end();
    }

    if (SD_MMC.begin("/sdcard", true, false, freq_khz)) {
        g_backend = STORAGE_BACKEND_SD_1BIT;
        log_i("%s: SD 1-bit @ %dMHz 挂载成功", TAG, c->storage.sd_freq_mhz);
        return true;
    }

    /* 再退一档频率：GPIO45 上拉导致的时序问题有时降频能救。 */
    SD_MMC.end();
    if (SD_MMC.begin("/sdcard", true, false, 4000)) {
        g_backend = STORAGE_BACKEND_SD_1BIT;
        log_w("%s: SD 1-bit @ 4MHz 挂载成功（已降频）", TAG);
        return true;
    }

    log_e("%s: SD 挂载彻底失败", TAG);
    SD_MMC.end();
    return false;
}

/* ─────────── PSRAM 降级 ─────────── */
static bool init_psram_fallback(void) {
    const pulse_config_t *c = config_get();
    if (!c->storage.fallback_to_psram) {
        log_e("%s: SD 不可用且 fallback_to_psram=false", TAG);
        return false;
    }

    size_t total = (size_t)c->storage.psram_buffer_kb * 1024;
    g_psram_seg_cap = total / PSRAM_MAX_SEGS;
    /* 单段至少要能装下一个完整分段，否则录音会被频繁截断。 */
    if (g_psram_seg_cap < 64 * 1024) g_psram_seg_cap = 64 * 1024;

    int ok = 0;
    for (int i = 0; i < PSRAM_MAX_SEGS; i++) {
        g_psram_segs[i].data = (uint8_t *)heap_caps_malloc(g_psram_seg_cap, MALLOC_CAP_SPIRAM);
        g_psram_segs[i].size = 0;
        g_psram_segs[i].used = false;
        if (g_psram_segs[i].data) ok++;
    }

    if (ok == 0) {
        log_e("%s: PSRAM 缓冲分配失败", TAG);
        return false;
    }

    g_backend = STORAGE_BACKEND_PSRAM;
    log_w("%s: 降级到 PSRAM 缓冲 %d 段 x %uKB（掉电丢失，仅供联调）", TAG, ok,
          (unsigned)(g_psram_seg_cap / 1024));
    return true;
}

bool storage_init(void) {
    if (try_mount_sd()) {
        /* SD 已挂载。此时是用「默认参数」挂载的 —— 挂载前读不到 config.json
         * （鸡生蛋：config 读文件需要 SD，SD 挂载需要 config 里的 sd_freq）。
         * 现在补一次全量配置加载（默认 → config.json → NVS），让文件配置生效。
         *
         * 不因 sd_freq/sd_use_1bit 变化而 remount：默认 20MHz 已成功挂载，
         * config 里改频率的唯一合理动机是「降频解决兼容性」，既然默认值已
         * 挂上就无需再动；若默认值挂不上，try_mount_sd 内部已自动降级。 */
        config_load();
        const pulse_config_t *c = config_get(); /* 重新获取，可能含文件配置 */

        /* 建立 spool 目录结构 */
        if (!ensure_dir(c->storage.spool_dir)) {
            log_e("%s: 创建 %s 失败", TAG, c->storage.spool_dir);
        }
        if (!ensure_dir(c->storage.failed_dir)) {
            log_e("%s: 创建 %s 失败", TAG, c->storage.failed_dir);
        }
        /* 清理上次异常断电留下的 .part 残file */
        File dir = SD_MMC.open(c->storage.spool_dir);
        if (dir && dir.isDirectory()) {
            File f;
            char victims[8][96];
            int nv = 0;
            while ((f = dir.openNextFile()) && nv < 8) {
                const char *n = f.name();
                size_t len = strlen(n);
                if (len > 5 && strcmp(n + len - 5, SEG_EXT_PARTIAL) == 0) {
                    snprintf(victims[nv], sizeof(victims[nv]), "%s/%s", c->storage.spool_dir, n);
                    nv++;
                }
                f.close();
            }
            dir.close();
            for (int i = 0; i < nv; i++) {
                SD_MMC.remove(victims[i]);
                log_w("%s: 清理残留 %s", TAG, victims[i]);
            }
        }
        return true;
    }

    return init_psram_fallback();
}

storage_backend_t storage_backend(void) {
    return g_backend;
}

const char *storage_backend_str(storage_backend_t b) {
    switch (b) {
        case STORAGE_BACKEND_SD_4BIT: return "SD 4-bit";
        case STORAGE_BACKEND_SD_1BIT: return "SD 1-bit";
        case STORAGE_BACKEND_PSRAM: return "PSRAM 降级";
        default: return "不可用";
    }
}

/* ─────────── 容量查询 ─────────── */
bool storage_stat(storage_info_t *out) {
    if (!out) return false;
    const pulse_config_t *c = config_get();
    memset(out, 0, sizeof(*out));
    out->backend = g_backend;

    if (g_backend == STORAGE_BACKEND_PSRAM) {
        out->total_bytes = (uint64_t)g_psram_seg_cap * PSRAM_MAX_SEGS;
        for (int i = 0; i < PSRAM_MAX_SEGS; i++) {
            if (g_psram_segs[i].used) {
                out->used_bytes += g_psram_segs[i].size;
                out->pending_count++;
                out->pending_bytes += g_psram_segs[i].size;
            }
        }
        out->free_bytes = out->total_bytes - out->used_bytes;
        out->writable = out->free_bytes > g_psram_seg_cap;
        return true;
    }

    if (g_backend == STORAGE_BACKEND_NONE) return false;

    out->total_bytes = SD_MMC.totalBytes();
    out->used_bytes = SD_MMC.usedBytes();
    out->free_bytes = out->total_bytes - out->used_bytes;

    /* 统计待上传队列 */
    File dir = SD_MMC.open(c->storage.spool_dir);
    if (dir && dir.isDirectory()) {
        File f;
        while ((f = dir.openNextFile())) {
            const char *n = f.name();
            size_t len = strlen(n);
            if (len > 4 && strcmp(n + len - 4, SEG_EXT_DONE) == 0) {
                out->pending_count++;
                out->pending_bytes += f.size();
            }
            f.close();
        }
        dir.close();
    }

    out->writable = out->free_bytes > (uint64_t)c->storage.min_free_mb * 1024 * 1024;
    return true;
}

/* ─────────── 段写入 ─────────── */
storage_segment_t *storage_segment_open(uint32_t recorded_at_unix) {
    const pulse_config_t *c = config_get();

    if (g_backend == STORAGE_BACKEND_NONE) return nullptr;
    if (g_seg_busy) {
        log_e("%s: 已有段在写入，不支持并发", TAG);
        return nullptr;
    }

    memset(&g_seg, 0, sizeof(g_seg));
    g_seg.recorded_at = recorded_at_unix;
    g_seg.pcm_bytes = 0;

    if (g_backend == STORAGE_BACKEND_PSRAM) {
        int slot = -1;
        for (int i = 0; i < PSRAM_MAX_SEGS; i++) {
            if (g_psram_segs[i].data && !g_psram_segs[i].used) { slot = i; break; }
        }
        if (slot < 0) {
            /* 满了：丢最旧的（recorded_at 最小） */
            uint32_t oldest = 0xFFFFFFFF;
            for (int i = 0; i < PSRAM_MAX_SEGS; i++) {
                if (g_psram_segs[i].data && g_psram_segs[i].recorded_at < oldest) {
                    oldest = g_psram_segs[i].recorded_at;
                    slot = i;
                }
            }
            if (slot < 0) return nullptr;
            log_w("%s: PSRAM 缓冲已满，丢弃最旧段", TAG);
        }
        g_seg.is_sd = false;
        g_seg.psram_idx = slot;
        g_psram_segs[slot].size = 0;
        g_psram_segs[slot].recorded_at = recorded_at_unix;
        g_seg_busy = true;
        return &g_seg;
    }

    /* SD 模式：先检查空间 */
    storage_info_t info;
    if (storage_stat(&info) && !info.writable) {
        /* 尝试淘汰旧段腾空间 */
        storage_evict_oldest((uint64_t)c->audio.segment_max_bytes);
        if (storage_stat(&info) && !info.writable) {
            log_e("%s: 剩余空间不足 %uMB，停止录音", TAG, c->storage.min_free_mb);
            return nullptr;
        }
    }

    /* 文件名用 unix 时间戳，天然有序，便于按时间淘汰与排队上传 */
    snprintf(g_seg.path_partial, sizeof(g_seg.path_partial), "%s/%010u%s", c->storage.spool_dir,
             recorded_at_unix, SEG_EXT_PARTIAL);
    snprintf(g_seg.path_done, sizeof(g_seg.path_done), "%s/%010u%s", c->storage.spool_dir,
             recorded_at_unix, SEG_EXT_DONE);

    g_seg.file = SD_MMC.open(g_seg.path_partial, FILE_WRITE);
    if (!g_seg.file) {
        log_e("%s: 创建 %s 失败", TAG, g_seg.path_partial);
        return nullptr;
    }

    /* 先占位写 44 字节头，关闭时回填真实长度。
     * 这样即使断电，文件也是可识别的 WAV（只是长度字段偏小）。 */
    uint8_t hdr[WAV_HEADER_SIZE];
    wav_header_build(hdr, 0);
    if (g_seg.file.write(hdr, WAV_HEADER_SIZE) != WAV_HEADER_SIZE) {
        log_e("%s: 写 WAV 头失败", TAG);
        g_seg.file.close();
        SD_MMC.remove(g_seg.path_partial);
        return nullptr;
    }

    g_seg.is_sd = true;
    g_seg_busy = true;
    return &g_seg;
}

size_t storage_segment_write(storage_segment_t *seg, const void *data, size_t len) {
    if (!seg || !data || len == 0) return 0;

    if (!seg->is_sd) {
        psram_seg_t *ps = &g_psram_segs[seg->psram_idx];
        size_t room = g_psram_seg_cap - ps->size;
        size_t n = len < room ? len : room;
        if (n > 0) {
            memcpy(ps->data + ps->size, data, n);
            ps->size += n;
            seg->pcm_bytes += n;
        }
        return n;
    }

    size_t n = seg->file.write((const uint8_t *)data, len);
    seg->pcm_bytes += n;
    return n;
}

size_t storage_segment_size(const storage_segment_t *seg) {
    return seg ? seg->pcm_bytes : 0;
}

bool storage_segment_close(storage_segment_t *seg, bool keep) {
    if (!seg) return false;
    bool ok = true;

    if (!seg->is_sd) {
        psram_seg_t *ps = &g_psram_segs[seg->psram_idx];
        ps->used = keep && ps->size > 0;
        if (!ps->used) ps->size = 0;
        g_seg_busy = false;
        return true;
    }

    /* 回填 WAV 头的真实长度 */
    if (keep && seg->pcm_bytes > 0) {
        uint8_t hdr[WAV_HEADER_SIZE];
        wav_header_build(hdr, (uint32_t)seg->pcm_bytes);
        if (!seg->file.seek(0)) {
            log_e("%s: seek(0) 失败，无法回填 WAV 头", TAG);
            ok = false;
        } else if (seg->file.write(hdr, WAV_HEADER_SIZE) != WAV_HEADER_SIZE) {
            log_e("%s: 回填 WAV 头失败", TAG);
            ok = false;
        }
    }
    seg->file.close();

    if (!keep || seg->pcm_bytes == 0) {
        SD_MMC.remove(seg->path_partial);
        g_seg_busy = false;
        return true;
    }

    if (ok) {
        /* .part → .wav：改名成功后上传任务才会看到它 */
        if (!SD_MMC.rename(seg->path_partial, seg->path_done)) {
            log_e("%s: 重命名 %s 失败", TAG, seg->path_partial);
            ok = false;
        }
    }
    if (!ok) SD_MMC.remove(seg->path_partial);

    g_seg_busy = false;
    return ok;
}

/* ─────────── 队列遍历 ─────────── */
bool storage_pending_at(int idx, char *path_out, size_t path_cap, size_t *size_out) {
    const pulse_config_t *c = config_get();
    if (!path_out || idx < 0) return false;

    if (g_backend == STORAGE_BACKEND_PSRAM) {
        int seen = 0;
        for (int i = 0; i < PSRAM_MAX_SEGS; i++) {
            if (!g_psram_segs[i].used) continue;
            if (seen == idx) {
                snprintf(path_out, path_cap, "psram:%d", i);
                if (size_out) *size_out = g_psram_segs[i].size;
                return true;
            }
            seen++;
        }
        return false;
    }

    if (g_backend == STORAGE_BACKEND_NONE) return false;

    File dir = SD_MMC.open(c->storage.spool_dir);
    if (!dir || !dir.isDirectory()) return false;

    int seen = 0;
    File f;
    bool found = false;
    while ((f = dir.openNextFile())) {
        const char *n = f.name();
        size_t len = strlen(n);
        if (len > 4 && strcmp(n + len - 4, SEG_EXT_DONE) == 0) {
            if (seen == idx) {
                snprintf(path_out, path_cap, "%s/%s", c->storage.spool_dir, n);
                if (size_out) *size_out = f.size();
                found = true;
                f.close();
                break;
            }
            seen++;
        }
        f.close();
    }
    dir.close();
    return found;
}

int storage_pending_count(void) {
    storage_info_t info;
    if (!storage_stat(&info)) return 0;
    return (int)info.pending_count;
}

size_t storage_read_file(const char *path, void *buf, size_t cap) {
    if (!path || !buf) return 0;

    if (strncmp(path, "psram:", 6) == 0) {
        int i = atoi(path + 6);
        if (i < 0 || i >= PSRAM_MAX_SEGS || !g_psram_segs[i].used) return 0;
        /* PSRAM 模式存的是裸 PCM，读出时补上 WAV 头 */
        size_t pcm = g_psram_segs[i].size;
        if (cap < pcm + WAV_HEADER_SIZE) return 0;
        wav_header_build((uint8_t *)buf, (uint32_t)pcm);
        memcpy((uint8_t *)buf + WAV_HEADER_SIZE, g_psram_segs[i].data, pcm);
        return pcm + WAV_HEADER_SIZE;
    }

    File f = SD_MMC.open(path, FILE_READ);
    if (!f) return 0;
    size_t n = f.read((uint8_t *)buf, cap);
    f.close();
    return n;
}

size_t storage_read_chunk(const char *path, size_t offset, void *buf, size_t len) {
    if (!path || !buf || len == 0) return 0;

    /* PSRAM 模式：文件由「44 字节 WAV 头 + 裸 PCM」拼成，
     * 头是读取时即时生成的，所以要按偏移分别处理两段。 */
    if (strncmp(path, "psram:", 6) == 0) {
        int i = atoi(path + 6);
        if (i < 0 || i >= PSRAM_MAX_SEGS || !g_psram_segs[i].used) return 0;

        size_t pcm = g_psram_segs[i].size;
        size_t total = pcm + WAV_HEADER_SIZE;
        if (offset >= total) return 0;

        size_t n = 0;
        uint8_t *out = (uint8_t *)buf;

        /* 落在头部的部分 */
        if (offset < WAV_HEADER_SIZE) {
            uint8_t hdr[WAV_HEADER_SIZE];
            wav_header_build(hdr, (uint32_t)pcm);
            size_t from_hdr = WAV_HEADER_SIZE - offset;
            if (from_hdr > len) from_hdr = len;
            memcpy(out, hdr + offset, from_hdr);
            n += from_hdr;
            offset += from_hdr;
        }

        /* 落在 PCM 的部分 */
        if (n < len && offset >= WAV_HEADER_SIZE) {
            size_t pcm_off = offset - WAV_HEADER_SIZE;
            if (pcm_off < pcm) {
                size_t from_pcm = pcm - pcm_off;
                if (from_pcm > len - n) from_pcm = len - n;
                memcpy(out + n, g_psram_segs[i].data + pcm_off, from_pcm);
                n += from_pcm;
            }
        }
        return n;
    }

    File f = SD_MMC.open(path, FILE_READ);
    if (!f) return 0;
    if (!f.seek(offset)) {
        f.close();
        return 0;
    }
    size_t n = f.read((uint8_t *)buf, len);
    f.close();
    return n;
}

size_t storage_file_size(const char *path) {
    if (!path) return 0;

    if (strncmp(path, "psram:", 6) == 0) {
        int i = atoi(path + 6);
        if (i < 0 || i >= PSRAM_MAX_SEGS || !g_psram_segs[i].used) return 0;
        return g_psram_segs[i].size + WAV_HEADER_SIZE;
    }

    File f = SD_MMC.open(path, FILE_READ);
    if (!f) return 0;
    size_t n = f.size();
    f.close();
    return n;
}

bool storage_delete(const char *path) {
    if (!path) return false;
    if (strncmp(path, "psram:", 6) == 0) {
        int i = atoi(path + 6);
        if (i < 0 || i >= PSRAM_MAX_SEGS) return false;
        g_psram_segs[i].used = false;
        g_psram_segs[i].size = 0;
        return true;
    }
    return SD_MMC.remove(path);
}

bool storage_move_to_failed(const char *path) {
    const pulse_config_t *c = config_get();
    if (!path) return false;
    if (strncmp(path, "psram:", 6) == 0) {
        /* PSRAM 模式没有 failed 区，直接丢弃 */
        return storage_delete(path);
    }
    const char *base = strrchr(path, '/');
    base = base ? base + 1 : path;
    char dst[96];
    snprintf(dst, sizeof(dst), "%s/%s", c->storage.failed_dir, base);
    if (SD_MMC.exists(dst)) SD_MMC.remove(dst);
    return SD_MMC.rename(path, dst);
}

uint64_t storage_evict_oldest(uint64_t need_bytes) {
    const pulse_config_t *c = config_get();
    if (g_backend != STORAGE_BACKEND_SD_4BIT && g_backend != STORAGE_BACKEND_SD_1BIT) return 0;

    uint64_t freed = 0;
    /* 文件名是 10 位零填充的时间戳，字典序即时间序，
     * 每轮找出当前最旧的一个删掉，直到腾出足够空间。 */
    while (freed < need_bytes) {
        File dir = SD_MMC.open(c->storage.spool_dir);
        if (!dir || !dir.isDirectory()) break;

        char oldest[96] = {0};
        size_t oldest_size = 0;
        File f;
        while ((f = dir.openNextFile())) {
            const char *n = f.name();
            size_t len = strlen(n);
            if (len > 4 && strcmp(n + len - 4, SEG_EXT_DONE) == 0) {
                if (oldest[0] == '\0' || strcmp(n, oldest) < 0) {
                    strncpy(oldest, n, sizeof(oldest) - 1);
                    oldest_size = f.size();
                }
            }
            f.close();
        }
        dir.close();

        if (oldest[0] == '\0') break;

        char full[96];
        snprintf(full, sizeof(full), "%s/%s", c->storage.spool_dir, oldest);
        if (!SD_MMC.remove(full)) break;
        freed += oldest_size;
        log_w("%s: 空间不足，淘汰最旧段 %s (%u 字节)", TAG, oldest, (unsigned)oldest_size);
    }
    return freed;
}

void storage_dump(void) {
    storage_info_t info;
    Serial.println();
    Serial.println("================ 存储状态 ================");

    if (!storage_stat(&info)) {
        Serial.println("  后端不可用，无法录音");
        Serial.println("=========================================");
        return;
    }

    Serial.printf("  后端       %s\n", storage_backend_str(info.backend));
    if (info.backend == STORAGE_BACKEND_PSRAM) {
        Serial.println("  警告：PSRAM 降级模式，掉电数据丢失，容量极小");
        Serial.printf("  容量       %.2f MB (单段上限 %uKB x %d)\n",
                      info.total_bytes / 1048576.0, (unsigned)(g_psram_seg_cap / 1024),
                      PSRAM_MAX_SEGS);
    } else {
        Serial.printf("  卡容量     %.2f GB\n", info.total_bytes / (1024.0 * 1024 * 1024));
        Serial.printf("  已用       %.2f MB\n", info.used_bytes / 1048576.0);
    }
    Serial.printf("  剩余       %.2f MB\n", info.free_bytes / 1048576.0);
    Serial.printf("  可写       %s\n", info.writable ? "是" : "否（空间不足）");
    Serial.printf("  待上传     %u 段, %.2f MB\n", info.pending_count,
                  info.pending_bytes / 1048576.0);

    /* 按当前码率估算可录时长，便于判断容量是否够用 */
    double hours = info.free_bytes / (double)AUDIO_BYTES_PER_SEC / 3600.0;
    Serial.printf("  可录时长   约 %.1f 小时（%u B/s，未计 VAD 静音节省）\n", hours,
                  AUDIO_BYTES_PER_SEC);
    Serial.println("=========================================");
}
