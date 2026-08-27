/**
 * @file pulse_vad.cpp
 * @brief VAD 实现：短时能量 + 过零率
 */
#include "pulse_vad.h"

#include <Arduino.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "vad";

static vad_state_t g_state = VAD_SILENCE;
static uint32_t g_speech_ms = 0;
static uint32_t g_silence_ms = 0;
static uint32_t g_maybe_ms = 0; /* MAYBE_SPEECH 状态累计时长 */

void vad_reset(void) {
    g_state = VAD_SILENCE;
    g_speech_ms = 0;
    g_silence_ms = 0;
    g_maybe_ms = 0;
}

vad_state_t vad_state(void) {
    return g_state;
}

const char *vad_state_str(vad_state_t s) {
    switch (s) {
        case VAD_SILENCE: return "静音";
        case VAD_MAYBE_SPEECH: return "疑似语音";
        case VAD_SPEECH: return "语音";
        case VAD_HANGOVER: return "尾音保持";
        default: return "未知";
    }
}

/* 单帧特征提取。
 * 用平均绝对幅度而非能量平方和：避免 int32 溢出，且省掉乘法，
 * 在 16kHz 下每帧几百个采样点，这点开销可以忽略。 */
static void analyze_frame(const int16_t *pcm, size_t count, vad_frame_t *f) {
    uint64_t sum_abs = 0;
    uint16_t zcr = 0;
    int16_t peak = 0;
    int prev_sign = 0;

    for (size_t i = 0; i < count; i++) {
        int16_t s = pcm[i];
        int32_t a = s < 0 ? -(int32_t)s : (int32_t)s;
        sum_abs += (uint64_t)a;
        if (a > peak) peak = (int16_t)a;

        /* 过零统计：只在符号真正翻转时计数，0 不算翻转 */
        int sign = (s > 0) ? 1 : (s < 0 ? -1 : 0);
        if (sign != 0) {
            if (prev_sign != 0 && sign != prev_sign) zcr++;
            prev_sign = sign;
        }
    }

    f->energy = count ? (uint32_t)(sum_abs / count) : 0;
    f->zcr = zcr;
    f->peak = peak;
    f->frame_is_speech = false;
}

bool vad_process(const int16_t *pcm, size_t count, vad_result_t *out) {
    const pulse_config_t *c = config_get();
    const cfg_vad_t *v = &c->vad;

    vad_frame_t f;
    analyze_frame(pcm, count, &f);

    /* 本帧时长（毫秒）。采样率固定 16kHz。 */
    uint32_t frame_ms = (uint32_t)((uint64_t)count * 1000 / AUDIO_SAMPLE_RATE);
    if (frame_ms == 0) frame_ms = 1;

    /* VAD 关闭时始终录音，等价于连续录制。 */
    if (!v->enabled) {
        g_state = VAD_SPEECH;
        f.frame_is_speech = true;
        if (out) {
            out->state = g_state;
            out->speech_started = false;
            out->speech_ended = false;
            out->speech_ms = (g_speech_ms += frame_ms);
            out->silence_ms = 0;
            out->last = f;
        }
        return true;
    }

    /* 判决：能量与过零率必须同时满足。
     * zcr 上下界用于区分人声与噪声，理由见头文件。 */
    bool loud = f.energy >= v->energy_threshold;
    bool voiced = (f.zcr >= v->zcr_min) && (f.zcr <= v->zcr_max);
    f.frame_is_speech = loud && voiced;

    bool started = false;
    bool ended = false;

    switch (g_state) {
        case VAD_SILENCE:
            if (f.frame_is_speech) {
                g_state = VAD_MAYBE_SPEECH;
                g_maybe_ms = frame_ms;
                /* confirm 时长为 0 时立刻确认 */
                if (g_maybe_ms >= v->speech_confirm_ms) {
                    g_state = VAD_SPEECH;
                    g_speech_ms = g_maybe_ms;
                    started = true;
                }
            } else {
                g_silence_ms += frame_ms;
            }
            break;

        case VAD_MAYBE_SPEECH:
            if (f.frame_is_speech) {
                g_maybe_ms += frame_ms;
                if (g_maybe_ms >= v->speech_confirm_ms) {
                    g_state = VAD_SPEECH;
                    g_speech_ms = g_maybe_ms;
                    g_silence_ms = 0;
                    started = true;
                }
            } else {
                /* 未达确认时长就消失，判为瞬时杂音，回到静音 */
                g_state = VAD_SILENCE;
                g_maybe_ms = 0;
                g_silence_ms += frame_ms;
            }
            break;

        case VAD_SPEECH:
            g_speech_ms += frame_ms;
            if (f.frame_is_speech) {
                g_silence_ms = 0;
            } else {
                g_silence_ms = frame_ms;
                g_state = VAD_HANGOVER;
            }
            break;

        case VAD_HANGOVER:
            g_speech_ms += frame_ms;
            if (f.frame_is_speech) {
                /* 尾音期内又有声音，说明只是句间停顿，继续录 */
                g_state = VAD_SPEECH;
                g_silence_ms = 0;
            } else {
                g_silence_ms += frame_ms;
                if (g_silence_ms >= v->hangover_ms) {
                    g_state = VAD_SILENCE;
                    ended = true;
                    g_maybe_ms = 0;
                }
            }
            break;
    }

    if (out) {
        out->state = g_state;
        out->speech_started = started;
        out->speech_ended = ended;
        out->speech_ms = g_speech_ms;
        out->silence_ms = g_silence_ms;
        out->last = f;
    }

    if (ended) g_speech_ms = 0;

    /* SPEECH 与 HANGOVER 都要录：后者是为了保住尾音 */
    return g_state == VAD_SPEECH || g_state == VAD_HANGOVER;
}

