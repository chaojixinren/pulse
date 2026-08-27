/**
 * @file pulse_ui.h
 * @brief LVGL 界面层：穿戴式语音记录设备的主界面
 *
 * 设计原则
 * --------
 * 屏幕只有 240x280，是**穿戴设备**而非内容消费设备，界面必须一眼看清状态。
 * 因此不做多级菜单，单屏承载全部关键信息，配网等低频操作用独立弹层。
 *
 * 数据注入（可测试性）
 * --------------------
 * 界面不直接调用任何 HAL 模块 —— 所有显示数据由上层（M10 测试或 M11 app）
 * 通过一个统一的 pulse_ui_state_t 结构体周期推入。这样：
 *   - M10 测试可以用模拟数据遍历所有界面状态，无需真实外设
 *   - M11 把真实模块状态填进同一个结构体即可
 *   - 界面层与硬件完全解耦
 *
 * 字体
 * ----
 * 默认用内置 Montserrat（仅 ASCII，英文/图标文本开箱即用）。
 * 中文界面需要用 tools/gen_font.py 生成 CJK 子集字体后，
 * 修改本文件的 UI_FONT 宏即可切换（见文件内注释）。
 */
#ifndef PULSE_UI_H
#define PULSE_UI_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* 录音状态 */
typedef enum {
    UI_REC_STOPPED = 0, /* 停止：用户手动收工或深睡 */
    UI_REC_IDLE,        /* 待机：录音开，VAD 静音中 */
    UI_REC_SPEECH,      /* 录音中：检测到人声 */
    UI_REC_HANGOVER,    /* 尾音保持 */
} ui_rec_state_t;

/* 上传状态 */
typedef enum {
    UI_UPLOAD_IDLE = 0,
    UI_UPLOAD_OFFLINE, /* 断网，段在本地排队 */
    UI_UPLOAD_SENDING,
    UI_UPLOAD_BACKOFF,
    UI_UPLOAD_DISABLED,
} ui_upload_state_t;

/* 存储后端 */
typedef enum {
    UI_STORAGE_SD = 0,
    UI_STORAGE_PSRAM, /* 降级：未插卡或挂载失败 */
    UI_STORAGE_NONE,
} ui_storage_kind_t;

/* 界面状态：M10 模拟 / M11 真实数据都填这里 */
typedef struct {
    /* 时间 */
    bool time_valid;
    int hour, minute, second;
    int year, month, day;

    /* 电源 */
    bool battery_present;
    int battery_percent;      /* 0~100，-1 未知 */
    bool charging;
    bool battery_warn;        /* 低电告警置位 */

    /* 网络 */
    bool wifi_connected;
    int8_t rssi;

    /* 录音 */
    ui_rec_state_t rec_state;
    uint32_t rec_duration_s;  /* 当前段已录时长 */

    /* 上传 */
    ui_upload_state_t upload_state;
    uint32_t pending_count;   /* 待上传段数 */
    uint32_t sent_ok;         /* 已上传 */
    uint32_t sent_failed;     /* 归档失败 */

    /* 存储 */
    ui_storage_kind_t storage_kind;
    uint64_t storage_free_mb;
    uint64_t storage_total_mb; /* 总容量（MB），容量条用；0 表示未知 */
} pulse_ui_state_t;

/** 初始化界面（创建所有对象）。须在 lv_port_init() 之后、持锁调用。 */
void pulse_ui_init(void);

/**
 * 推入最新状态并刷新界面。须持 LVGL 锁调用。
 * 内部做差分，只在值变化时更新对象，避免无谓重绘。
 */
void pulse_ui_update(const pulse_ui_state_t *st);

/**
 * 弹出一条短暂提示（如「配网成功」「低电告警」）。
 * 自动在几秒后消失。须持锁调用。
 */
void pulse_ui_toast(const char *msg);

/**
 * 显示配网提示覆盖层（WiFi 未配置时）。
 * @param show true 显示，false 隐藏
 */
void pulse_ui_show_provision_hint(bool show);

/**
 * 更新配网提示层显示的热点名。ap_ssid 为空表示热点尚未开启。
 * 须持 LVGL 锁调用。
 */
void pulse_ui_set_provision_ap(const char *ap_ssid);

/** 设置配网提示模式：false=初次配网（默认），true=更换网络 */
void pulse_ui_set_provision_mode(bool switching);

/** 当前是否需要配网提示（供 app 层判断）*/
bool pulse_ui_provision_hint_visible(void);

/* ─────────── 设置页（亮度调节）─────────── */

/** 背光亮度变化回调。commit=false 表示拖动预览（实时调背光），
 *  true 表示松开确认（应持久化）。app 层注册。 */
typedef void (*pulse_ui_backlight_cb_t)(uint8_t percent, bool commit);

/** 注册背光回调（app 层在 pulse_ui_init 后调用）*/
void pulse_ui_set_backlight_cb(pulse_ui_backlight_cb_t cb);

/** 同步当前亮度到 UI（app 初始化后调用，作为设置页滑块的初始值）*/
void pulse_ui_set_backlight(uint8_t percent);

/** WiFi 设置按钮回调（app 层注册，点击后启动 AP 配网流程）*/
typedef void (*pulse_ui_wifi_cb_t)(void);

/** 注册 WiFi 设置回调（app 层在 pulse_ui_init 后调用）*/
void pulse_ui_set_wifi_cb(pulse_ui_wifi_cb_t cb);

/** 打开/关闭设置页。须持 LVGL 锁调用。 */
void pulse_ui_open_settings(void);
void pulse_ui_close_settings(void);
bool pulse_ui_settings_visible(void);

void pulse_ui_dump(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_UI_H */
