/**
 * @file lv_port.cpp
 * @brief LVGL 8.4 移植层实现
 */
#include "lv_port.h"

#include <Arduino.h>
#include <esp_heap_caps.h>
#include <esp_timer.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <lvgl.h>

#include "board_pins.h"
#include "config/pulse_config.h"
#include "hal/display/pulse_display.h"
#include "hal/touch/pulse_touch.h"

static const char *TAG = "lvgl";

/* 绘制缓冲行数。40 行 x 240 像素 x 2 字节 = 19.2KB/缓冲，双缓冲共 38.4KB。
 * 取 40 行是权衡：太小则 flush 次数多、SPI 事务开销占比高；
 * 太大则占用内部 RAM（DMA 要求）过多。 */
#define LV_BUF_LINES 40

static bool g_ready = false;
static lv_disp_draw_buf_t g_draw_buf;
static lv_color_t *g_buf1 = nullptr;
static lv_color_t *g_buf2 = nullptr;
static lv_disp_drv_t g_disp_drv;
static lv_indev_drv_t g_indev_drv;
static lv_disp_t *g_disp = nullptr;
static esp_timer_handle_t g_tick_timer = nullptr;
static SemaphoreHandle_t g_mutex = nullptr;
static size_t g_buf_bytes = 0;
static bool g_buf_in_psram = false;

/* ─────────────── 送显 ─────────────── */

/* DMA 完成回调（ISR 上下文）：通知 LVGL 这一块已发完。 */
static bool flush_done_cb(void *user_ctx) {
    (void)user_ctx;
    lv_disp_flush_ready(&g_disp_drv);
    /* 不需要唤醒任务，lv_timer_handler 下次轮询即可。 */
    return false;
}

static void disp_flush_cb(lv_disp_drv_t *drv, const lv_area_t *area, lv_color_t *color_p) {
    /* 用异步提交：DMA 期间 LVGL 可以在另一个缓冲上继续绘制，
     * 完成后由 flush_done_cb 调 lv_disp_flush_ready。
     * 这是双缓冲能提速的关键，不能用阻塞版 display_blit。 */
    if (!display_blit_async(area->x1, area->y1, area->x2, area->y2, (const uint16_t *)color_p)) {
        /* 提交失败也必须告知 LVGL，否则渲染流水线会永久卡住。 */
        lv_disp_flush_ready(drv);
    }
}

/* ─────────────── 触摸输入 ─────────────── */

static void indev_read_cb(lv_indev_drv_t *drv, lv_indev_data_t *data) {
    (void)drv;
    touch_data_t t;
    if (!touch_read(&t) || !t.pressed) {
        data->state = LV_INDEV_STATE_RELEASED;
        return;
    }
    data->state = LV_INDEV_STATE_PRESSED;
    data->point.x = t.x;
    data->point.y = t.y;
}

/* ─────────────── tick ─────────────── */

static void tick_cb(void *arg) {
    (void)arg;
    /* esp_timer 回调运行在专用任务里，只做加计数，不碰 LVGL 对象，
     * 因此无需加锁。 */
    lv_tick_inc(1);
}

/* ─────────────── 缓冲分配 ─────────────── */

/* LVGL 的绘制缓冲会被直接交给 SPI DMA，必须在内部 RAM（DMA 可达）。
 * PSRAM 虽然大，但 ESP32-S3 的 SPI DMA 不能直接读 PSRAM，
 * 强行使用会导致花屏或 DMA 错误。因此这里显式要求 MALLOC_CAP_DMA。 */
static bool alloc_buffers(void) {
    const size_t px = LCD_H_RES * LV_BUF_LINES;
    g_buf_bytes = px * sizeof(lv_color_t);

    g_buf1 = (lv_color_t *)heap_caps_malloc(g_buf_bytes, MALLOC_CAP_DMA | MALLOC_CAP_INTERNAL);
    g_buf2 = (lv_color_t *)heap_caps_malloc(g_buf_bytes, MALLOC_CAP_DMA | MALLOC_CAP_INTERNAL);

    if (!g_buf1) {
        log_e("%s: 绘制缓冲分配失败（需 %u 字节 DMA 内部 RAM）", TAG, (unsigned)g_buf_bytes);
        return false;
    }
    if (!g_buf2) {
        /* 单缓冲仍可工作，只是刷新时 LVGL 需等 DMA 完成，帧率略降。 */
        log_w("%s: 第二绘制缓冲分配失败，降级为单缓冲", TAG);
    }
    g_buf_in_psram = false;
    return true;
}

/* ─────────────── 对外接口 ─────────────── */

