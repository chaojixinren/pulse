/**
 * @file pulse_ui.cpp
 * @brief LVGL 界面实现：单屏穿戴式状态界面
 *
 * 布局（240x280 竖屏，从上到下）：
 *   ┌────────────────────┐
 *   │ 顶部状态栏          │ 时间 + WiFi/RSSI + 电量
 *   │                     │
 *   │ 录音状态大卡片       │ 状态图标 + 当前段时长
 *   │  （VAD 状态）        │
 *   │                     │
 *   │ 上传队列            │ 待传段数 + 成功/失败 + 上传状态
 *   │                     │
 *   │ 存储/降级提示        │ SD 容量 / PSRAM 降级警告
 *   │                     │
 *   │ 底部（可选配网提示） │
 *   └────────────────────┘
 */
#include "pulse_ui.h"

#include <lvgl.h>

#include <Arduino.h>
#include <stdio.h>
#include <string.h>

#include "board_pins.h"

LV_FONT_DECLARE(pulse_cjk_12);
LV_FONT_DECLARE(pulse_cjk_14);
LV_FONT_DECLARE(pulse_cjk_16);
LV_FONT_DECLARE(pulse_cjk_20);
LV_FONT_DECLARE(pulse_cjk_28);

static const char *TAG = "ui";

/* ─────────── 颜色（深色主题，参考 Material Dark）─────────── */
#define C_BG 0x101418
#define C_CARD 0x1C2228
#define C_ACCENT 0x4FC3F7
#define C_TEXT 0xE8EAED
#define C_DIM 0x78909C
#define C_OK 0x66BB6A
#define C_WARN 0xFFA726
#define C_ERR 0xEF5350
#define C_REC 0xFF5252

static lv_obj_t *g_scr = nullptr;

/* 状态栏 */
static lv_obj_t *g_lbl_time = nullptr;
static lv_obj_t *g_wifi_bars[4];       /* WiFi 信号格（4 竖条）*/
static lv_obj_t *g_batt_frame = nullptr; /* 电池外框 */
static lv_obj_t *g_batt_fill = nullptr;  /* 电池内部填充 */
static lv_obj_t *g_batt_cap = nullptr;   /* 电池头 */
static lv_obj_t *g_lbl_batt = nullptr;   /* 电量百分比文字 */

/* 录音卡片 */
static lv_obj_t *g_card_rec = nullptr;
static lv_obj_t *g_lbl_rec_state = nullptr;
static lv_obj_t *g_lbl_rec_dur = nullptr;
static lv_obj_t *g_rec_dot = nullptr;    /* 录音脉冲圆点 */
static lv_anim_t g_rec_anim;             /* 呼吸动画 */
static bool g_rec_anim_running = false;

/* 上传 */
static lv_obj_t *g_lbl_upload_state = nullptr;
static lv_obj_t *g_lbl_upload_count = nullptr;
static lv_obj_t *g_upload_dot = nullptr; /* 上传状态彩色圆点 */

/* 存储 */
static lv_obj_t *g_lbl_storage = nullptr;
static lv_obj_t *g_bar_storage = nullptr; /* 存储容量条 */

/* 配网提示覆盖层 */
static lv_obj_t *g_provision_layer = nullptr;
static lv_obj_t *g_lbl_prov = nullptr;
static bool g_prov_switching = false;

/* toast */
static lv_obj_t *g_toast = nullptr;
static lv_timer_t *g_toast_timer = nullptr;

/* 设置页 */
static lv_obj_t *g_settings_layer = nullptr;
static lv_obj_t *g_slider_backlight = nullptr;
static lv_obj_t *g_lbl_backlight_val = nullptr;
static pulse_ui_backlight_cb_t g_backlight_cb = nullptr;
static pulse_ui_wifi_cb_t g_wifi_cb = nullptr;
static uint8_t g_cur_backlight = 70; /* 初始默认值，app 初始化后用 pulse_ui_set_backlight 同步 */

/* ─────────── 差分缓存，避免每帧重建字符串 ─────────── */
typedef struct {
    char time[16];
    char wifi[12];
    char batt[12];
    char rec_state[16];
    char rec_dur[16];
    char upload_state[24];
    /* 最大 "队列 UINT_MAX  已传 UINT_MAX  失败 UINT_MAX" = 43 字符 + NUL */
    char upload_count[48];
    char storage[48];
    int battery_pct;
    int pending;
    int bar_val;
} ui_cache_t;

static ui_cache_t g_cache;
static bool g_cache_init = false;

/* ─────────── 辅助 ─────────── */

