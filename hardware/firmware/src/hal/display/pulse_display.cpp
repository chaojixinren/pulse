/**
 * @file pulse_display.cpp
 * @brief ST7789 显示驱动实现（esp_lcd + LEDC 背光）
 */
#include "pulse_display.h"

#include <Arduino.h>
#include <driver/gpio.h>
#include <driver/ledc.h>
#include <driver/spi_master.h>
#include <esp_lcd_panel_io.h>
#include <esp_lcd_panel_ops.h>
#include <esp_lcd_panel_vendor.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "lcd";

/* 背光 PWM：13 位分辨率在 5kHz 下足够细腻，且远离音频频段避免啸叫。 */
#define BL_LEDC_TIMER LEDC_TIMER_0
#define BL_LEDC_CHANNEL LEDC_CHANNEL_0
#define BL_LEDC_MODE LEDC_LOW_SPEED_MODE
#define BL_LEDC_RES LEDC_TIMER_13_BIT
#define BL_LEDC_FREQ_HZ 5000
#define BL_DUTY_MAX ((1u << 13) - 1u)

/* SPI 时钟。ST7789 规格上限约 80MHz，但飞线/长排线容易花屏，
 * 40MHz 是稳定与刷新率的折中。240x280x2B = 134KB/帧，40MHz 下约 27ms。 */
#define LCD_PIXEL_CLOCK_HZ (40 * 1000 * 1000)
#define LCD_SPI_HOST SPI2_HOST

static esp_lcd_panel_io_handle_t g_io = nullptr;
static esp_lcd_panel_handle_t g_panel = nullptr;
static bool g_ready = false;
static uint8_t g_backlight = 0;
static bool g_swap_xy = false;

/* 送显完成处理。esp_lcd 配了 trans_queue 后 draw_bitmap 是异步返回的，
 * 调用方若立刻改写缓冲，DMA 会读到新数据造成撕裂。
 *
 * 两条路径：
 *   display_blit()       阻塞语义，用信号量等 DMA 完成，返回后可复用缓冲
 *   display_blit_async() 立即返回，完成时调用注册的回调（LVGL 用）
 *
 * HAL 层不认识 LVGL：异步回调由 lv_port 通过 display_set_flush_done_cb()
 * 注册进来，这样 M2 这类不含 LVGL 的测试环境仍能独立编译。
 *
 * 这些定义必须位于 display_init() 之前 —— 初始化时要把回调填进 io_cfg。 */
static SemaphoreHandle_t g_trans_done = nullptr;
static volatile bool g_blocking_blit = false;
static display_flush_done_cb_t g_flush_cb = nullptr;
static void *g_flush_ctx = nullptr;

static bool IRAM_ATTR on_color_trans_done(esp_lcd_panel_io_handle_t io,
                                          esp_lcd_panel_io_event_data_t *edata, void *user_ctx) {
    (void)io;
    (void)edata;
    (void)user_ctx;
    if (g_blocking_blit) {
        if (g_trans_done) {
            BaseType_t woken = pdFALSE;
            xSemaphoreGiveFromISR(g_trans_done, &woken);
            return woken == pdTRUE;
        }
        return false;
    }
    if (g_flush_cb) return g_flush_cb(g_flush_ctx);
    return false;
}

void display_set_flush_done_cb(display_flush_done_cb_t cb, void *user_ctx) {
    g_flush_cb = cb;
    g_flush_ctx = user_ctx;
}

uint16_t display_color565(uint8_t r, uint8_t g, uint8_t b) {
    /* 先算出标准 RGB565，再交换高低字节得到线序。
     * 交换的原因见 pulse_display.h 的注释：ESP32 小端 + ST7789 先收高字节。 */
    uint16_t c = (uint16_t)(((r & 0xF8) << 8) | ((g & 0xFC) << 3) | (b >> 3));
    return (uint16_t)((c >> 8) | (c << 8));
}

void display_shared_reset(void) {
    /* GPIO13 同时接 LCD_RST 与 TP_RST，两颗芯片都是低电平复位。
     * ST7789 要求复位脉冲 >=10us、复位后等待 >=120ms；
     * CST816 要求复位后等待约 50ms。取两者较严值。 */
    gpio_config_t io_conf = {};
    io_conf.pin_bit_mask = 1ULL << PIN_LCD_RST;
    io_conf.mode = GPIO_MODE_OUTPUT;
    io_conf.pull_up_en = GPIO_PULLUP_DISABLE;
    io_conf.pull_down_en = GPIO_PULLDOWN_DISABLE;
    io_conf.intr_type = GPIO_INTR_DISABLE;
    gpio_config(&io_conf);

    gpio_set_level((gpio_num_t)PIN_LCD_RST, 1);
    delay(10);
    gpio_set_level((gpio_num_t)PIN_LCD_RST, 0);
    delay(20); /* 远超 10us 最小值，兼顾 CST816 */
    gpio_set_level((gpio_num_t)PIN_LCD_RST, 1);
    delay(150); /* ST7789 >=120ms，CST816 >=50ms */

    log_i("%s: 共用复位完成 (GPIO%d, LCD+TP)", TAG, PIN_LCD_RST);
}