bool lv_port_init(void) {
    const pulse_config_t *c = config_get();

    if (!display_is_ready()) {
        log_e("%s: display 未就绪，请先调用 display_init()", TAG);
        return false;
    }

    g_mutex = xSemaphoreCreateRecursiveMutex();
    if (!g_mutex) {
        log_e("%s: 互斥锁创建失败", TAG);
        return false;
    }

    lv_init();

    if (!alloc_buffers()) {
        return false;
    }
    lv_disp_draw_buf_init(&g_draw_buf, g_buf1, g_buf2, LCD_H_RES * LV_BUF_LINES);

    lv_disp_drv_init(&g_disp_drv);
    g_disp_drv.hor_res = (lv_coord_t)display_width();
    g_disp_drv.ver_res = (lv_coord_t)display_height();
    g_disp_drv.flush_cb = disp_flush_cb;
    g_disp_drv.draw_buf = &g_draw_buf;
    /* 屏幕旋转已在 ST7789 硬件层通过 swap_xy/mirror 完成，
     * LVGL 侧不再重复旋转，否则触摸坐标与显示会错位。 */
    g_disp_drv.rotated = LV_DISP_ROT_NONE;
    g_disp_drv.sw_rotate = 0;
    g_disp = lv_disp_drv_register(&g_disp_drv);
    if (!g_disp) {
        log_e("%s: lv_disp_drv_register 失败", TAG);
        return false;
    }

    /* 注册 DMA 完成回调，把异步送显接回 LVGL 的流水线。 */
    display_set_flush_done_cb(flush_done_cb, nullptr);

    lv_indev_drv_init(&g_indev_drv);
    g_indev_drv.type = LV_INDEV_TYPE_POINTER;
    g_indev_drv.read_cb = indev_read_cb;
    g_indev_drv.disp = g_disp;
    if (!lv_indev_drv_register(&g_indev_drv)) {
        log_w("%s: 触摸输入设备注册失败，界面将不可交互", TAG);
    }

    /* 1ms 周期的 tick。LVGL 所有动画与超时都依赖它，精度不足会导致动画卡顿。 */
    esp_timer_create_args_t targs = {};
    targs.callback = tick_cb;
    targs.name = "lv_tick";
    if (esp_timer_create(&targs, &g_tick_timer) != ESP_OK ||
        esp_timer_start_periodic(g_tick_timer, 1000) != ESP_OK) {
        log_e("%s: lv_tick 定时器启动失败", TAG);
        return false;
    }

    g_ready = true;
    log_i("%s: LVGL %d.%d.%d 就绪 %dx%d 缓冲 %u 字节 x%d", TAG, LVGL_VERSION_MAJOR,
          LVGL_VERSION_MINOR, LVGL_VERSION_PATCH, display_width(), display_height(),
          (unsigned)g_buf_bytes, g_buf2 ? 2 : 1);
    (void)c;
    return true;
}

bool lv_port_is_ready(void) {
    return g_ready;
}

bool lv_port_lock(uint32_t timeout_ms) {
    if (!g_mutex) return false;
    TickType_t ticks = (timeout_ms == 0) ? portMAX_DELAY : pdMS_TO_TICKS(timeout_ms);
    return xSemaphoreTakeRecursive(g_mutex, ticks) == pdTRUE;
}

void lv_port_unlock(void) {
    if (g_mutex) xSemaphoreGiveRecursive(g_mutex);
}

uint32_t lv_port_task_handler(void) {
    if (!g_ready) return 100;
    uint32_t next = 100;
    if (lv_port_lock(50)) {
        next = lv_timer_handler();
        lv_port_unlock();
    }
    /* lv_timer_handler 可能返回 LV_NO_TIMER_READY(UINT32_MAX)，钳位避免长睡。 */
    if (next > 100) next = 100;
    if (next < 1) next = 1;
    return next;
}

void lv_port_dump_buffers(void) {
    Serial.println();
    Serial.println("============== LVGL 缓冲分配 ==============");
    Serial.printf("  单缓冲   %u 字节 (%d 行 x %d 像素 x %u B)\n", (unsigned)g_buf_bytes,
                  LV_BUF_LINES, LCD_H_RES, (unsigned)sizeof(lv_color_t));
    Serial.printf("  缓冲数   %d %s\n", g_buf2 ? 2 : 1, g_buf2 ? "(双缓冲)" : "(单缓冲，降级)");
    Serial.printf("  位置     %s\n", g_buf_in_psram ? "PSRAM" : "内部 RAM (DMA 要求)");
    Serial.printf("  合计     %u 字节\n", (unsigned)(g_buf_bytes * (g_buf2 ? 2 : 1)));
    Serial.printf("  LV_COLOR_16_SWAP = %d\n", LV_COLOR_16_SWAP);
    Serial.printf("  剩余内部堆 %u 字节, PSRAM %u 字节\n",
                  (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
                  (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));
    Serial.println("  说明：ESP32-S3 的 SPI DMA 无法直接读 PSRAM，");
    Serial.println("        故绘制缓冲必须留在内部 RAM。");
    Serial.println("=========================================");
}