static lv_color_t lc(uint32_t c) {
    /* LVGL 8.x 的 lv_color_t 在 16 位深度下布局为 R5G6B5 的 little-endian。
     * 用 lv_color_hex 最稳妥：它正确处理 RGB888 -> RGB565 与字节序。
     * 之前误用 display_color565（那是 ST7789 线序，不是 LVGL 约定），
     * 会导致颜色错乱。 */
    return lv_color_hex(c & 0xFFFFFF);
}

static const char *rec_state_label(ui_rec_state_t s) {
    switch (s) {
        case UI_REC_STOPPED: return "已停止";
        case UI_REC_IDLE: return "待机（静音）";
        case UI_REC_SPEECH: return "录音中";
        case UI_REC_HANGOVER: return "尾音…";
        default: return "?";
    }
}

static const char *upload_state_label(ui_upload_state_t s) {
    switch (s) {
        case UI_UPLOAD_OFFLINE: return "离线排队中";
        case UI_UPLOAD_SENDING: return "上传中…";
        case UI_UPLOAD_BACKOFF: return "退避等待";
        case UI_UPLOAD_DISABLED: return "未配置";
        default: return "已同步";
    }
}

static const char *storage_label(ui_storage_kind_t k) {
    switch (k) {
        case UI_STORAGE_PSRAM: return "未插卡";
        case UI_STORAGE_NONE: return "存储不可用";
        default: return "SD 卡";
    }
}

static void update_if_changed(lv_obj_t *obj, const char *val, char *cache, size_t cache_cap) {
    if (!obj) return;
    if (strncmp(cache, val, cache_cap) == 0) return;
    strncpy(cache, val, cache_cap - 1);
    cache[cache_cap - 1] = '\0';
    lv_label_set_text(obj, val);
}

/* toast 自动消失 */
static void toast_timer_cb(lv_timer_t *t) {
    (void)t;
    if (g_toast) {
        lv_obj_del(g_toast);
        g_toast = nullptr;
    }
    if (g_toast_timer) {
        lv_timer_del(g_toast_timer);
        g_toast_timer = nullptr;
    }
}

