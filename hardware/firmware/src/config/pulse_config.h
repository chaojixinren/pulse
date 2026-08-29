/**
 * @file pulse_config.h
 * @brief Pulse 运行时配置（三层覆盖：编译期默认 → TF 卡 config.json → NVS）
 *
 * 设计约定
 * ---------
 * 1. 本文件只放"可调参数"。焊死的引脚在 board_pins.h。
 * 2. 加载顺序：内置默认值 → /config.json（TF 卡，现场调参）→ NVS（配网写入的凭据）。
 *    后者覆盖前者。NVS 优先级最高，因为 WiFi 密码和 device_token 是运行时写入的。
 * 3. 任何模块都不得自己 open("/config.json")，统一走 config_get()。
 * 4. 无 TF 卡 / JSON 损坏 / 字段缺失 都不是致命错误，回退到默认值继续启动。
 */
#ifndef PULSE_CONFIG_H
#define PULSE_CONFIG_H

#include <stdbool.h>
#include <stdint.h>

#include "board_pins.h"

#ifdef __cplusplus
extern "C" {
#endif

#define CFG_STR_LEN 64
#define CFG_URL_LEN 160
#define CFG_TOKEN_LEN 128

/* ───────── 网络 ───────── */
typedef struct {
    char ssid[CFG_STR_LEN];
    char password[CFG_STR_LEN];
    uint32_t connect_timeout_ms; /* 单次连接超时 */
    uint32_t reconnect_min_ms;   /* 断线重连退避下限 */
    uint32_t reconnect_max_ms;   /* 断线重连退避上限 */
} cfg_wifi_t;

/* ───────── 云端 ───────── */
typedef struct {
    char base_url[CFG_URL_LEN];  /* 例: https://api.example.com */
    char upload_path[CFG_STR_LEN]; /* 例: /api/v1/audio/upload */
    char device_id[CFG_STR_LEN];   /* 空则启动时用 MAC 生成 */
    /* 后端目前只认用户级 JWT（middleware.Auth），设备鉴权尚未实现。
     * 做成 scheme + token 两段拼接，等后端补上设备鉴权后改配置即可，
     * 无需重新编译固件。 */
    char auth_scheme[16]; /* "Bearer" 或 "Device" */
    char auth_token[CFG_TOKEN_LEN];
    uint32_t request_timeout_ms;
    uint8_t max_retry;          /* 单段最大重试次数，超出转入 failed/ */
    uint32_t retry_backoff_ms;  /* 首次重试退避，之后指数增长 */
    bool verify_tls;            /* false 则跳过证书校验（仅调试用） */
} cfg_cloud_t;

/* ───────── 音频采集 ───────── */
typedef struct {
    /* 采样率/位宽/声道被后端写死，故不在此暴露，见 board_pins.h */
    uint16_t dma_buf_count;
    uint16_t dma_buf_len;       /* 单个 DMA 缓冲的采样点数 */
    int16_t gain_shift;         /* PDM 转 PCM 后的左移位数，用于补增益 */
    bool dc_removal;            /* PDM 麦常见 DC 偏移，默认开高通滤除 */
    uint32_t segment_ms;        /* 分段时长，默认 30000 */
    uint32_t segment_max_bytes; /* 分段字节上限，默认 960000 (=30s) */
} cfg_audio_t;

/* ───────── VAD 人声检测 ───────── */
typedef struct {
    bool enabled;
    uint32_t energy_threshold;  /* 短时能量阈值，需现场实测调整 */
    uint16_t zcr_min;           /* 过零率下限，滤掉直流与低频噪声 */
    uint16_t zcr_max;           /* 过零率上限，滤掉白噪声/摩擦音 */
    uint32_t speech_confirm_ms; /* 连续超阈值多久才判定为语音起点 */
    uint32_t hangover_ms;       /* 语音结束后继续录多久，避免切掉尾音 */
    uint32_t preroll_ms;        /* 语音起点前回溯多久，避免切掉首字 */
    uint32_t min_segment_ms;    /* 短于此长度的段直接丢弃，认为是杂音 */
} cfg_vad_t;

/* ───────── 存储 ───────── */
typedef struct {
    char spool_dir[CFG_STR_LEN];  /* 待上传目录 */
    char failed_dir[CFG_STR_LEN]; /* 超过重试上限的段 */
    uint32_t max_spool_mb;        /* spool 占用上限，超出按时间淘汰最旧 */
    uint32_t min_free_mb;         /* 剩余空间低于此值停止录音 */
    uint8_t sd_freq_mhz;          /* SDMMC 时钟；GPIO45/46 有上拉时需降频 */
    bool sd_use_1bit;             /* 4-bit 起不来时回退 1-bit */
    /* 无 TF 卡时是否降级到 PSRAM 环形缓冲（8MB 约 4 分钟音频）。
     * 保证没插卡也能联调 M5/M8/M10。 */
    bool fallback_to_psram;
    uint32_t psram_buffer_kb;
} cfg_storage_t;

/* ───────── 电源（三级功耗）───────── */
typedef struct {
    uint32_t screen_dim_ms;    /* 无触摸多久后灭屏 → Idle */
    uint32_t deep_sleep_ms;    /* Idle 后多久进深睡；0 = 永不自动深睡 */
    bool sleep_requires_empty_spool; /* 队列没传完就不睡 */
    uint8_t battery_warn_pct;  /* 低电告警阈值，默认 10 */
    uint8_t battery_crit_pct;  /* 强制深睡保护阈值，默认 5 */
    uint8_t backlight_pct;     /* 默认背光亮度 */
    uint32_t pmu_poll_ms;      /* AXP2101 轮询周期 */
} cfg_power_t;

/* ───────── 界面 ───────── */
typedef struct {
    uint8_t rotation;      /* 0~3 */
    uint32_t lvgl_tick_ms;
    char timezone[CFG_STR_LEN]; /* POSIX TZ，例 CST-8 */
    char ntp_server[CFG_STR_LEN];
} cfg_ui_t;

/* ───────── 顶层 ───────── */
typedef struct {
    cfg_wifi_t wifi;
    cfg_cloud_t cloud;
    cfg_audio_t audio;
    cfg_vad_t vad;
    cfg_storage_t storage;
    cfg_power_t power;
    cfg_ui_t ui;

    bool loaded_from_file; /* 是否成功读到 /config.json */
    bool loaded_from_nvs;
} pulse_config_t;

/* 加载配置：默认值 → /config.json → NVS。
 * 返回 false 表示 config.json 缺失或解析失败（已回退默认值，仍可继续运行）。 */
bool config_load(void);

/* 取全局只读配置。config_load() 之前调用会得到编译期默认值。 */
const pulse_config_t *config_get(void);

/* 把 WiFi 凭据与 device_token 持久化到 NVS（配网流程使用） */
bool config_save_wifi(const char *ssid, const char *password);
bool config_save_token(const char *scheme, const char *token);

/* 把背光亮度持久化到 NVS（设置页滑块使用，5~100） */
bool config_save_backlight(uint8_t percent);

/* 串口打印全部配置项（token/密码脱敏），供 M0 测试与现场排障 */
void config_dump(void);

/* 运行时校验配置自洽性，返回问题条数并逐条打印 */
int config_validate(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_CONFIG_H */
