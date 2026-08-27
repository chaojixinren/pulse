/**
 * @file pulse_power.cpp
 * @brief 三级功耗状态机实现
 */
#include "pulse_power.h"

#include <Arduino.h>
#include <driver/rtc_io.h>
#include <esp_sleep.h>

#include <string.h>

#include "board_pins.h"
#include "config/pulse_config.h"
#include "hal/pmu/pulse_pmu.h"

static const char *TAG = "power";

static pulse_power_hooks_t g_hooks;
static pulse_power_state_t g_state = PULSE_PWR_ACTIVE;
static pulse_wake_cause_t g_wake_cause = PULSE_WAKE_UNKNOWN;
static uint32_t g_last_activity_ms = 0;
static uint32_t g_last_speech_ms = 0;
static uint32_t g_uptime_s = 0;
static uint32_t g_uptime_prev_ms = 0;
static bool g_battery_warned = false;
static bool g_crit_handled = false;
static uint32_t g_sleep_blocked = 0;

/* ─────────── 初始化 ─────────── */

bool pulse_power_init(const pulse_power_hooks_t *hooks) {
    if (hooks) {
        memcpy(&g_hooks, hooks, sizeof(g_hooks));
    } else {
        memset(&g_hooks, 0, sizeof(g_hooks));
    }

    g_last_activity_ms = millis();
    g_last_speech_ms = millis();
    g_battery_warned = false;
    g_crit_handled = false;

    /* 解析唤醒原因。esp_sleep 的 ext1 唤醒状态在深睡唤醒后仍保留在 RTC 域，
     * 无需额外跨复位存储：先判断唤醒类型，再读具体引脚。 */
    esp_sleep_wakeup_cause_t wc = esp_sleep_get_wakeup_cause();
    if (wc == ESP_SLEEP_WAKEUP_EXT1) {
        uint64_t status = esp_sleep_get_ext1_wakeup_status();
        if (status & (1ULL << PIN_PMU_PWRON)) {
            g_wake_cause = PULSE_WAKE_PWR_KEY;
        } else if (status & (1ULL << PIN_PMU_IRQ)) {
            g_wake_cause = PULSE_WAKE_PMU_IRQ;
        } else {
            g_wake_cause = PULSE_WAKE_UNKNOWN;
        }
        log_i("%s: 深睡唤醒，原因 %s", TAG, pulse_wake_cause_str(g_wake_cause));
    } else if (wc == ESP_SLEEP_WAKEUP_TIMER) {
        g_wake_cause = PULSE_WAKE_TIMER;
        log_i("%s: 定时器唤醒", TAG);
    } else {
        g_wake_cause = PULSE_WAKE_POWER_ON;
        log_i("%s: 冷启动（非深睡唤醒）", TAG);
    }

    /* 清除唤醒状态位，避免影响下次深睡判定。 */
    esp_sleep_disable_wakeup_source(ESP_SLEEP_WAKEUP_ALL);

    log_i("%s: 电源管理就绪，当前状态 %s", TAG, pulse_power_state_str(g_state));
    return true;
}

/* ─────────── 活动通知 ─────────── */

void pulse_power_notify_activity(void) {
    g_last_activity_ms = millis();
    if (g_state == PULSE_PWR_IDLE) {
        g_state = PULSE_PWR_ACTIVE;
        log_i("%s: 检测到用户活动，回到 Active", TAG);
        if (g_hooks.on_exit_idle) g_hooks.on_exit_idle();
    }
}

/* Active → Idle：灭屏但录音/传输继续。自动超时与 PWR 短按共用。 */
static void enter_idle(const char *reason) {
    if (g_state != PULSE_PWR_ACTIVE) return;
    g_state = PULSE_PWR_IDLE;
    log_i("%s: %s，进入 Idle（灭屏，录音/传输继续）", TAG, reason);
    if (g_hooks.on_enter_idle) g_hooks.on_enter_idle();
}

void pulse_power_notify_speech(void) {
    g_last_speech_ms = millis();
}

/* ─────────── 状态机 ─────────── */

