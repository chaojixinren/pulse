/**
 * @file pulse_vad.h
 * @brief 轻量 VAD（人声活动检测）：短时能量 + 过零率
 *
 * 为什么不用神经网络 VAD
 * ----------------------
 * WebRTC VAD 或 Silero 这类方案精度更高，但前者需要额外移植、后者要跑推理，
 * 在还要同时驱动 LVGL + SD 写入 + WiFi 上传的 S3 上性价比不高。
 * 能量 + 过零率的组合对「安静房间里的人说话」这个主场景足够用，
 * 计算量只有几十条指令每帧。
 *
 * 判决逻辑
 * --------
 *   能量超阈值 且 过零率落在 [zcr_min, zcr_max] 区间 -> 该帧为语音
 *   连续语音帧累计超过 speech_confirm_ms -> 进入 SPEECH 状态
 *   进入 SPEECH 后，静音持续超过 hangover_ms -> 回到 SILENCE
 *
 * 过零率区间的作用：
 *   过低 -> 直流漂移或低频噪声（空调、电源纹波）
 *   过高 -> 白噪声、摩擦音、爆音
 * 人声浊音段的过零率通常落在中间区域。
 *
 * pre-roll
 * --------
 * 从「检测到语音」到「确认是语音」有 speech_confirm_ms 的延迟，
 * 直接开录会丢掉首字。因此调用方需维护一个 preroll_ms 的环形缓冲，
 * 在 VAD 报告语音起点时把缓冲里的历史数据一并写入。
 * 本模块只负责判决，不管缓冲（见 pulse_audio.cpp）。
 */
#ifndef PULSE_VAD_H
#define PULSE_VAD_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    VAD_SILENCE = 0,
    VAD_MAYBE_SPEECH, /* 超阈值但还未达 confirm 时长 */
    VAD_SPEECH,
    VAD_HANGOVER, /* 语音刚结束，仍在录以免切掉尾音 */
} vad_state_t;

/* 单帧分析结果，便于测试时输出能量曲线来标定阈值。 */
typedef struct {
    uint32_t energy; /* 平均绝对幅度（不是平方和，避免溢出与开销）*/
    uint16_t zcr;    /* 每帧过零次数 */
    int16_t peak;    /* 峰值绝对幅度，用于判断是否削波 */
    bool frame_is_speech;
} vad_frame_t;

typedef struct {
    vad_state_t state;
    /* 状态跳变标志，供调用方触发开段/关段 */
    bool speech_started;
    bool speech_ended;
    uint32_t speech_ms;  /* 当前语音周期累计时长（SPEECH + HANGOVER 持续毫秒数）。
                           * 注意：句间停顿（回到 SILENCE）时会清零，
                           * 但连续说话期间的句间 hangover 会累加。 */
    uint32_t silence_ms; /* 当前静音已持续时长（SILENCE 或 HANGOVER 中的静音累计） */
    vad_frame_t last;
} vad_result_t;

/** 复位 VAD 状态机（开始录音或配置变更时调用）*/
void vad_reset(void);

/**
 * 分析一帧 PCM。
 * @param pcm    16bit 单声道采样
 * @param count  采样点数（非字节数）
 * @param out    输出状态。out->speech_started / speech_ended 为一次性脉冲
 * @return true 表示当前应当录音（SPEECH 或 HANGOVER 状态）
 */
bool vad_process(const int16_t *pcm, size_t count, vad_result_t *out);

/** 当前状态 */
vad_state_t vad_state(void);

const char *vad_state_str(vad_state_t s);

/**
 * 自动标定建议：统计一段时间的能量分布，给出推荐阈值。
 * 用法见 M5 测试 —— 先安静 5 秒，再说话 5 秒，脚本会给出建议值。
 */
void vad_calib_reset(void);
void vad_calib_feed(const vad_frame_t *f, bool is_speech_period);
void vad_calib_report(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_VAD_H */
