/**
 * @file pulse_app.h
 * @brief 应用编排层：把全部 HAL 模块串成完整固件
 *
 * 职责
 * ----
 * 这是 M11，也是最终交付的主固件。它只做**编排**，不含具体驱动逻辑：
 *   - 按依赖顺序初始化所有模块（PMU → 存储 → RTC → 显示 → 触摸 → LVGL
 *     → 音频 → WiFi → 上传 → 电源）
 *   - 创建 FreeRTOS 任务，划分实时域（音频）与网络域（WiFi/上传）
 *   - 实现 power hooks（灭屏/亮屏/收尾/低电告警），把电源状态机接到真外设
 *   - 把各模块状态汇总成 pulse_ui_state_t，周期推给界面层
 *
 * 任务划分（FreeRTOS）
 * --------------------
 * 双核 S3，音频与网络天然分开：
 *
 *   Core 0（protocol core，Arduino 默认 loop 所在核）
 *     - UI 任务：LVGL 渲染 + 触摸（对延迟敏感，优先）
 *     - 网络任务：WiFi tick + 上传 tick（阻塞型 HTTP，独立任务不卡 UI）
 *
 *   Core 1（app core）
 *     - 音频任务：I2S 读取 + VAD + 段写入（实时性最高，独占）
 *
 * 电源 tick 挂在 UI 任务里低频跑，PMU 事件在音频任务外独立处理。
 *
 * 关键安全点
 * ----------
 * - LVGL 非线程安全：所有 lv_* 调用都走 lv_port_lock() 保护。
 * - 跨任务通信用 Queue + EventGroup，不共享裸指针。
 * - 音频环形缓冲与 pre-roll 在 PSRAM，DMA 缓冲在内部 RAM。
 */
#ifndef PULSE_APP_H
#define PULSE_APP_H

#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

/** 启动应用（创建任务，之后由 FreeRTOS 调度，本函数返回）*/
void pulse_app_start(void);

/** 供 Arduino setup() 调用的同步初始化。失败返回 false。 */
bool pulse_app_init(void);

/**
 * 一站式启动：初始化 + 诊断打印 + 启动任务。
 * 供 main.cpp 的 setup() 调用。内部处理初始化失败的错误循环。
 */
void pulse_app_bootstrap(void);

/** 供 main.cpp 的 loop() 调用：时间/电量低频刷新。 */
void pulse_app_loop(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_APP_H */
