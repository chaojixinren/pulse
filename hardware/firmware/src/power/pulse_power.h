/**
 * @file pulse_power.h
 * @brief 三级功耗状态机 + 深睡唤醒
 *
 * 为什么是三级而不是「5 分钟静音就深睡」
 * --------------------------------------
 * 深睡会关掉 I2S 与 CPU，麦克风完全停摆，而板上没有「听到人声自动唤醒」
 * 的硬件通路（那需要模拟 VAD 唤醒电路）。若按静音 5 分钟就深睡，
 * 设备会在没人按键的整个白天里聋掉，与「全天候记录重要对话」直接冲突。
 *
 * 因此分三级：
 *   Active      屏亮 + 录音        30s 无触摸 → Idle
 *   Idle        屏灭 + **继续录音**  省掉功耗大头（背光），但不漏对话
 *   DeepSleep   全关               仅手动收工 / 电量告急 / 显式配置时进入
 *
 * config.json 的 power.deep_sleep_ms 默认为 0 = 永不自动深睡。
 * 若你确实要原本的「静音即深睡」语义，把它设为 300000 即可，
 * 代价是设备在静音期听不到声音。
 *
 * 唤醒源
 * ------
 * GPIO7  PWR 键（按下拉低，同时是 AXP2101_PWRON）
 * GPIO6  AXP2101_IRQ（低电平有效，充电器插入/电量事件）
 * 两者都是低电平有效，故用 ext1 + ESP_EXT1_WAKEUP_ANY_LOW 一并覆盖。
 * 二者均落在 GPIO0~21（RTCIO 域），深睡期间仍可检测。
 *
 * 分层约定
 * --------
 * 本模块只管状态机与睡眠，不直接依赖 display/audio/storage ——
 * 那些动作通过 hooks 由 app 层注册。这样 M9 能独立编译测试，
 * 也避免了 HAL 之间的横向耦合。
 */
#ifndef PULSE_POWER_H
#define PULSE_POWER_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    PULSE_PWR_ACTIVE = 0, /* 屏亮，录音 */
    PULSE_PWR_IDLE,       /* 屏灭，仍录音 */
    PULSE_PWR_SLEEPING,   /* 即将进入深睡（过渡态）*/
} pulse_power_state_t;

/** 深睡触发原因，用于日志与唤醒后的行为决策 */
typedef enum {
    PULSE_SLEEP_NONE = 0,
    PULSE_SLEEP_MANUAL,      /* 手动触发（当前无按键绑定，保留给测试与未来扩展） */
    PULSE_SLEEP_IDLE_TIMEOUT,/* 达到 deep_sleep_ms（默认关闭）*/
    PULSE_SLEEP_BATTERY_CRIT,/* 电量低于 crit 阈值，保护性深睡 */
} pulse_sleep_reason_t;

/** 唤醒原因 */
typedef enum {
    PULSE_WAKE_UNKNOWN = 0,
    PULSE_WAKE_POWER_ON,   /* 冷启动，非深睡唤醒 */
    PULSE_WAKE_PWR_KEY,
    PULSE_WAKE_PMU_IRQ,
    PULSE_WAKE_TIMER,
} pulse_wake_cause_t;

/**
 * 外设动作钩子。由 app 层注册，power 模块在状态切换时调用。
 * 任何一个为 NULL 都会被跳过。
 */
typedef struct {
    void (*on_enter_idle)(void);  /* 灭屏（背光 PWM=0，可选 display_on_off）*/
    void (*on_exit_idle)(void);   /* 恢复背光与显示 */
    /* 返回 false 可否决深睡。用于「spool 还没传完就不睡」。 */
    bool (*can_sleep)(void);
    /* 进深睡前的收尾：关段、停 I2S、触摸进低功耗、断网。 */
    void (*before_sleep)(void);
    /* 低电与危急告警，用于在屏幕上弹提示 */
    void (*on_battery_warn)(int percent);
    void (*on_battery_crit)(int percent);
} pulse_power_hooks_t;

typedef struct {
    pulse_power_state_t state;
    pulse_wake_cause_t wake_cause;   /* 本次开机的唤醒原因 */
    uint32_t idle_ms;                /* 当前无活动时长 */
    uint32_t uptime_s;
    int battery_percent;             /* -1 表示未知 */
    bool charging;
    bool battery_warned;             /* 已发出低电告警 */
    uint32_t sleep_blocked_count;    /* 因 can_sleep 否决而未睡的次数 */
} pulse_power_status_t;

/**
 * 初始化电源管理。会读取并清除深睡唤醒原因。
 * @param hooks 可为 NULL（此时仅维护状态机，不操作外设）
 */
bool pulse_power_init(const pulse_power_hooks_t *hooks);

/**
 * 驱动状态机。需周期调用（建议 100~200ms）。
 * 负责：无活动计时、PMU 轮询、低电告警、深睡判定。
 */
void pulse_power_tick(void);

/** 报告用户活动（触摸、按键）。会把状态拉回 Active 并重置计时。 */
void pulse_power_notify_activity(void);

/** 报告有语音活动。用于「静音才算无活动」的判定。 */
void pulse_power_notify_speech(void);

pulse_power_state_t pulse_power_state(void);
const char *pulse_power_state_str(pulse_power_state_t s);
const char *pulse_wake_cause_str(pulse_wake_cause_t c);
const char *pulse_sleep_reason_str(pulse_sleep_reason_t r);

/** 本次开机的唤醒原因 */
pulse_wake_cause_t pulse_power_wake_cause(void);

bool pulse_power_get_status(pulse_power_status_t *out);

/**
 * 立刻请求进入深睡。会先询问 can_sleep，再调 before_sleep。
 * @param reason 触发原因，写入日志
 * @param force  true 则忽略 can_sleep 的否决（电量危急时用）
 * @return 仅在被否决时返回 false；成功则不返回（芯片进入深睡）
 */
bool pulse_power_deep_sleep(pulse_sleep_reason_t reason, bool force);

/** 配置深睡唤醒源但不睡（供测试单独验证配置是否成功）*/
bool pulse_power_arm_wakeup(void);

void pulse_power_dump(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_POWER_H */
