/**
 * @file board_pins.h
 * @brief Pulse 硬件板级引脚定义（唯一真源）
 *
 * 本文件只描述"焊死在 PCB 上、运行时不可改"的连接关系。
 * 所有可调参数（采样率、分段阈值、超时、服务器地址）一律放 config.json，
 * 不在此处硬编码。
 *
 * 硬件: ESP32-S3R8 + W25Q128JVPIQ
 * 屏幕: P169H002-CTP (1.69" 240x280 ST7789V + CST816 电容触摸)
 * 麦克风: MSM261D4030H1CPM (PDM 数字输出)
 * PMU: AXP2101   RTC: BM8563EMA
 */
#ifndef PULSE_BOARD_PINS_H
#define PULSE_BOARD_PINS_H

/* ───────────────── 按键 ───────────────── */
#define PIN_BTN_BOOT 0 /* BOOT 键拉低接地；同时是 strapping 引脚 */

/* ───────────────── 麦克风 (I2S0 PDM RX) ───────────────── */
#define PIN_MIC_DATA 1
#define PIN_MIC_CLK 2

/* ───────────────── I2C 总线 (AXP2101 + BM8563 + CST816 共用) ───────────────── */
#define PIN_I2C_SCL 4
#define PIN_I2C_SDA 5

/* ───────────────── 电源管理 AXP2101 ───────────────── */
#define PIN_PMU_IRQ 6   /* 低电平有效，下降沿中断 */
#define PIN_PMU_PWRON 7 /* PWR 键拉低接地 */

/* ───────────────── 显示 ST7789 (SPI2 / FSPI) ───────────────── */
#define PIN_LCD_BL 8   /* 经 AO3400A 驱动 LEDK，LEDC PWM 调背光 */
#define PIN_LCD_DC 9
#define PIN_LCD_CS 10
#define PIN_LCD_SCLK 11
#define PIN_LCD_MOSI 12
#define PIN_LCD_RST 13 /* 与触摸屏 RST 共用同一引脚 */

/* ───────────────── 触摸 CST816 ───────────────── */
#define PIN_TP_RST 13 /* 复用 PIN_LCD_RST：复位时序必须同时兼顾两颗芯片 */
#define PIN_TP_INT 14 /* 低电平有效 */

/* ───────────────── TF 卡 (SDMMC 4-bit) ───────────────── */
#define PIN_SD_D0 45
#define PIN_SD_D1 46
#define PIN_SD_D2 39
#define PIN_SD_D3 40
#define PIN_SD_CMD 41
#define PIN_SD_CLK 42

/* ───────────────── USB ───────────────── */
#define PIN_USB_DN 19
#define PIN_USB_DP 20

/* ───────────────── I2C 从机地址 ───────────────── */
#define I2C_ADDR_AXP2101 0x34
#define I2C_ADDR_BM8563 0x51
#define I2C_ADDR_CST816 0x15

/* ───────────────── 屏幕几何 ───────────────── */
/* ST7789 内部 GRAM 为 240x320，而面板只有 240x280，
 * 因此纵向需要 20 像素偏移把可视区居中。offset 值需 M2 实测确认。 */
#define LCD_H_RES 240
#define LCD_V_RES 280
#define LCD_X_OFFSET 0
#define LCD_Y_OFFSET 20

/* ───────────────── 音频硬约束 ───────────────── */
/* 后端 stt.go 把格式写死为 pcm_s16le / 16000Hz / 16bit / mono，
 * 这三个值不是可调参数，改了云端就转写不出来。 */
#define AUDIO_SAMPLE_RATE 16000
#define AUDIO_BITS_PER_SAMPLE 16
#define AUDIO_CHANNELS 1
#define AUDIO_BYTES_PER_SEC (AUDIO_SAMPLE_RATE * (AUDIO_BITS_PER_SAMPLE / 8) * AUDIO_CHANNELS)

/* ═════════════════ 编译期自检 ═════════════════
 * 目的：把"引脚踩了保留区"或"两个外设抢同一脚"这类错误挡在编译阶段，
 * 而不是等到板子上电才发现。
 */
#ifdef __cplusplus

/* ESP32-S3R8 保留引脚：
 *   GPIO26~32 → SPI0，接封装外 W25Q128 flash
 *   GPIO33~37 → octal PSRAM 专用，八线模式下完全不可他用
 * 参考 Espressif 硬件设计指南 "In cases where 1.8V or 3.3V, octal,
 * in-package or off-package SPI flash/PSRAM is used, GPIO33 ~ GPIO37
 * are occupied and cannot be used for other functions."
 */
constexpr bool pin_is_reserved(int p) {
    return (p >= 26 && p <= 37);
}

/* 参与冲突检测的引脚清单。GPIO13 被 LCD_RST 与 TP_RST 共用，
 * 属于有意设计，故只登记一次。 */
constexpr int kPulsePins[] = {
    PIN_BTN_BOOT, PIN_MIC_DATA, PIN_MIC_CLK, PIN_I2C_SCL, PIN_I2C_SDA,
    PIN_PMU_IRQ,  PIN_PMU_PWRON, PIN_LCD_BL, PIN_LCD_DC, PIN_LCD_CS,
    PIN_LCD_SCLK, PIN_LCD_MOSI, PIN_LCD_RST, PIN_TP_INT,
    PIN_SD_D0,    PIN_SD_D1,    PIN_SD_D2,  PIN_SD_D3,  PIN_SD_CMD, PIN_SD_CLK,
    PIN_USB_DN,   PIN_USB_DP,
};
constexpr int kPulsePinCount = sizeof(kPulsePins) / sizeof(kPulsePins[0]);

constexpr bool pins_has_duplicate(int i = 0, int j = 1) {
    return i >= kPulsePinCount       ? false
           : j >= kPulsePinCount     ? pins_has_duplicate(i + 1, i + 2)
           : kPulsePins[i] == kPulsePins[j] ? true
                                     : pins_has_duplicate(i, j + 1);
}

constexpr bool pins_any_reserved(int i = 0) {
    return i >= kPulsePinCount ? false
           : pin_is_reserved(kPulsePins[i]) ? true
                                            : pins_any_reserved(i + 1);
}

static_assert(!pins_has_duplicate(), "board_pins.h: 存在重复引脚定义（GPIO13 共用除外）");
static_assert(!pins_any_reserved(), "board_pins.h: 有引脚落在 GPIO26~37 保留区（flash / octal PSRAM 专用）");
static_assert(LCD_V_RES + LCD_Y_OFFSET <= 320, "ST7789 GRAM 纵向溢出：V_RES + Y_OFFSET 不得超过 320");
static_assert(LCD_H_RES + LCD_X_OFFSET <= 240, "ST7789 GRAM 横向溢出：H_RES + X_OFFSET 不得超过 240");
static_assert(AUDIO_BYTES_PER_SEC == 32000, "音频码率应为 32000 B/s（16kHz/16bit/mono）");

#endif /* __cplusplus */

#endif /* PULSE_BOARD_PINS_H */
