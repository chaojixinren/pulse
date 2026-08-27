/**
 * @file pulse_pmu.h
 * @brief AXP2101 电源管理 HAL
 *
 * 硬件电源架构（已确认）
 * ----------------------
 * AXP2101 的电源轨实际使用情况：
 *   VRTC  → BM8563 RTC 专用供电（always-on，深睡期间保持）
 *   DCDC1 → 3.3V 系统轨（ESP32-S3、LCD、触摸、TF 卡、I2C 总线共用）
 *   其余轨（DCDC3/5、ALDO1~4、BLDO1/2）全部悬空，不接任何负载
 *
 * 因此 pmu_init() 不需要配置任何电源轨——VRTC 默认 always-on，
 * DCDC1 默认使能，均符合系统需求。
 *
 * 安全约定
 * --------
 * 本模块默认**只读**：不修改任何电源轨电压、不改变轨使能状态。
 * 虽然轨映射已确认，但 pmu_init() 仍保持只读——当前硬件默认配置
 * 已满足需求，无需额外配置。
 *
 * I2C: GPIO4(SCL) / GPIO5(SDA)，从机地址 0x34，与 BM8563/CST816 共用总线。
 * IRQ: GPIO6，低电平有效，下降沿触发。
 */
#ifndef PULSE_PMU_H
#define PULSE_PMU_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* 充电状态。不直接暴露 XPowersLib 的枚举，避免上层模块被库类型污染。 */
typedef enum {
    PMU_CHG_UNKNOWN = 0,
    PMU_CHG_TRICKLE,  /* 涓流 */
    PMU_CHG_PRE,      /* 预充 */
    PMU_CHG_CC,       /* 恒流 */
    PMU_CHG_CV,       /* 恒压 */
    PMU_CHG_DONE,     /* 充满 */
    PMU_CHG_NOT_CHARGING,
} pmu_chg_state_t;

/* 电源事件。由 GPIO6 中断触发后在任务上下文中读取 I2C 得到。 */
typedef enum {
    PMU_EVT_PWRKEY_SHORT = 0, /* PWR 短按：唤醒 / 切换屏幕 */
    PMU_EVT_PWRKEY_LONG,      /* PWR 长按：关机 */
    PMU_EVT_VBUS_INSERT,
    PMU_EVT_VBUS_REMOVE,
    PMU_EVT_BAT_INSERT,
    PMU_EVT_BAT_REMOVE,
    PMU_EVT_CHG_START,
    PMU_EVT_CHG_DONE,
    PMU_EVT_LOW_WARN, /* SOC 掉到告警阈值 1 */
    PMU_EVT_LOW_CRIT, /* SOC 掉到告警阈值 2 */
    PMU_EVT_COUNT,
} pmu_event_t;

typedef struct {
    bool battery_present;
    int percent;      /* 0~100；无电池或读取失败为 -1 */
    uint16_t batt_mv; /* 电池电压 mV；无电池为 0 */
    bool vbus_present;
    uint16_t vbus_mv;
    bool charging;
    pmu_chg_state_t chg_state;
    float die_temp_c; /* PMIC 芯片温度 */
} pmu_status_t;

/**
 * 初始化 PMU。只读探测 + ADC + 中断使能，不修改电源轨。
 * @return false 表示 I2C 无应答或 CHIP_ID 不匹配
 */
bool pmu_init(void);

/** PMU 是否已成功初始化 */
bool pmu_is_ready(void);

/** 读取当前电池/充电状态。失败返回 false，out 内容不可信。 */
bool pmu_read(pmu_status_t *out);

/**
 * 取出自上次调用以来累积的事件。
 * 必须在任务上下文调用（内部走 I2C，不能在 ISR 里用）。
 * @param buf 输出缓冲
 * @param max buf 容量
 * @return 实际写入的事件数；无事件返回 0
 */
int pmu_poll_events(pmu_event_t *buf, int max);

/** 事件名，用于日志 */
const char *pmu_event_str(pmu_event_t e);

/** 充电状态名，用于日志与 UI */
const char *pmu_chg_state_str(pmu_chg_state_t s);

/**
 * 打印所有电源轨的当前电压与使能状态（只读诊断）。
 * 用于确认原理图上哪一路给哪个外设供电。
 */
void pmu_dump_rails(void);

/** 软关机（切断 BATFET）。深睡唤醒无效，需按 PWR 键重新上电。 */
void pmu_shutdown(void);

/**
 * I2C 总线是否已被 pmu_init() 初始化。
 * 供其他 I2C 从机驱动（RTC/触摸）判断是否需要自行调用 Wire.begin()，
 * 避免重复初始化导致正在进行的 I2C 事务被中断。
 */
bool pmu_i2c_is_initialized(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_PMU_H */