void pulse_power_tick(void) {
    const pulse_config_t *c = config_get();
    uint32_t now = millis();

    /* uptime 秒计数 */
    if (now - g_uptime_prev_ms >= 1000) {
        g_uptime_s += (now - g_uptime_prev_ms) / 1000;
        g_uptime_prev_ms = now;
    }

    if (g_state == PULSE_PWR_SLEEPING) return; /* 即将深睡，不再处理 */

    /* ── Active → Idle：无触摸达到 screen_dim_ms ── */
    uint32_t idle_ms = now - g_last_activity_ms;
    if (g_state == PULSE_PWR_ACTIVE && idle_ms >= c->power.screen_dim_ms) {
        enter_idle("无触摸超时");
    }

    /* ── PMU 轮询 ── */
    static uint32_t last_pmu = 0;
    if (now - last_pmu >= c->power.pmu_poll_ms) {
        last_pmu = now;

        /* 处理 PMU 事件：按键唤醒/深睡 */
        pmu_event_t evts[PMU_EVT_COUNT];
        int n = pmu_poll_events(evts, PMU_EVT_COUNT);
        for (int i = 0; i < n; i++) {
            switch (evts[i]) {
                case PMU_EVT_PWRKEY_SHORT:
                    /* 短按语义按当前状态区分：
                     *   Active(亮屏) -> 灭屏进 Idle（录音/传输继续）
                     *   Idle(灭屏)   -> 唤醒回 Active
                     * 深睡不绑按键：长按是 AXP2101 硬件断电，软件
                     * deep_sleep 来不及执行。深睡仅由电量危急与
                     * deep_sleep_ms 静置超时触发。 */
                    if (g_state == PULSE_PWR_ACTIVE) {
                        enter_idle("亮屏短按");
                    } else {
                        pulse_power_notify_activity();
                    }
                    break;
                case PMU_EVT_PWRKEY_LONG:
                    /* 长按由 AXP2101 硬件直接断电（PWRON 引脚行为），
                     * 是固件死机时的强制复位兜底，软件不介入。 */
                    log_w("%s: PWR 长按（AXP2101 硬件断电），软件不介入", TAG);
                    break;
                case PMU_EVT_LOW_WARN:
                case PMU_EVT_LOW_CRIT:
                    break; /* 电量阈值走下方轮询统一处理 */
                default:
                    break;
            }
        }

        /* 电量告警（轮询为主，IRQ 事件为辅 ——
         * AXP2101 的 SOC 阈值中断在部分固件版本不可靠） */
        pmu_status_t ps;
        if (pmu_read(&ps) && ps.battery_present && ps.percent >= 0) {
            if (ps.percent <= (int)c->power.battery_crit_pct) {
                if (!g_crit_handled) {
                    g_crit_handled = true;
                    log_w("%s: 电量 %d%% 低于危急阈值 %u%%，强制深睡保护", TAG, ps.percent,
                          c->power.battery_crit_pct);
                    if (g_hooks.on_battery_crit) g_hooks.on_battery_crit(ps.percent);
                    /* 保护性深睡，忽略 can_sleep 否决 */
                    pulse_power_deep_sleep(PULSE_SLEEP_BATTERY_CRIT, true);
                    return;
                }
            } else if (ps.percent <= (int)c->power.battery_warn_pct) {
                if (!g_battery_warned) {
                    g_battery_warned = true;
                    log_w("%s: 低电 %d%%，将在屏幕上提示", TAG, ps.percent);
                    if (g_hooks.on_battery_warn) g_hooks.on_battery_warn(ps.percent);
                }
            } else {
                /* 电量回升，解除告警锁存。
                 * g_battery_warned：只要高于 warn 阈值就解锁。
                 * g_crit_handled：必须高于 crit 阈值才解锁，否则 5%~10% 区间
                 * 内波动时危急告警会永久失效（设备过放风险）。 */
                g_battery_warned = false;
                if (ps.percent > (int)c->power.battery_crit_pct) {
                    g_crit_handled = false;
                }
            }
        }
    }

    /* ── Idle → DeepSleep：需显式配置 deep_sleep_ms，且静音超时 ──
     * 「无活动」= 无触摸 且 无语音，避免正说话时把人当闲置。 */
    if (g_state == PULSE_PWR_IDLE && c->power.deep_sleep_ms != 0) {
        uint32_t no_touch = now - g_last_activity_ms;
        uint32_t no_speech = now - g_last_speech_ms;
        if (no_touch >= c->power.deep_sleep_ms && no_speech >= c->power.deep_sleep_ms) {
            pulse_power_deep_sleep(PULSE_SLEEP_IDLE_TIMEOUT, false);
        }
    }
}