static void backlight_init(void) {
    ledc_timer_config_t tcfg = {};
    tcfg.speed_mode = BL_LEDC_MODE;
    tcfg.duty_resolution = BL_LEDC_RES;
    tcfg.timer_num = BL_LEDC_TIMER;
    tcfg.freq_hz = BL_LEDC_FREQ_HZ;
    tcfg.clk_cfg = LEDC_AUTO_CLK;
    ledc_timer_config(&tcfg);

    ledc_channel_config_t ccfg = {};
    ccfg.gpio_num = PIN_LCD_BL;
    ccfg.speed_mode = BL_LEDC_MODE;
    ccfg.channel = BL_LEDC_CHANNEL;
    ccfg.timer_sel = BL_LEDC_TIMER;
    ccfg.duty = 0; /* 从全暗开始，避免初始化过程中的花屏被看到 */
    ccfg.hpoint = 0;
    ledc_channel_config(&ccfg);
}

void display_set_backlight(uint8_t percent) {
    if (!g_ready) return;
    if (percent > 100) percent = 100;
    g_backlight = percent;
    /* AO3400A 是 N 沟道 MOS 低边驱动 LEDK：GPIO 高 -> 导通 -> 亮，
     * 因此占空比与亮度正相关，无需反相。 */
    uint32_t duty = (uint32_t)((uint64_t)BL_DUTY_MAX * percent / 100u);
    ledc_set_duty(BL_LEDC_MODE, BL_LEDC_CHANNEL, duty);
    ledc_update_duty(BL_LEDC_MODE, BL_LEDC_CHANNEL);
}

uint8_t display_get_backlight(void) {
    return g_backlight;
}

bool display_init(void) {
    const pulse_config_t *c = config_get();

    backlight_init();

    /* SPI 总线。max_transfer_sz 限制单次 DMA 传输大小：
     * 按整屏一行的字节数 x 若干行给足即可，过大会白占内部 DMA 描述符。 */
    spi_bus_config_t bus = {};
    bus.sclk_io_num = PIN_LCD_SCLK;
    bus.mosi_io_num = PIN_LCD_MOSI;
    bus.miso_io_num = -1; /* ST7789 只写不读 */
    bus.quadwp_io_num = -1;
    bus.quadhd_io_num = -1;
    bus.max_transfer_sz = LCD_H_RES * 80 * sizeof(uint16_t);

    esp_err_t err = spi_bus_initialize(LCD_SPI_HOST, &bus, SPI_DMA_CH_AUTO);
    if (err != ESP_OK) {
        log_e("%s: spi_bus_initialize 失败: %s", TAG, esp_err_to_name(err));
        return false;
    }

    /* 送显完成信号量，供 display_blit() 的阻塞语义使用。 */
    if (!g_trans_done) {
        g_trans_done = xSemaphoreCreateBinary();
        if (!g_trans_done) {
            log_e("%s: 送显信号量创建失败", TAG);
            spi_bus_free(LCD_SPI_HOST);
            return false;
        }
    }

    esp_lcd_panel_io_spi_config_t io_cfg = {};
    io_cfg.cs_gpio_num = PIN_LCD_CS;
    io_cfg.dc_gpio_num = PIN_LCD_DC;
    io_cfg.spi_mode = 0;
    io_cfg.pclk_hz = LCD_PIXEL_CLOCK_HZ;
    io_cfg.trans_queue_depth = 10;
    io_cfg.on_color_trans_done = on_color_trans_done;
    io_cfg.user_ctx = nullptr;
    io_cfg.lcd_cmd_bits = 8;
    io_cfg.lcd_param_bits = 8;

    err = esp_lcd_new_panel_io_spi((esp_lcd_spi_bus_handle_t)LCD_SPI_HOST, &io_cfg, &g_io);
    if (err != ESP_OK) {
        log_e("%s: esp_lcd_new_panel_io_spi 失败: %s", TAG, esp_err_to_name(err));
        spi_bus_free(LCD_SPI_HOST);
        return false;
    }

    /* reset_gpio_num = -1：复位由 display_shared_reset() 统一处理，
     * 避免 esp_lcd 私自复位 GPIO13 把触摸芯片一起带走。 */
    esp_lcd_panel_dev_config_t panel_cfg = {};
    panel_cfg.reset_gpio_num = -1;
    /* ST7789 面板通常为 BGR 排列；若 M2 测试看到红蓝互换，改成 RGB。 */
    panel_cfg.color_space = ESP_LCD_COLOR_SPACE_BGR;
    panel_cfg.bits_per_pixel = 16;

    err = esp_lcd_new_panel_st7789(g_io, &panel_cfg, &g_panel);
    if (err != ESP_OK) {
        log_e("%s: esp_lcd_new_panel_st7789 失败: %s", TAG, esp_err_to_name(err));
        esp_lcd_panel_io_del(g_io);
        g_io = nullptr;
        spi_bus_free(LCD_SPI_HOST);
        return false;
    }

    /* 这里不调用 esp_lcd_panel_reset()：reset_gpio_num 为 -1 时它只发软复位命令，
     * 硬复位已由 display_shared_reset() 完成。 */
    esp_lcd_panel_reset(g_panel);
    esp_lcd_panel_init(g_panel);

    /* ST7789 面板普遍需要反色，否则黑白颠倒。 */
    esp_lcd_panel_invert_color(g_panel, true);

    /* rotation: 0/2 为纵向，1/3 为横向（交换 xy）。 */
    switch (c->ui.rotation) {
        case 1:
            esp_lcd_panel_swap_xy(g_panel, true);
            esp_lcd_panel_mirror(g_panel, true, false);
            g_swap_xy = true;
            break;
        case 2:
            esp_lcd_panel_swap_xy(g_panel, false);
            esp_lcd_panel_mirror(g_panel, true, true);
            g_swap_xy = false;
            break;
        case 3:
            esp_lcd_panel_swap_xy(g_panel, true);
            esp_lcd_panel_mirror(g_panel, false, true);
            g_swap_xy = true;
            break;
        default:
            esp_lcd_panel_swap_xy(g_panel, false);
            esp_lcd_panel_mirror(g_panel, false, false);
            g_swap_xy = false;
            break;
    }

    /* GRAM 偏移：240x320 的 GRAM 里对齐 240x280 可视区。
     * swap_xy 后 x/y 的偏移含义也要跟着交换。 */
    if (g_swap_xy) {
        esp_lcd_panel_set_gap(g_panel, LCD_Y_OFFSET, LCD_X_OFFSET);
    } else {
        esp_lcd_panel_set_gap(g_panel, LCD_X_OFFSET, LCD_Y_OFFSET);
    }

    esp_lcd_panel_disp_on_off(g_panel, true);

    g_ready = true;
    log_i("%s: ST7789 就绪 %dx%d rotation=%u offset=(%d,%d) %uMHz", TAG, display_width(),
          display_height(), c->ui.rotation, LCD_X_OFFSET, LCD_Y_OFFSET,
          LCD_PIXEL_CLOCK_HZ / 1000000);
    return true;
}

