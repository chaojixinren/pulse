/**
 * @file lv_conf.h
 * @brief LVGL 8.4 配置（Pulse 项目）
 *
 * 只覆盖需要偏离默认值的选项；未定义的项由 lv_conf_internal.h 提供默认值。
 * 通过 build_flags 的 -DLV_CONF_INCLUDE_SIMPLE 与 -Iinclude 被找到。
 */
#ifndef LV_CONF_H
#define LV_CONF_H

#include <stdint.h>

/* ───────────── 颜色 ───────────── */
/* ST7789 使用 RGB565 */
#define LV_COLOR_DEPTH 16
/* 必须为 1。ST7789 走 SPI 时先收到的字节被当作 RGB565 的高字节，
 * 而 LVGL 默认按小端存 uint16（低字节在前），不交换会导致红蓝错乱。
 * 与 pulse_display.cpp 的 display_color565() 保持同一约定：
 * 显示层统一接收「线序（大端）RGB565」。
 * 若 M2/M3 实测颜色正常而改此值后异常，说明面板走了非典型字节序，改回 0。 */
#define LV_COLOR_16_SWAP 1

/* ───────────── 内存 ───────────── */
/* 用 C 标准库 malloc，从而能利用 PSRAM（Arduino 默认 heap 已含 PSRAM 时）。
 * 自带 allocator 只能用内部 RAM，对 LVGL 对象较多的界面容易吃紧。 */
#define LV_MEM_CUSTOM 1
#define LV_MEM_CUSTOM_INCLUDE <stdlib.h>
#define LV_MEM_CUSTOM_ALLOC malloc
#define LV_MEM_CUSTOM_FREE free
#define LV_MEM_CUSTOM_REALLOC realloc

/* ───────────── 时钟与刷新 ───────────── */
/* 由 esp_timer 周期调用 lv_tick_inc()，故不用 LVGL 自带的 tick 获取 */
#define LV_TICK_CUSTOM 0
#define LV_DISP_DEF_REFR_PERIOD 30
#define LV_INDEV_DEF_READ_PERIOD 30

/* 屏幕 DPI：1.69" 240x280 约为 216 DPI，影响默认控件尺寸 */
#define LV_DPI_DEF 216

/* ───────────── 渲染 ───────────── */
#define LV_DRAW_COMPLEX 1
#define LV_SHADOW_CACHE_SIZE 0
#define LV_IMG_CACHE_DEF_SIZE 0

/* ───────────── 日志 ───────────── */
/* 打开警告级日志，接到 Serial 便于排查；正式发布可降为 NONE */
#define LV_USE_LOG 1
#define LV_LOG_LEVEL LV_LOG_LEVEL_WARN
#define LV_LOG_PRINTF 0

/* ───────────── 断言 ───────────── */
/* 内存不足与空指针必须抓到，否则表现为随机花屏难以定位 */
#define LV_USE_ASSERT_NULL 1
#define LV_USE_ASSERT_MALLOC 1
#define LV_USE_ASSERT_STYLE 0
#define LV_USE_ASSERT_MEM_INTEGRITY 0
#define LV_USE_ASSERT_OBJ 0

/* ───────────── 字体 ───────────── */
/* 内置 Montserrat 仅含 ASCII。中文界面需要在 M10 阶段用
 * lv_font_conv 生成中文子集字体，不能依赖内置字体。 */
#define LV_FONT_MONTSERRAT_12 1
#define LV_FONT_MONTSERRAT_14 1
#define LV_FONT_MONTSERRAT_16 1
#define LV_FONT_MONTSERRAT_20 1
#define LV_FONT_MONTSERRAT_28 1
#define LV_FONT_DEFAULT &lv_font_montserrat_14

/* ───────────── 控件 ───────────── */
#define LV_USE_ARC 1
#define LV_USE_BAR 1
#define LV_USE_BTN 1
#define LV_USE_BTNMATRIX 1
#define LV_USE_CANVAS 0
#define LV_USE_CHECKBOX 1
#define LV_USE_DROPDOWN 1
#define LV_USE_IMG 1
#define LV_USE_LABEL 1
#define LV_USE_LINE 1
#define LV_USE_ROLLER 1
#define LV_USE_SLIDER 1
#define LV_USE_SWITCH 1
#define LV_USE_TEXTAREA 1
#define LV_USE_TABLE 0

/* 布局：Flex 用于状态栏与列表排布 */
#define LV_USE_FLEX 1
#define LV_USE_GRID 1

/* 主题 */
#define LV_USE_THEME_DEFAULT 1
#define LV_THEME_DEFAULT_DARK 1
#define LV_THEME_DEFAULT_GROW 1

/* ───────────── 示例与 demo ─────────────
 * 关闭以节省 flash；M3 的触摸验证用自写界面，不依赖官方 demo。 */
#define LV_BUILD_EXAMPLES 0
#define LV_USE_DEMO_WIDGETS 0

#endif /* LV_CONF_H */