/* ─────────── 深睡 ─────────── */

bool pulse_power_arm_wakeup(void) {
    /* GPIO6 (PMU IRQ) 与 GPIO7 (PWR 键) 都是低电平有效，
     * 深睡期间用 ext1 + ANY_LOW 同时监听。
     * 注意：ext1 只能监听 RTC 域引脚（GPIO0~21），两者均满足。 */
    uint64_t mask = (1ULL << PIN_PMU_IRQ) | (1ULL << PIN_PMU_PWRON);
    esp_err_t err = esp_sleep_enable_ext1_wakeup(mask, ESP_EXT1_WAKEUP_ANY_LOW);
    if (err != ESP_OK) {
        log_e("%s: ext1 唤醒配置失败: %s", TAG, esp_err_to_name(err));
        return false;
    }

    /* 深睡期间保持 RTC GPIO 的方向与内部上拉。
     * ext1 的 ANY_LOW 需要这些引脚在睡眠时仍有正确的上拉配置，
     * 否则浮空引脚可能误触发唤醒。 */
    rtc_gpio_set_direction((gpio_num_t)PIN_PMU_IRQ, RTC_GPIO_MODE_INPUT_ONLY);
    rtc_gpio_pullup_en((gpio_num_t)PIN_PMU_IRQ);
    rtc_gpio_set_direction((gpio_num_t)PIN_PMU_PWRON, RTC_GPIO_MODE_INPUT_ONLY);
    rtc_gpio_pullup_en((gpio_num_t)PIN_PMU_PWRON);

    log_i("%s: 唤醒源已配置（GPIO%d PWR 键 + GPIO%d PMU IRQ，低电平）", TAG, PIN_PMU_PWRON,
          PIN_PMU_IRQ);
    return true;
}

bool pulse_power_deep_sleep(pulse_sleep_reason_t reason, bool force) {
    /* 否决检查：spool 没传完就不睡（除非强制，如电量危急） */
    if (!force && g_hooks.can_sleep && !g_hooks.can_sleep()) {
        g_sleep_blocked++;
        log_w("%s: 深睡被否决（spool 未清空或外设未就绪），已累计 %u 次", TAG, g_sleep_blocked);
        return false;
    }

    g_state = PULSE_PWR_SLEEPING;
    log_i("%s: 进入深睡，原因 %s", TAG, pulse_sleep_reason_str(reason));

    /* 收尾：关段、停 I2S、触摸低功耗、断网 —— 由 app 层注册。 */
    if (g_hooks.before_sleep) g_hooks.before_sleep();

    /* 唤醒原因不在此记录：esp_sleep 的 ext1 唤醒状态在深睡后保留在 RTC 域，
     * init 阶段用 esp_sleep_get_ext1_wakeup_status() 读取具体引脚即可。 */

    /* 让 SPI/显示/外设引脚进入 hold，避免睡眠期间漏电或误动作 */
    rtc_gpio_hold_en((gpio_num_t)PIN_PMU_IRQ);
    rtc_gpio_hold_en((gpio_num_t)PIN_PMU_PWRON);

    if (!pulse_power_arm_wakeup()) {
        /* 配置失败则放弃深睡，回到 Active，避免「睡了再也起不来」。
         * 重置 g_crit_handled：否则下次电量仍 ≤ crit 阈值时危急告警
         * 会被跳过（设备失去低电保护）。 */
        g_crit_handled = false;
        g_state = PULSE_PWR_ACTIVE;
        return false;
    }

    /* 可选：加一个兜底定时唤醒（默认 0 = 不加）。 */
    delay(20); /* 让串口把日志吐完 */

    /* 进入深睡。此函数不返回。 */
    esp_deep_sleep_start();

    /* 理论上走不到这里；若编译器告警，这里返回 false 兜底。 */
    return false;
}

