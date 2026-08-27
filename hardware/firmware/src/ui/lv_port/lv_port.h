/**
 * @file lv_port.h
 * @brief LVGL 8.4 移植层：绑定 ST7789 显示与 CST816 触摸
 *
 * 职责边界
 * --------
 * 本层只做三件事：
 *   1. 分配绘制缓冲（优先 PSRAM，DMA 中转缓冲用内部 RAM）
 *   2. 注册 flush_cb（送显）与 read_cb（触摸输入）
 *   3. 用 esp_timer 周期调用 lv_tick_inc()
 *
 * 不在此层创建任何界面元素 —— 具体屏幕由 M10 的 src/ui/screens/ 负责。
 *
 * 线程约定（重要）
 * ----------------
 * LVGL 8.x 不是线程安全的。lv_timer_handler() 与所有 lv_xxx() 调用
 * 必须在同一个任务里，或者由调用方自行加锁。本层提供 lv_port_lock()/
 * lv_port_unlock() 供其他任务（如 power_task 改亮度、net_task 更新状态）
 * 安全地操作 LVGL 对象。
 */
#ifndef PULSE_LV_PORT_H
#define PULSE_LV_PORT_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * 初始化 LVGL 与显示/输入驱动。
 * 前置条件：display_shared_reset() -> display_init() -> touch_init() 均已完成。
 * @return false 表示绘制缓冲分配失败
 */
bool lv_port_init(void);

bool lv_port_is_ready(void);

/**
 * 驱动 LVGL 运行。需周期调用（建议 5ms），内部已含互斥锁。
 * @return 距下次应调用的建议毫秒数
 */
uint32_t lv_port_task_handler(void);

/** 获取 LVGL 互斥锁。跨任务操作 lv_xxx() 前必须调用。 */
bool lv_port_lock(uint32_t timeout_ms);

/** 释放 LVGL 互斥锁 */
void lv_port_unlock(void);

/** 上报绘制缓冲的实际分配情况，用于确认是否落在 PSRAM */
void lv_port_dump_buffers(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_LV_PORT_H */
