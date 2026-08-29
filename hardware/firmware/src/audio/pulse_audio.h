/**
 * @file pulse_audio.h
 * @brief 音频采集 HAL：I2S0 PDM RX (MSM261D4030H1CPM)
 *
 * 驱动选型（重要）
 * ----------------
 * 必须用 **legacy I2S 驱动**（driver/i2s.h）。原因：
 * 你的工具链是 Arduino-ESP32 core 2.0.17 / IDF 4.4.7，其中
 * driver/i2s_pdm.h（新驱动）**不存在**，网上大量 PDM 例程用的
 * `I2S.setPinsPdmRx()` 是 core 3.x API，在这里编译不过。
 * platformio.ini 已锁 espressif32@^6.5.0 防止误升级。
 *
 * PDM 引脚映射的坑
 * ----------------
 * PDM RX 模式下 i2s_pin_config_t 的字段含义与标准 I2S 不同：
 *   ws_io_num   -> PDM CLK  (GPIO2)
 *   data_in_num -> PDM DATA (GPIO1)
 *   bck_io_num  -> 必须为 -1
 * 填错会表现为「初始化成功但读到全 0」。
 *
 * 硬件自带 PDM→PCM 转换与抽取滤波，i2s_read() 直接得到 16bit PCM，
 * 无需软件解调。
 *
 * 采样参数被后端锁定
 * ------------------
 * 后端 stt.go 写死 pcm_s16le / 16000Hz / 16bit / mono，
 * 故这三个值定义在 board_pins.h 且不可配置。
 */
#ifndef PULSE_AUDIO_H
#define PULSE_AUDIO_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "pulse_vad.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    uint64_t total_samples;  /* 累计采样点数 */
    uint64_t total_bytes;    /* 累计 PCM 字节数 */
    uint32_t read_errors;    /* i2s_read 失败次数 */
    uint32_t overruns;       /* DMA 溢出（读取跟不上）次数 */
    int16_t last_peak;       /* 最近一帧峰值，用于判断削波 */
    uint32_t last_energy;    /* 最近一帧能量 */
    bool dc_offset_warning;  /* 检出显著直流偏移 */
    int32_t dc_offset;       /* 当前估计的直流偏移量 */
} audio_stats_t;

/**
 * 初始化 I2S0 PDM RX。
 * @return false 表示 I2S 驱动安装或引脚配置失败
 */
bool audio_init(void);

bool audio_is_ready(void);

/** 停止 I2S 并释放资源（进深睡前调用，可显著降低功耗）*/
void audio_stop(void);

/** 重新启动已 stop 的 I2S */
bool audio_resume(void);

/**
 * 读取一帧 PCM。阻塞直到读满或超时。
 * @param pcm       输出缓冲（16bit 单声道）
 * @param max_count 缓冲可容纳的采样点数
 * @param got       实际读到的采样点数
 * @param timeout_ms 超时
 * @return false 表示读取失败
 */
bool audio_read(int16_t *pcm, size_t max_count, size_t *got, uint32_t timeout_ms);

/**
 * 读取一帧并做 VAD 判决，同时维护 pre-roll 环形缓冲。
 *
 * @param vad_out    VAD 结果。vad_out->speech_started 为 true 时，
 *                   调用方应先取 pre-roll 数据再写当前帧。
 * @return 同 audio_read
 */
bool audio_read_with_vad(int16_t *pcm, size_t max_count, size_t *got, vad_result_t *vad_out,
                         uint32_t timeout_ms);

/**
 * 取出 pre-roll 环形缓冲中的历史音频。
 * 在 vad_out->speech_started 为 true 时调用，把语音起点之前的
 * preroll_ms 毫秒数据补进段首，避免丢掉首字。
 *
 * @param out       输出缓冲
 * @param max_count 容量（采样点）
 * @return 实际写出的采样点数
 */
size_t audio_get_preroll(int16_t *out, size_t max_count);

/** 清空 pre-roll 缓冲（开新段后调用，避免重复写入）*/
void audio_clear_preroll(void);

/** 采集统计 */
void audio_get_stats(audio_stats_t *out);
void audio_reset_stats(void);

/** 串口打印音频诊断 */
void audio_dump(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_AUDIO_H */