/* ─────────── 查询接口 ─────────── */

pulse_power_state_t pulse_power_state(void) {
    return g_state;
}

pulse_wake_cause_t pulse_power_wake_cause(void) {
    return g_wake_cause;
}

const char *pulse_power_state_str(pulse_power_state_t s) {
    switch (s) {
        case PULSE_PWR_IDLE: return "Idle（灭屏录音）";
        case PULSE_PWR_SLEEPING: return "进入深睡";
        default: return "Active";
    }
}

const char *pulse_wake_cause_str(pulse_wake_cause_t c) {
    switch (c) {
        case PULSE_WAKE_PWR_KEY: return "PWR 键";
        case PULSE_WAKE_PMU_IRQ: return "PMU 事件（IRQ）";
        case PULSE_WAKE_TIMER: return "定时器";
        case PULSE_WAKE_POWER_ON: return "冷启动上电";
        default: return "未知";
    }
}

const char *pulse_sleep_reason_str(pulse_sleep_reason_t r) {
    switch (r) {
        case PULSE_SLEEP_MANUAL: return "手动触发";
        case PULSE_SLEEP_IDLE_TIMEOUT: return "静置超时";
        case PULSE_SLEEP_BATTERY_CRIT: return "电量危急保护";
        default: return "未指定";
    }
}

bool pulse_power_get_status(pulse_power_status_t *out) {
    if (!out) return false;
    memset(out, 0, sizeof(*out));
    out->state = g_state;
    out->wake_cause = g_wake_cause;
    out->idle_ms = millis() - g_last_activity_ms;
    out->uptime_s = g_uptime_s;
    out->sleep_blocked_count = g_sleep_blocked;

    pmu_status_t ps;
    if (pmu_read(&ps)) {
        out->battery_percent = ps.percent;
        out->charging = ps.charging;
    } else {
        out->battery_percent = -1;
    }
    out->battery_warned = g_battery_warned;
    return true;
}

void pulse_power_dump(void) {
    const pulse_config_t *c = config_get();
    pulse_power_status_t st;
    pulse_power_get_status(&st);

    Serial.println();
    Serial.println("================ 电源管理 ================");
    Serial.printf("  状态        %s\n", pulse_power_state_str(st.state));
    Serial.printf("  唤醒原因    %s\n", pulse_wake_cause_str(st.wake_cause));
    Serial.printf("  无活动时长  %.1f 秒\n", st.idle_ms / 1000.0);
    Serial.printf("  运行时长    %u 秒\n", st.uptime_s);
    if (st.battery_percent >= 0) {
        Serial.printf("  电量        %d%% %s\n", st.battery_percent,
                      st.charging ? "(充电中)" : "");
    } else {
        Serial.println("  电量        未知（无电池）");
    }
    Serial.printf("  低电告警    %s\n", st.battery_warned ? "已触发" : "未触发");
    Serial.printf("  深睡被否决  %u 次\n", st.sleep_blocked_count);
    Serial.println();
    Serial.println("  三级功耗配置：");
    Serial.printf("    Active -> Idle   无触摸 %u ms\n", c->power.screen_dim_ms);
    Serial.printf("    Idle -> 深睡     %s\n",
                  c->power.deep_sleep_ms == 0
                      ? "永不（deep_sleep_ms=0）"
                      : String("静置 " + String(c->power.deep_sleep_ms) + " ms").c_str());
    Serial.printf("    低电告警 %u%%  危急 %u%%\n", c->power.battery_warn_pct,
                  c->power.battery_crit_pct);
    Serial.printf("    背光 %u%%\n", c->power.backlight_pct);
    Serial.println("=========================================");
}
