/**
 * @file pulse_display.h
 * @brief ST7789 显示屏 HAL (P169H002-CTP, 240x280, SPI2/FSPI)
 *
 * 共用复位引脚的处理（重要）
 * --------------------------
 * GPIO13 同时接 LCD_RST 与 TP_RST。若把它交给 esp_lcd 的 reset_gpio_num，
 * 那么 esp_lcd_panel_reset() 会连带把 CST816 触摸芯片一起复位，而触摸驱动
 * 并不知情 —— 表现为触摸初始化后莫名失效。
 *
 * 因此本模块把 panel 的 reset_gpio_num 设为 -1，改由 display_shared_reset()
 * 显式复位，并要求上层遵守顺序：
 *      display_shared_reset();   // 一次复位两颗芯片
 *      display_init();           // 屏幕初始化
 *      touch_init();             // 触摸初始化必须在此之后
 *
 * 几何偏移
 * --------
 * ST7789 内部 GRAM 为 240x320，面板可视区只有 240x280，需要 y 方向偏移把
 * 可视区对齐。LCD_Y_OFFSET 的默认值 20 需要 M2 测试实测确认。
 */
#ifndef PULSE_DISPLAY_H
#define PULSE_DISPLAY_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * 打包 RGB565，返回值为**线序（大端在前）**，可直接送显。
 *
 * 为什么要交换字节：esp_lcd 按内存顺序把缓冲发出去，而 ST7789 在 16 位模式下
 * 期望先收到高字节（R5G3）。ESP32 是小端，本机 uint16 的 0xF800（红）在内存里
 * 是 {0x00, 0xF8}，先发 0x00 会被面板当成高字节，红色就显示成蓝色。
 * 因此这里预先交换，与 lv_conf.h 的 LV_COLOR_16_SWAP=1 保持同一约定。
 */
uint16_t display_color565(uint8_t r, uint8_t g, uint8_t b);

/**
 * 复位 LCD 与触摸芯片（共用 GPIO13）。
 * 必须在 display_init() 与 touch_init() 之前调用，且只调用一次。
 */
void display_shared_reset(void);

/**
 * 初始化 SPI 总线 + ST7789 面板 + 背光 PWM。
 * @return false 表示 SPI 总线或面板初始化失败
 */
bool display_init(void);

bool display_is_ready(void);

/**
 * 向指定矩形推送像素。坐标为面板可视区坐标（已自动加上 GRAM 偏移）。
 * @param x2,y2 为闭区间右下角
 */
bool display_blit(int x1, int y1, int x2, int y2, const uint16_t *pixels);

/** 用单色填充整屏 */
bool display_fill(uint16_t color);

/** 设置背光亮度 0~100（LEDC PWM，经 AO3400A 驱动 LEDK）*/
void display_set_backlight(uint8_t percent);

/** 当前背光亮度 */
uint8_t display_get_backlight(void);

/** 打开/关闭显示（省电用，比关背光更彻底）*/
void display_on_off(bool on);

/**
 * 异步送显：提交后立即返回，不等 DMA 完成。
 * 供 LVGL 使用 —— LVGL 有自己的双缓冲机制，等待由 flush_ready 回调驱动。
 * 普通代码请用 display_blit()（阻塞语义，返回后可安全复用缓冲）。
 */
bool display_blit_async(int x1, int y1, int x2, int y2, const uint16_t *pixels);

/**
 * 注册 DMA 送显完成回调（仅对 display_blit_async 生效）。
 * 在 ISR 上下文被调用，回调内不得阻塞；返回 true 表示唤醒了高优先级任务。
 * 传 NULL 可注销。
 *
 * 这样设计是为了让 HAL 层不依赖 LVGL：lv_port 注册自己的回调，
 * 而 M2 等不含 LVGL 的测试环境依然能独立编译。
 */
typedef bool (*display_flush_done_cb_t)(void *user_ctx);
void display_set_flush_done_cb(display_flush_done_cb_t cb, void *user_ctx);

/** 面板可视区宽高（考虑 rotation 后的逻辑尺寸）*/
int display_width(void);
int display_height(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_DISPLAY_H */
