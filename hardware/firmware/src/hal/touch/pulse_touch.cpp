/**
 * @file pulse_touch.cpp
 * @brief CST816 电容触摸实现
 */
#include "pulse_touch.h"

#include <Arduino.h>
#include <Wire.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "tp";

/* CST816 寄存器（与 esphome cst816 驱动一致，已核对） */
#define CST816_REG_STATUS 0x00      /* 手势码 */
#define CST816_REG_TOUCH_NUM 0x02   /* 手指数 */
#define CST816_REG_XPOS_HIGH 0x03   /* 低 4 位为 X 的高 4 位 */
#define CST816_REG_XPOS_LOW 0x04
#define CST816_REG_YPOS_HIGH 0x05
#define CST816_REG_YPOS_LOW 0x06
#define CST816_REG_CHIP_ID 0xA7
#define CST816_REG_SLEEP 0xE5
#define CST816_REG_IRQ_CTL 0xFA
#define CST816_REG_DIS_AUTOSLEEP 0xFE

#define CST816_IRQ_EN_MOTION 0x70
#define CST816_SLEEP_CMD 0x03

static bool g_ready = false;
static uint8_t g_chip_id = 0;

/* INT 引脚由 ISR 置标志。CST816 只在触摸时拉低 INT，
 * 用它做快速预判可以避免无触摸时高频空跑 I2C。 */
static volatile bool g_int_flag = false;

static void IRAM_ATTR touch_int_handler(void) {
    g_int_flag = true;
}

/* 读寄存器。CST816 无触摸时不响应 I2C 是正常现象，故失败不打日志。 */
static bool reg_read(uint8_t reg, uint8_t *buf, size_t len) {
    Wire.beginTransmission(I2C_ADDR_CST816);
    Wire.write(reg);
    if (Wire.endTransmission(false) != 0) return false; /* 保持总线，接着读 */

    size_t got = Wire.requestFrom((uint8_t)I2C_ADDR_CST816, (uint8_t)len);
    if (got != len) return false;
    for (size_t i = 0; i < len; i++) buf[i] = Wire.read();
    return true;
}

static bool reg_write(uint8_t reg, uint8_t val) {
    Wire.beginTransmission(I2C_ADDR_CST816);
    Wire.write(reg);
    Wire.write(val);
    return Wire.endTransmission() == 0;
}

bool touch_init(void) {
    /* I2C 可能已由 PMU 初始化过；Wire.begin 可重复调用。
     * 100kHz：CST816 在 400kHz 下偶发异常，与 PMU/RTC 共线时统一用 100k。 */
    Wire.begin(PIN_I2C_SDA, PIN_I2C_SCL, 100000U);

    /* 注意：不在此处复位。GPIO13 由 display_shared_reset() 统一管理，
     * 这里若再复位一次会把已初始化好的 LCD 一起打掉。 */

    /* 尝试读 CHIP_ID。CST816 冷启动无触摸时通常 NAK，属正常，多试几次。 */
    for (int i = 0; i < 3; i++) {
        uint8_t id = 0;
        if (reg_read(CST816_REG_CHIP_ID, &id, 1) && id != 0x00 && id != 0xFF) {
            g_chip_id = id;
            break;
        }
        delay(20);
    }

    if (g_chip_id) {
        log_i("%s: CST816 CHIP_ID=0x%02X (%s)", TAG, g_chip_id, touch_chip_name(g_chip_id));
    } else {
        /* 关键：不返回 false。芯片在无触摸时不应答是设计如此，
         * 触摸一下就会开始正常响应。 */
        log_w("%s: 未读到 CHIP_ID（CST816 无触摸时不响应 I2C，属正常）", TAG);
    }

    /* 关闭自动休眠：否则一段时间无操作后触摸失灵，需要按键唤醒。
     * 写入失败同样不阻塞初始化。 */
    reg_write(CST816_REG_DIS_AUTOSLEEP, 0x01);
    reg_write(CST816_REG_IRQ_CTL, CST816_IRQ_EN_MOTION);

    pinMode(PIN_TP_INT, INPUT_PULLUP);
    attachInterrupt(digitalPinToInterrupt(PIN_TP_INT), touch_int_handler, FALLING);

    g_ready = true;
    log_i("%s: 触摸就绪 (INT=GPIO%d)", TAG, PIN_TP_INT);
    return true;
}