void pulse_ui_toast(const char *msg) {
    if (!g_scr) return;

    /* 移除旧 toast */
    if (g_toast) {
        lv_obj_del(g_toast);
        g_toast = nullptr;
    }
    if (g_toast_timer) {
        lv_timer_del(g_toast_timer);
        g_toast_timer = nullptr;
    }

    g_toast = lv_label_create(g_scr);
    lv_label_set_text(g_toast, msg);
    lv_obj_set_style_text_font(g_toast, &pulse_cjk_14, LV_PART_MAIN);
    lv_obj_set_style_bg_color(g_toast, lc(0x263238), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_toast, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_set_style_text_color(g_toast, lc(C_TEXT), LV_PART_MAIN);
    lv_obj_set_style_pad_all(g_toast, 8, LV_PART_MAIN);
    lv_obj_set_style_radius(g_toast, 6, LV_PART_MAIN);
    lv_label_set_long_mode(g_toast, LV_LABEL_LONG_WRAP);
    lv_obj_set_width(g_toast, 200);
    lv_obj_align(g_toast, LV_ALIGN_BOTTOM_MID, 0, -8);
    lv_obj_move_foreground(g_toast);

    g_toast_timer = lv_timer_create(toast_timer_cb, 3000, nullptr);
    lv_timer_set_repeat_count(g_toast_timer, 1);
}

/* ─────────── 录音脉冲动画 ─────────── */

static void rec_dot_anim_cb(void *var, int32_t v) {
    lv_obj_set_style_opa((lv_obj_t *)var, (lv_opa_t)v, LV_PART_MAIN);
}

static void start_rec_pulse(void) {
    if (!g_rec_dot || g_rec_anim_running) return;
    g_rec_anim_running = true;
    lv_anim_init(&g_rec_anim);
    lv_anim_set_var(&g_rec_anim, g_rec_dot);
    lv_anim_set_exec_cb(&g_rec_anim, rec_dot_anim_cb);
    lv_anim_set_values(&g_rec_anim, LV_OPA_20, LV_OPA_COVER);
    lv_anim_set_time(&g_rec_anim, 600);
    lv_anim_set_playback_time(&g_rec_anim, 600);
    lv_anim_set_repeat_count(&g_rec_anim, LV_ANIM_REPEAT_INFINITE);
    lv_anim_start(&g_rec_anim);
}

static void stop_rec_pulse(void) {
    if (!g_rec_anim_running) return;
    g_rec_anim_running = false;
    lv_anim_del(g_rec_dot, rec_dot_anim_cb);
    if (g_rec_dot) lv_obj_set_style_opa(g_rec_dot, LV_OPA_COVER, LV_PART_MAIN);
}

/* ─────────── 设置页事件 ─────────── */

static void backlight_slider_cb(lv_event_t *e) {
    int v = lv_slider_get_value(g_slider_backlight);
    bool commit = (lv_event_get_code(e) == LV_EVENT_RELEASED);
    if (g_lbl_backlight_val) {
        lv_label_set_text_fmt(g_lbl_backlight_val, "%d%%", v);
    }
    if (g_backlight_cb) {
        g_backlight_cb((uint8_t)v, commit);
    }
}

static void settings_btn_cb(lv_event_t *e) {
    (void)e;
    pulse_ui_open_settings();
}

static void settings_close_cb(lv_event_t *e) {
    (void)e;
    pulse_ui_close_settings();
}

static void wifi_btn_cb(lv_event_t *e) {
    (void)e;
    /* 触发 app 层启动 AP 配网，然后关掉设置页（配网提示会接管显示）*/
    if (g_wifi_cb) g_wifi_cb();
    pulse_ui_close_settings();
}

static void prov_close_cb(lv_event_t *e) {
    (void)e;
    pulse_ui_show_provision_hint(false);
}

/* ─────────── 界面构建 ─────────── */

void pulse_ui_init(void) {
    g_scr = lv_scr_act();
    lv_obj_set_style_bg_color(g_scr, lc(C_BG), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_scr, LV_OPA_COVER, LV_PART_MAIN);
    /* 关键：screen 背景默认 bg_opa 是透明的，不设则不填充背景色，
     * ST7789 显存初始是随机的，会显示花屏。 */
    lv_obj_set_style_bg_opa(g_scr, LV_OPA_COVER, LV_PART_MAIN);

    /* 顶部状态栏：时间（左）+ WiFi 信号格 + 电池（右）*/
    lv_obj_t *bar = lv_obj_create(g_scr);
    lv_obj_remove_style_all(bar);
    lv_obj_set_size(bar, 240, 36);
    lv_obj_align(bar, LV_ALIGN_TOP_MID, 0, 0);
    lv_obj_set_flex_flow(bar, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(bar, LV_FLEX_ALIGN_SPACE_BETWEEN, LV_FLEX_ALIGN_CENTER,
                          LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_hor(bar, 10, LV_PART_MAIN);

    g_lbl_time = lv_label_create(bar);
    lv_obj_set_style_text_font(g_lbl_time, &pulse_cjk_16, LV_PART_MAIN);
    lv_obj_set_style_text_color(g_lbl_time, lc(C_TEXT), LV_PART_MAIN);
    lv_label_set_text(g_lbl_time, "--:--");

    /* 右侧容器：WiFi 格 + 电池，flex 横排右对齐 */
    lv_obj_t *right = lv_obj_create(bar);
    lv_obj_remove_style_all(right);
    lv_obj_set_size(right, 92, 20);
    lv_obj_set_flex_flow(right, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(right, LV_FLEX_ALIGN_END, LV_FLEX_ALIGN_CENTER,
                          LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_column(right, 8, LV_PART_MAIN);

    /* WiFi 信号格：4 竖条底部对齐，高度递增 */
    lv_obj_t *wifi = lv_obj_create(right);
    lv_obj_remove_style_all(wifi);
    lv_obj_set_size(wifi, 17, 14);
    lv_obj_set_flex_flow(wifi, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(wifi, LV_FLEX_ALIGN_END, LV_FLEX_ALIGN_END, LV_FLEX_ALIGN_END);
    lv_obj_set_style_pad_column(wifi, 1, LV_PART_MAIN);
    for (int i = 0; i < 4; i++) {
        g_wifi_bars[i] = lv_obj_create(wifi);
        lv_obj_remove_style_all(g_wifi_bars[i]);
        lv_obj_set_size(g_wifi_bars[i], 3, 4 + i * 3);
        lv_obj_set_style_bg_color(g_wifi_bars[i], lc(C_DIM), LV_PART_MAIN);
        lv_obj_set_style_bg_opa(g_wifi_bars[i], LV_OPA_COVER, LV_PART_MAIN);
        lv_obj_set_style_radius(g_wifi_bars[i], 1, LV_PART_MAIN);
    }

    /* 电池：外框 + 内部填充 + 头 + 百分比文字 */
    lv_obj_t *batt = lv_obj_create(right);
    lv_obj_remove_style_all(batt);
    lv_obj_set_size(batt, 64, 14);
    lv_obj_set_flex_flow(batt, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(batt, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER,
                          LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_column(batt, 3, LV_PART_MAIN);

    g_batt_frame = lv_obj_create(batt);
    lv_obj_remove_style_all(g_batt_frame);
    lv_obj_set_size(g_batt_frame, 26, 13);
    lv_obj_set_style_border_width(g_batt_frame, 1, LV_PART_MAIN);
    lv_obj_set_style_border_color(g_batt_frame, lc(C_DIM), LV_PART_MAIN);
    lv_obj_set_style_radius(g_batt_frame, 3, LV_PART_MAIN);
    lv_obj_set_style_pad_all(g_batt_frame, 1, LV_PART_MAIN);

    g_batt_fill = lv_obj_create(g_batt_frame);
    lv_obj_remove_style_all(g_batt_fill);
    lv_obj_set_size(g_batt_fill, 0, 11);
    lv_obj_set_style_bg_color(g_batt_fill, lc(C_OK), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_batt_fill, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_set_style_radius(g_batt_fill, 1, LV_PART_MAIN);
    lv_obj_align(g_batt_fill, LV_ALIGN_LEFT_MID, 0, 0);

    g_batt_cap = lv_obj_create(batt);
    lv_obj_remove_style_all(g_batt_cap);
    lv_obj_set_size(g_batt_cap, 2, 6);
    lv_obj_set_style_bg_color(g_batt_cap, lc(C_DIM), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_batt_cap, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_set_style_radius(g_batt_cap, 1, LV_PART_MAIN);

    g_lbl_batt = lv_label_create(batt);
    lv_obj_set_style_text_color(g_lbl_batt, lc(C_TEXT), LV_PART_MAIN);
    lv_obj_set_style_text_font(g_lbl_batt, &pulse_cjk_12, LV_PART_MAIN);
    lv_label_set_text(g_lbl_batt, "--%");

    /* 录音卡片 */
    g_card_rec = lv_obj_create(g_scr);
    lv_obj_remove_style_all(g_card_rec);
    lv_obj_set_style_bg_color(g_card_rec, lc(C_CARD), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_card_rec, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_set_style_radius(g_card_rec, 12, LV_PART_MAIN);
    lv_obj_set_style_pad_all(g_card_rec, 14, LV_PART_MAIN);
    lv_obj_set_size(g_card_rec, 216, 92);
    lv_obj_align(g_card_rec, LV_ALIGN_TOP_MID, 0, 40);

    g_lbl_rec_state = lv_label_create(g_card_rec);
    lv_obj_set_style_text_font(g_lbl_rec_state, &pulse_cjk_20, LV_PART_MAIN);
    lv_obj_set_style_text_color(g_lbl_rec_state, lc(C_TEXT), LV_PART_MAIN);
    lv_label_set_text(g_lbl_rec_state, "待机");
    lv_obj_align(g_lbl_rec_state, LV_ALIGN_TOP_LEFT, 0, 0);

    g_lbl_rec_dur = lv_label_create(g_card_rec);
    lv_obj_set_style_text_font(g_lbl_rec_dur, &pulse_cjk_28, LV_PART_MAIN);
    lv_obj_set_style_text_color(g_lbl_rec_dur, lc(C_ACCENT), LV_PART_MAIN);
    lv_label_set_text(g_lbl_rec_dur, "00:00");
    lv_obj_align(g_lbl_rec_dur, LV_ALIGN_BOTTOM_LEFT, 0, 0);

    /* 录音脉冲圆点：右上角，录音时红色呼吸，非录音灰色 */
    g_rec_dot = lv_obj_create(g_card_rec);
    lv_obj_remove_style_all(g_rec_dot);
    lv_obj_set_size(g_rec_dot, 10, 10);
    lv_obj_set_style_radius(g_rec_dot, LV_RADIUS_CIRCLE, LV_PART_MAIN);
    lv_obj_set_style_bg_color(g_rec_dot, lc(C_DIM), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_rec_dot, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_align(g_rec_dot, LV_ALIGN_TOP_RIGHT, 0, 2);

    /* 上传卡片 */
    lv_obj_t *card_up = lv_obj_create(g_scr);
    lv_obj_remove_style_all(card_up);
    lv_obj_set_style_bg_color(card_up, lc(C_CARD), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(card_up, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_set_style_radius(card_up, 12, LV_PART_MAIN);
    lv_obj_set_style_pad_all(card_up, 12, LV_PART_MAIN);
    lv_obj_set_size(card_up, 216, 72);
    lv_obj_align(card_up, LV_ALIGN_TOP_MID, 0, 140);

    g_lbl_upload_state = lv_label_create(card_up);
    lv_obj_set_style_text_color(g_lbl_upload_state, lc(C_TEXT), LV_PART_MAIN);
    lv_obj_set_style_text_font(g_lbl_upload_state, &pulse_cjk_14, LV_PART_MAIN);
    lv_label_set_text(g_lbl_upload_state, "上传");
    lv_obj_align(g_lbl_upload_state, LV_ALIGN_TOP_LEFT, 0, 0);

    /* 上传状态圆点：右上角，颜色随状态变化 */
    g_upload_dot = lv_obj_create(card_up);
    lv_obj_remove_style_all(g_upload_dot);
    lv_obj_set_size(g_upload_dot, 8, 8);
    lv_obj_set_style_radius(g_upload_dot, LV_RADIUS_CIRCLE, LV_PART_MAIN);
    lv_obj_set_style_bg_color(g_upload_dot, lc(C_DIM), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_upload_dot, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_align(g_upload_dot, LV_ALIGN_TOP_RIGHT, 0, 2);

    g_lbl_upload_count = lv_label_create(card_up);
    lv_obj_set_style_text_color(g_lbl_upload_count, lc(C_DIM), LV_PART_MAIN);
    lv_obj_set_style_text_font(g_lbl_upload_count, &pulse_cjk_12, LV_PART_MAIN);
    lv_label_set_text(g_lbl_upload_count, "");
    lv_obj_align(g_lbl_upload_count, LV_ALIGN_BOTTOM_LEFT, 0, 0);

    /* 存储状态 */
    g_lbl_storage = lv_label_create(g_scr);
    lv_obj_set_style_text_color(g_lbl_storage, lc(C_DIM), LV_PART_MAIN);
    lv_obj_set_style_text_font(g_lbl_storage, &pulse_cjk_12, LV_PART_MAIN);
    lv_label_set_text(g_lbl_storage, "");
    lv_obj_align(g_lbl_storage, LV_ALIGN_TOP_MID, 0, 218);

    /* 存储容量条 */
    g_bar_storage = lv_bar_create(g_scr);
    lv_obj_set_size(g_bar_storage, 216, 6);
    lv_obj_align(g_bar_storage, LV_ALIGN_TOP_MID, 0, 236);
    lv_obj_set_style_radius(g_bar_storage, 3, LV_PART_MAIN);
    lv_obj_set_style_bg_color(g_bar_storage, lc(C_CARD), LV_PART_MAIN);
    lv_obj_set_style_bg_color(g_bar_storage, lc(C_ACCENT), LV_PART_INDICATOR);
    lv_bar_set_range(g_bar_storage, 0, 100);
    lv_bar_set_value(g_bar_storage, 0, LV_ANIM_OFF);

    /* 配网提示覆盖层（默认隐藏）*/
    g_provision_layer = lv_obj_create(g_scr);
    lv_obj_remove_style_all(g_provision_layer);
    lv_obj_set_style_bg_color(g_provision_layer, lc(C_CARD), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_provision_layer, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_set_style_radius(g_provision_layer, 10, LV_PART_MAIN);
    lv_obj_set_style_pad_all(g_provision_layer, 14, LV_PART_MAIN);
    lv_obj_set_size(g_provision_layer, 200, 130);
    lv_obj_center(g_provision_layer);

    g_lbl_prov = lv_label_create(g_provision_layer);
    lv_obj_set_style_text_color(g_lbl_prov, lc(C_WARN), LV_PART_MAIN);
    lv_obj_set_style_text_font(g_lbl_prov, &pulse_cjk_14, LV_PART_MAIN);
    lv_obj_set_style_text_align(g_lbl_prov, LV_TEXT_ALIGN_CENTER, LV_PART_MAIN);
    lv_label_set_text(g_lbl_prov, "未连接 WiFi\n\n正在开启热点...");
    lv_obj_align(g_lbl_prov, LV_ALIGN_TOP_MID, 0, 4);

    /* 关闭按钮 */
    lv_obj_t *btn_prov_close = lv_btn_create(g_provision_layer);
    lv_obj_set_size(btn_prov_close, 88, 32);
    lv_obj_align(btn_prov_close, LV_ALIGN_BOTTOM_MID, 0, 0);
    lv_obj_add_event_cb(btn_prov_close, prov_close_cb, LV_EVENT_CLICKED, nullptr);
    lv_obj_t *lbl_prov_close = lv_label_create(btn_prov_close);
    lv_obj_set_style_text_font(lbl_prov_close, &pulse_cjk_14, LV_PART_MAIN);
    lv_label_set_text(lbl_prov_close, "关闭");
    lv_obj_center(lbl_prov_close);

    lv_obj_add_flag(g_provision_layer, LV_OBJ_FLAG_HIDDEN);

    /* 设置入口按钮：右下角 */
    lv_obj_t *btn_set = lv_btn_create(g_scr);
    lv_obj_set_size(btn_set, 52, 26);
    lv_obj_align(btn_set, LV_ALIGN_BOTTOM_RIGHT, -10, -8);
    lv_obj_add_event_cb(btn_set, settings_btn_cb, LV_EVENT_CLICKED, nullptr);
    lv_obj_t *lbl_set = lv_label_create(btn_set);
    lv_obj_set_style_text_font(lbl_set, &pulse_cjk_14, LV_PART_MAIN);
    lv_label_set_text(lbl_set, "设置");
    lv_obj_center(lbl_set);

    /* 设置页覆盖层（默认隐藏）：亮度滑块 + 关闭 */
    g_settings_layer = lv_obj_create(g_scr);
    lv_obj_remove_style_all(g_settings_layer);
    lv_obj_set_style_bg_color(g_settings_layer, lc(C_CARD), LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_settings_layer, LV_OPA_COVER, LV_PART_MAIN);
    lv_obj_set_style_radius(g_settings_layer, 12, LV_PART_MAIN);
    lv_obj_set_style_pad_all(g_settings_layer, 16, LV_PART_MAIN);
    lv_obj_set_size(g_settings_layer, 200, 180);
    lv_obj_center(g_settings_layer);

    lv_obj_t *lbl_title = lv_label_create(g_settings_layer);
    lv_label_set_text(lbl_title, "设置");
    lv_obj_set_style_text_font(lbl_title, &pulse_cjk_16, LV_PART_MAIN);
    lv_obj_set_style_text_color(lbl_title, lc(C_ACCENT), LV_PART_MAIN);
    lv_obj_align(lbl_title, LV_ALIGN_TOP_LEFT, 0, 0);

    lv_obj_t *lbl_bl = lv_label_create(g_settings_layer);
    lv_obj_set_style_text_font(lbl_bl, &pulse_cjk_14, LV_PART_MAIN);
    lv_label_set_text(lbl_bl, "亮度");
    lv_obj_set_style_text_color(lbl_bl, lc(C_DIM), LV_PART_MAIN);
    lv_obj_align(lbl_bl, LV_ALIGN_TOP_LEFT, 0, 30);

    g_lbl_backlight_val = lv_label_create(g_settings_layer);
    lv_label_set_text_fmt(g_lbl_backlight_val, "%u%%", g_cur_backlight);
    lv_obj_set_style_text_color(g_lbl_backlight_val, lc(C_TEXT), LV_PART_MAIN);
    lv_obj_align(g_lbl_backlight_val, LV_ALIGN_TOP_RIGHT, 0, 30);

    g_slider_backlight = lv_slider_create(g_settings_layer);
    lv_obj_set_width(g_slider_backlight, 168);
    lv_obj_align(g_slider_backlight, LV_ALIGN_TOP_MID, 0, 58);
    lv_slider_set_range(g_slider_backlight, 5, 100);
    lv_slider_set_value(g_slider_backlight, g_cur_backlight, LV_ANIM_OFF);
    lv_obj_add_event_cb(g_slider_backlight, backlight_slider_cb, LV_EVENT_VALUE_CHANGED, nullptr);
    lv_obj_add_event_cb(g_slider_backlight, backlight_slider_cb, LV_EVENT_RELEASED, nullptr);

    /* 底部两个按钮横排：WiFi（重新配网）+ Close */
    lv_obj_t *btn_wifi = lv_btn_create(g_settings_layer);
    lv_obj_set_size(btn_wifi, 78, 32);
    lv_obj_align(btn_wifi, LV_ALIGN_BOTTOM_LEFT, 0, 0);
    lv_obj_add_event_cb(btn_wifi, wifi_btn_cb, LV_EVENT_CLICKED, nullptr);
    lv_obj_t *lbl_wifi = lv_label_create(btn_wifi);
    lv_obj_set_style_text_font(lbl_wifi, &pulse_cjk_14, LV_PART_MAIN);
    lv_label_set_text(lbl_wifi, "WiFi");
    lv_obj_center(lbl_wifi);

    lv_obj_t *btn_close = lv_btn_create(g_settings_layer);
    lv_obj_set_size(btn_close, 78, 32);
    lv_obj_align(btn_close, LV_ALIGN_BOTTOM_RIGHT, 0, 0);
    lv_obj_add_event_cb(btn_close, settings_close_cb, LV_EVENT_CLICKED, nullptr);
    lv_obj_t *lbl_close = lv_label_create(btn_close);
    lv_obj_set_style_text_font(lbl_close, &pulse_cjk_14, LV_PART_MAIN);
    lv_label_set_text(lbl_close, "关闭");
    lv_obj_center(lbl_close);

    lv_obj_add_flag(g_settings_layer, LV_OBJ_FLAG_HIDDEN);

    g_cache_init = false;
    memset(&g_cache, 0, sizeof(g_cache));

    log_i("%s: 界面构建完成", TAG);
}

/* ─────────── 状态刷新 ─────────── */

void pulse_ui_update(const pulse_ui_state_t *st) {
    if (!g_scr || !st) return;

    char buf[64];

    /* 时间 */
    if (st->time_valid) {
        snprintf(buf, sizeof(buf), "%02d:%02d", st->hour, st->minute);
    } else {
        snprintf(buf, sizeof(buf), "--:--");
    }
    update_if_changed(g_lbl_time, buf, g_cache.time, sizeof(g_cache.time));

    /* WiFi 信号格：断网全灰，按 RSSI 亮 1~4 格 */
    int wifi_level = 0;
    if (st->wifi_connected) {
        if (st->rssi > -60) wifi_level = 4;
        else if (st->rssi > -70) wifi_level = 3;
        else if (st->rssi > -80) wifi_level = 2;
        else wifi_level = 1;
    }
    for (int i = 0; i < 4; i++) {
        lv_obj_set_style_bg_color(g_wifi_bars[i], (i < wifi_level) ? lc(C_TEXT) : lc(C_DIM),
                                  LV_PART_MAIN);
    }

    /* 电池：填充宽度 + 百分比文字，颜色按电量分档 */
    if (st->battery_present && st->battery_percent >= 0) {
        int pct = st->battery_percent;
        if (pct > 100) pct = 100;
        if (pct < 0) pct = 0;
        lv_obj_set_width(g_batt_fill, 24 * pct / 100);
        lv_color_t bc = (pct > 20) ? lc(C_OK) : (pct > 10) ? lc(C_WARN) : lc(C_ERR);
        lv_obj_set_style_bg_color(g_batt_fill, bc, LV_PART_MAIN);
        lv_obj_set_style_bg_opa(g_batt_fill, LV_OPA_COVER, LV_PART_MAIN);
        snprintf(buf, sizeof(buf), "%d%%%s", pct, st->charging ? "+" : "");
    } else {
        lv_obj_set_width(g_batt_fill, 0);
        snprintf(buf, sizeof(buf), "--%%");
    }
    update_if_changed(g_lbl_batt, buf, g_cache.batt, sizeof(g_cache.batt));

    /* 录音状态 */
    const char *rs = rec_state_label(st->rec_state);
    update_if_changed(g_lbl_rec_state, rs, g_cache.rec_state, sizeof(g_cache.rec_state));

    /* 录音卡片背景随状态变色 + 脉冲圆点动画 */
    lv_color_t card_color = (st->rec_state == UI_REC_SPEECH) ? lc(0x3A1A1A) : lc(C_CARD);
    lv_obj_set_style_bg_color(g_card_rec, card_color, LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_card_rec, LV_OPA_COVER, LV_PART_MAIN);
    lv_color_t state_color = (st->rec_state == UI_REC_SPEECH) ? lc(C_REC) : lc(C_TEXT);
    lv_obj_set_style_text_color(g_lbl_rec_state, state_color, LV_PART_MAIN);
    if (st->rec_state == UI_REC_SPEECH) {
        lv_obj_set_style_bg_color(g_rec_dot, lc(C_REC), LV_PART_MAIN);
        lv_obj_set_style_bg_opa(g_rec_dot, LV_OPA_COVER, LV_PART_MAIN);
        start_rec_pulse();
    } else {
        stop_rec_pulse();
        lv_obj_set_style_bg_color(g_rec_dot, lc(C_DIM), LV_PART_MAIN);
        lv_obj_set_style_bg_opa(g_rec_dot, LV_OPA_COVER, LV_PART_MAIN);
    }

    snprintf(buf, sizeof(buf), "%02u:%02u", st->rec_duration_s / 60, st->rec_duration_s % 60);
    update_if_changed(g_lbl_rec_dur, buf, g_cache.rec_dur, sizeof(g_cache.rec_dur));

    /* 上传状态 + 状态圆点 */
    const char *us = upload_state_label(st->upload_state);
    update_if_changed(g_lbl_upload_state, us, g_cache.upload_state, sizeof(g_cache.upload_state));
    lv_color_t udot;
    switch (st->upload_state) {
        case UI_UPLOAD_OFFLINE: udot = lc(C_WARN); break;
        case UI_UPLOAD_SENDING: udot = lc(C_ACCENT); break;
        case UI_UPLOAD_BACKOFF: udot = lc(C_WARN); break;
        case UI_UPLOAD_DISABLED: udot = lc(C_ERR); break;
        default: udot = lc(C_OK); break; /* IDLE = 已同步 */
    }
    lv_obj_set_style_bg_color(g_upload_dot, udot, LV_PART_MAIN);
    lv_obj_set_style_bg_opa(g_upload_dot, LV_OPA_COVER, LV_PART_MAIN);
    snprintf(buf, sizeof(buf), "队列 %u  已传 %u  失败 %u", st->pending_count, st->sent_ok,
             st->sent_failed);
    update_if_changed(g_lbl_upload_count, buf, g_cache.upload_count, sizeof(g_cache.upload_count));

    /* 存储 + 容量条 */
    const char *sl = storage_label(st->storage_kind);
    if (st->storage_kind == UI_STORAGE_SD) {
        snprintf(buf, sizeof(buf), "%s  剩余 %.1f GB", sl, st->storage_free_mb / 1024.0);
        if (st->storage_total_mb > 0) {
            int used_pct = 100 - (int)(st->storage_free_mb * 100 / st->storage_total_mb);
            if (used_pct < 0) used_pct = 0;
            if (used_pct > 100) used_pct = 100;
            lv_bar_set_value(g_bar_storage, used_pct, LV_ANIM_OFF);
        } else {
            lv_bar_set_value(g_bar_storage, 0, LV_ANIM_OFF);
        }
    } else {
        snprintf(buf, sizeof(buf), "%s（仅临时缓冲，掉电丢失）", sl);
        lv_bar_set_value(g_bar_storage, 0, LV_ANIM_OFF);
    }
    update_if_changed(g_lbl_storage, buf, g_cache.storage, sizeof(g_cache.storage));

    /* 配网提示由 app 层显式控制，这里不自动切换 */
}

void pulse_ui_show_provision_hint(bool show) {
    if (!g_provision_layer) return;
    if (show) {
        lv_obj_clear_flag(g_provision_layer, LV_OBJ_FLAG_HIDDEN);
    } else {
        lv_obj_add_flag(g_provision_layer, LV_OBJ_FLAG_HIDDEN);
    }
}

void pulse_ui_set_provision_ap(const char *ap_ssid) {
    if (!g_lbl_prov) return;
    const char *title = g_prov_switching ? "更换网络" : "未连接 WiFi";
    if (!ap_ssid || ap_ssid[0] == '\0') {
        /* AP 尚未开启（NO_CREDENTIALS 的短暂瞬间）*/
        char buf[64];
        snprintf(buf, sizeof(buf), "%s\n\n正在开启热点...", title);
        lv_label_set_text(g_lbl_prov, buf);
        return;
    }
    char buf[96];
    snprintf(buf, sizeof(buf), "%s\n\n连接热点 %s\n浏览器打开 192.168.4.1", title, ap_ssid);
    lv_label_set_text(g_lbl_prov, buf);
}

void pulse_ui_set_provision_mode(bool switching) {
    g_prov_switching = switching;
}

bool pulse_ui_provision_hint_visible(void) {
    if (!g_provision_layer) return false;
    return !lv_obj_has_flag(g_provision_layer, LV_OBJ_FLAG_HIDDEN);
}

/* ─────────── 设置页 ─────────── */

void pulse_ui_set_backlight_cb(pulse_ui_backlight_cb_t cb) {
    g_backlight_cb = cb;
}

void pulse_ui_set_wifi_cb(pulse_ui_wifi_cb_t cb) {
    g_wifi_cb = cb;
}

void pulse_ui_set_backlight(uint8_t percent) {
    if (percent < 5) percent = 5;
    if (percent > 100) percent = 100;
    g_cur_backlight = percent;
    if (g_slider_backlight) {
        lv_slider_set_value(g_slider_backlight, percent, LV_ANIM_OFF);
    }
    if (g_lbl_backlight_val) {
        lv_label_set_text_fmt(g_lbl_backlight_val, "%u%%", percent);
    }
}

void pulse_ui_open_settings(void) {
    if (!g_settings_layer) return;
    lv_obj_clear_flag(g_settings_layer, LV_OBJ_FLAG_HIDDEN);
    lv_obj_move_foreground(g_settings_layer);
}

void pulse_ui_close_settings(void) {
    if (!g_settings_layer) return;
    lv_obj_add_flag(g_settings_layer, LV_OBJ_FLAG_HIDDEN);
}

bool pulse_ui_settings_visible(void) {
    if (!g_settings_layer) return false;
    return !lv_obj_has_flag(g_settings_layer, LV_OBJ_FLAG_HIDDEN);
}

void pulse_ui_dump(void) {
    Serial.println();
    Serial.println("================ LVGL 界面 ================");
    Serial.printf("  屏幕对象  %s\n", g_scr ? "已创建" : "未创建");
    Serial.printf("  配网提示  %s\n", pulse_ui_provision_hint_visible() ? "显示中" : "隐藏");
    Serial.println("  说明：界面使用内置 Montserrat 字体（ASCII）。");
    Serial.println("        中文需用 tools/gen_font.py 生成 CJK 子集后替换 UI_FONT。");
    Serial.println("=========================================");
}
