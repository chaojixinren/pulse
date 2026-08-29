/**
 * @file pulse_touch.h
 * @brief CST816 电容触摸 HAL (I2C 0x15, INT=GPIO14, RST 与 LCD 共用 GPIO13)
 *
 * 芯片行为注意（来自 Espressif esp_lcd_touch_cst816s 的已知问题）
 * ------------------------------------------------------------
 * CST816 **只在触摸事件之后才响应 I2C**。冷启动时读 CHIP_ID 很可能直接 NAK，
 * 这是芯片特性而非接线故障。因此 touch_init() 把「读不到 ID」当作警告而非失败，
 * 否则每次上电都会误判触摸坏了。
 *
 * 同理，无触摸时的寄存器读取会失败，touch_read() 把读失败一律当作「未按下」。
 *
 * 复位顺序：必须先 display_shared_reset()（GPIO13 同时复位 LCD 与 TP），
 * 再 display_init()，最后 touch_init()。
 */
#ifndef PULSE_TOUCH_H
#define PULSE_TOUCH_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* CST816 手势码（寄存器 0x00）*/
typedef enum {
    TOUCH_GESTURE_NONE = 0x00,
    TOUCH_GESTURE_SWIPE_UP = 0x01,
    TOUCH_GESTURE_SWIPE_DOWN = 0x02,
    TOUCH_GESTURE_SWIPE_LEFT = 0x03,
    TOUCH_GESTURE_SWIPE_RIGHT = 0x04,
    TOUCH_GESTURE_SINGLE_CLICK = 0x05,
    TOUCH_GESTURE_DOUBLE_CLICK = 0x0B,
    TOUCH_GESTURE_LONG_PRESS = 0x0C,
} touch_gesture_t;

typedef struct {
    bool pressed;
    int16_t x; /* 已按 ui.rotation 转换为屏幕逻辑坐标 */
    int16_t y;
    uint8_t gesture; /* touch_gesture_t */
} touch_data_t;

/**
 * 初始化触摸。要求 display_shared_reset() 已执行过。
 * @return 始终返回 true（除 I2C 总线本身不可用）。读不到 CHIP_ID 只告警，
 *         因为 CST816 在无触摸时本就不响应。
 */
bool touch_init(void);

bool touch_is_ready(void);

/** 读取当前触摸状态。无触摸或 I2C 无响应时 out->pressed = false。 */
bool touch_read(touch_data_t *out);

/** 最近一次成功读到的 CHIP_ID；0 表示从未读到 */
uint8_t touch_chip_id(void);

/** CHIP_ID 对应的型号名 */
const char *touch_chip_name(uint8_t id);

/** 手势名，用于日志与调试 */
const char *touch_gesture_str(uint8_t g);

/** 让触摸芯片进入低功耗（深睡前调用）*/
void touch_sleep(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_TOUCH_H */