bool touch_is_ready(void) {
    return g_ready;
}

uint8_t touch_chip_id(void) {
    return g_chip_id;
}

const char *touch_chip_name(uint8_t id) {
    switch (id) {
        case 0xB4: return "CST816S";
        case 0xB5: return "CST816T";
        case 0xB6: return "CST816D";
        case 0xB7: return "CST820";
        case 0x11: return "CST826";
        case 0x13: return "CST836";
        case 0x20: return "CST716";
        default: return "未知型号";
    }
}

const char *touch_gesture_str(uint8_t g) {
    switch (g) {
        case TOUCH_GESTURE_NONE: return "无";
        case TOUCH_GESTURE_SWIPE_UP: return "上滑";
        case TOUCH_GESTURE_SWIPE_DOWN: return "下滑";
        case TOUCH_GESTURE_SWIPE_LEFT: return "左滑";
        case TOUCH_GESTURE_SWIPE_RIGHT: return "右滑";
        case TOUCH_GESTURE_SINGLE_CLICK: return "单击";
        case TOUCH_GESTURE_DOUBLE_CLICK: return "双击";
        case TOUCH_GESTURE_LONG_PRESS: return "长按";
        default: return "其他";
    }
}

/* 把面板原始坐标按 rotation 映射到屏幕逻辑坐标。
 * 触摸面板的原生方向与 rotation=0 一致，其余角度做相应变换。 */
static void map_rotation(uint16_t raw_x, uint16_t raw_y, int16_t *out_x, int16_t *out_y) {
    const pulse_config_t *c = config_get();
    switch (c->ui.rotation) {
        case 1: /* 顺时针 90 度，逻辑分辨率变为 280x240 */
            *out_x = (int16_t)raw_y;
            *out_y = (int16_t)(LCD_H_RES - 1 - raw_x);
            break;
        case 2: /* 180 度 */
            *out_x = (int16_t)(LCD_H_RES - 1 - raw_x);
            *out_y = (int16_t)(LCD_V_RES - 1 - raw_y);
            break;
        case 3: /* 顺时针 270 度 */
            *out_x = (int16_t)(LCD_V_RES - 1 - raw_y);
            *out_y = (int16_t)raw_x;
            break;
        default:
            *out_x = (int16_t)raw_x;
            *out_y = (int16_t)raw_y;
            break;
    }
}

bool touch_read(touch_data_t *out) {
    if (!g_ready || !out) return false;

    out->pressed = false;
    out->x = 0;
    out->y = 0;
    out->gesture = TOUCH_GESTURE_NONE;

    /* 一次连续读出 0x00~0x06，减少 I2C 事务数。
     * buf[0]=手势 buf[2]=手指数 buf[3..6]=坐标 */
    uint8_t buf[7] = {0};
    if (!reg_read(CST816_REG_STATUS, buf, sizeof(buf))) {
        /* 无触摸时读失败属正常，直接报告未按下。 */
        g_int_flag = false;
        return true;
    }

    /* 顺带补齐 CHIP_ID：初始化时可能没读到，这里有机会就补上。 */
    if (g_chip_id == 0) {
        uint8_t id = 0;
        if (reg_read(CST816_REG_CHIP_ID, &id, 1) && id != 0x00 && id != 0xFF) {
            g_chip_id = id;
            log_i("%s: 触摸后读到 CHIP_ID=0x%02X (%s)", TAG, id, touch_chip_name(id));
        }
    }

    uint8_t fingers = buf[2] & 0x0F;
    out->gesture = buf[0];

    if (fingers == 0) {
        g_int_flag = false;
        return true;
    }

    /* 12 位坐标：高字节只取低 4 位。 */
    uint16_t raw_x = (uint16_t)(((buf[3] & 0x0F) << 8) | buf[4]);
    uint16_t raw_y = (uint16_t)(((buf[5] & 0x0F) << 8) | buf[6]);

    /* 越界值说明读到了脏数据（常见于 I2C 抢线），丢弃。 */
    if (raw_x >= LCD_H_RES || raw_y >= LCD_V_RES) {
        return true;
    }

    map_rotation(raw_x, raw_y, &out->x, &out->y);
    out->pressed = true;
    return true;
}

void touch_sleep(void) {
    if (!g_ready) return;
    reg_write(CST816_REG_SLEEP, CST816_SLEEP_CMD);
}
