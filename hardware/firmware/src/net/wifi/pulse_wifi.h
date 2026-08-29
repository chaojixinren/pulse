/**
 * @file pulse_wifi.h
 * @brief WiFi 连接管理 + AP 配网
 *
 * 命名约定
 * --------
 * 所有对外符号统一加 pulse_ 前缀。M6 曾因 rtc_init() 与 ESP-IDF 内部的
 * rtc_init() 重名导致链接失败（multiple definition），HAL 层不应占用
 * IDF 的通用命名空间。
 *
 * 连接策略
 * --------
 * 用非阻塞状态机，不用 WiFi.begin() 后死等 —— 否则录音任务会被卡住。
 * 断线后按指数退避重连：reconnect_min_ms 起，每次翻倍，上限
 * reconnect_max_ms。这样弱信号环境下不会疯狂重试烧电。
 *
 * 配网方式
 * --------
 * 设备无键盘，采用 AP 热点 + 网页表单：
 *   1. 无凭据或连接失败超过阈值 → 开启 AP "Pulse-XXXX"
 *   2. 手机连上后访问 192.168.4.1 填写 SSID/密码
 *   3. 保存到 NVS（config_save_wifi），切回 STA 模式连接
 *
 * 不用 SmartConfig/ESPTouch：它依赖特定手机 App，现场调试更麻烦。
 * 网页表单只要有浏览器就能用。
 */
#ifndef PULSE_WIFI_H
#define PULSE_WIFI_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    PULSE_WIFI_IDLE = 0,      /* 未启动 */
    PULSE_WIFI_NO_CREDENTIALS, /* 无凭据，需配网 */
    PULSE_WIFI_CONNECTING,
    PULSE_WIFI_CONNECTED,
    PULSE_WIFI_RECONNECT_WAIT, /* 退避等待中 */
    PULSE_WIFI_AP_PROVISION,   /* AP 配网模式 */
} pulse_wifi_state_t;

typedef struct {
    pulse_wifi_state_t state;
    bool connected;
    char ssid[64];
    char ip[16];
    int8_t rssi;
    uint32_t connect_attempts;
    uint32_t disconnect_count;
    uint32_t connected_seconds; /* 本次连接已保持时长 */
    uint32_t backoff_ms;        /* 当前退避间隔 */
} pulse_wifi_status_t;

/**
 * 初始化 WiFi。不阻塞，仅设置模式并发起首次连接。
 * 无凭据时进入 PULSE_WIFI_NO_CREDENTIALS，由调用方决定是否开配网。
 */
bool pulse_wifi_init(void);

/**
 * 驱动状态机。需周期调用（建议 100~500ms 一次）。
 * 负责超时判定、退避重连、AP 门户请求处理。
 */
void pulse_wifi_tick(void);

bool pulse_wifi_is_connected(void);
pulse_wifi_state_t pulse_wifi_state(void);
const char *pulse_wifi_state_str(pulse_wifi_state_t s);
bool pulse_wifi_get_status(pulse_wifi_status_t *out);

/** 用新凭据立刻重连（配网成功后调用）*/
bool pulse_wifi_connect(const char *ssid, const char *password);

/**
 * 启动 AP 配网门户。
 * AP 名为 Pulse-<MAC 后 4 位>，无密码，网关 192.168.4.1。
 */
bool pulse_wifi_start_provisioning(void);

/** 关闭配网门户并回到 STA 模式 */
void pulse_wifi_stop_provisioning(void);

/** 断开并关闭射频（进深睡前调用，省电）*/
void pulse_wifi_shutdown(void);

/** 扫描周边 AP，结果打印到串口（配网时供参考）*/
int pulse_wifi_scan(void);

void pulse_wifi_dump(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_WIFI_H */