/* ─────────── 阈值自动标定 ───────────
 * 分别统计「安静期」与「说话期」的能量与过零率分布，
 * 给出建议阈值。避免手工试错。 */
typedef struct {
    uint64_t energy_sum;
    uint32_t energy_min;
    uint32_t energy_max;
    uint64_t zcr_sum;
    uint16_t zcr_min;
    uint16_t zcr_max;
    uint32_t frames;
    int16_t peak;
} calib_bucket_t;

static calib_bucket_t g_quiet, g_speech;

static void bucket_reset(calib_bucket_t *b) {
    b->energy_sum = 0;
    b->energy_min = 0xFFFFFFFF;
    b->energy_max = 0;
    b->zcr_sum = 0;
    b->zcr_min = 0xFFFF;
    b->zcr_max = 0;
    b->frames = 0;
    b->peak = 0;
}

void vad_calib_reset(void) {
    bucket_reset(&g_quiet);
    bucket_reset(&g_speech);
}

void vad_calib_feed(const vad_frame_t *f, bool is_speech_period) {
    if (!f) return;
    calib_bucket_t *b = is_speech_period ? &g_speech : &g_quiet;
    b->energy_sum += f->energy;
    if (f->energy < b->energy_min) b->energy_min = f->energy;
    if (f->energy > b->energy_max) b->energy_max = f->energy;
    b->zcr_sum += f->zcr;
    if (f->zcr < b->zcr_min) b->zcr_min = f->zcr;
    if (f->zcr > b->zcr_max) b->zcr_max = f->zcr;
    if (f->peak > b->peak) b->peak = f->peak;
    b->frames++;
}

static void bucket_report(const char *name, const calib_bucket_t *b) {
    if (b->frames == 0) {
        Serial.printf("  %s: 无数据\n", name);
        return;
    }
    Serial.printf("  %s (%u 帧):\n", name, b->frames);
    Serial.printf("    能量  平均 %u  范围 %u ~ %u\n", (unsigned)(b->energy_sum / b->frames),
                  b->energy_min, b->energy_max);
    Serial.printf("    过零率 平均 %u  范围 %u ~ %u\n", (unsigned)(b->zcr_sum / b->frames),
                  b->zcr_min, b->zcr_max);
    Serial.printf("    峰值  %d %s\n", b->peak,
                  b->peak >= 32000 ? "（接近满量程，可能削波，建议减小 gain_shift）" : "");
}

void vad_calib_report(void) {
    Serial.println();
    Serial.println("============== VAD 阈值标定报告 ==============");
    bucket_report("安静期", &g_quiet);
    bucket_report("说话期", &g_speech);

    if (g_quiet.frames == 0 || g_speech.frames == 0) {
        Serial.println("  数据不足，无法给出建议");
        Serial.println("===========================================");
        return;
    }

    uint32_t quiet_avg = (uint32_t)(g_quiet.energy_sum / g_quiet.frames);
    uint32_t speech_avg = (uint32_t)(g_speech.energy_sum / g_speech.frames);

    Serial.println();
    if (speech_avg <= quiet_avg * 2) {
        Serial.println("  警告：说话与安静的能量差距不足 2 倍，VAD 无法可靠区分。");
        Serial.println("        可能原因：麦克风增益太低（调大 audio.gain_shift）、");
        Serial.println("        环境噪声过大、或麦克风未正常工作。");
    } else {
        /* 阈值取安静上界与说话均值的几何中点偏保守侧：
         * 宁可漏掉一点点弱语音，也不要把噪声当语音一直录。 */
        uint32_t suggest = g_quiet.energy_max + (speech_avg - g_quiet.energy_max) / 3;
        if (suggest <= g_quiet.energy_max) suggest = g_quiet.energy_max + 1;

        Serial.println("  建议写入 config.json 的 vad 段：");
        Serial.printf("    \"energy_threshold\": %u,\n", suggest);
        /* 过零率区间放宽 30% 余量，避免临界抖动 */
        uint16_t zmin = g_speech.zcr_min > 3 ? (uint16_t)(g_speech.zcr_min * 0.7) : 1;
        uint16_t zmax = (uint16_t)(g_speech.zcr_max * 1.3);
        Serial.printf("    \"zcr_min\": %u,\n", zmin);
        Serial.printf("    \"zcr_max\": %u\n", zmax);
        Serial.printf("  （安静上界 %u，说话均值 %u，信噪比约 %.1f 倍）\n", g_quiet.energy_max,
                      speech_avg, (double)speech_avg / (quiet_avg ? quiet_avg : 1));
    }
    Serial.println("===========================================");
}