bool display_is_ready(void) {
    return g_ready;
}

int display_width(void) {
    return g_swap_xy ? LCD_V_RES : LCD_H_RES;
}

int display_height(void) {
    return g_swap_xy ? LCD_H_RES : LCD_V_RES;
}

bool display_blit(int x1, int y1, int x2, int y2, const uint16_t *pixels) {
    if (!g_ready || !pixels) return false;

    /* esp_lcd 的 draw_bitmap 使用半开区间，故右下角 +1。 */
    g_blocking_blit = true;
    esp_err_t err = esp_lcd_panel_draw_bitmap(g_panel, x1, y1, x2 + 1, y2 + 1, pixels);
    if (err != ESP_OK) {
        g_blocking_blit = false;
        return false;
    }
    /* 等 DMA 真正把这一块发完再返回，调用方即可安全复用缓冲。
     * 超时（200ms）说明 SPI 传输可能卡住，返回 false 让调用方感知异常。 */
    if (g_trans_done) {
        if (xSemaphoreTake(g_trans_done, pdMS_TO_TICKS(200)) != pdTRUE) {
            g_blocking_blit = false;
            return false;
        }
    }
    g_blocking_blit = false;
    return true;
}

bool display_blit_async(int x1, int y1, int x2, int y2, const uint16_t *pixels) {
    if (!g_ready || !pixels) return false;
    return esp_lcd_panel_draw_bitmap(g_panel, x1, y1, x2 + 1, y2 + 1, pixels) == ESP_OK;
}

bool display_fill(uint16_t color) {
    if (!g_ready) return false;

    /* 逐行填充：一行 240 像素只需 480 字节，避免为整屏分配 134KB。
     * DMA 要求缓冲区在内部 RAM，不能用 PSRAM，因此用栈上数组。 */
    const int w = display_width();
    const int h = display_height();
    static uint16_t line[LCD_V_RES]; /* 取较大边，容纳任意 rotation 下的一行 */
    for (int i = 0; i < w && i < LCD_V_RES; i++) {
        line[i] = color;
    }
    for (int y = 0; y < h; y++) {
        if (esp_lcd_panel_draw_bitmap(g_panel, 0, y, w, y + 1, line) != ESP_OK) {
            return false;
        }
    }
    return true;
}

void display_on_off(bool on) {
    if (!g_ready) return;
    esp_lcd_panel_disp_on_off(g_panel, on);
}
