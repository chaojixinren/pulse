/**
 * @file pulse_pmu.cpp
 * @brief AXP2101 电源管理实现（只读安全模式）
 */
#include "pulse_pmu.h"

#include <Arduino.h>
#include <Wire.h>

#define XPOWERS_CHIP_AXP2101
#include <XPowersLib.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "pmu";

static XPowersAXP2101 g_pmu;
static bool g_ready = false;
/* I2C 总线初始化标志：避免其他模块重复调用 Wire.begin() 导致正在进行的
 * I2C 事务被中断。pmu_init() 最先初始化总线（它在 pulse_app_init 中最先执行）。 */
static bool g_i2c_initialized = false;

/* IRQ 只置标志，所有 I2C 读取都推迟到任务上下文。
 * AXP2101 的 IRQ 是电平保持型：不清中断寄存器就一直拉低，
 * 因此这里用 volatile 标志而非计数，避免 ISR 内做任何耗时操作。 */
static volatile bool g_irq_pending = false;

static void IRAM_ATTR pmu_irq_handler(void) {
    g_irq_pending = true;
}

bool pmu_init(void) {
    const pulse_config_t *c = config_get();
    (void)c;

    /* I2C 总线可能已由其他模块（RTC/触摸）初始化过，Wire.begin 可重复调用。
     * 100kHz 是三颗从机的公共安全速率；CST816 在 400kHz 下偶有异常。 */
    Wire.begin(PIN_I2C_SDA, PIN_I2C_SCL, 100000U);
    g_i2c_initialized = true;

    if (!g_pmu.begin(Wire, I2C_ADDR_AXP2101, PIN_I2C_SDA, PIN_I2C_SCL)) {
        log_e("%s: AXP2101 无应答（地址 0x%02X）", TAG, I2C_ADDR_AXP2101);
        return false;
    }

    uint8_t id = g_pmu.getChipID();
    if (id != XPOWERS_AXP2101_CHIP_ID) {
        log_e("%s: CHIP_ID 不匹配，读到 0x%02X，期望 0x%02X", TAG, id, XPOWERS_AXP2101_CHIP_ID);
        return false;
    }

    /* 打开测量通道。这些只影响 ADC，不改变任何输出电压。 */
    g_pmu.enableBattDetection();
    g_pmu.enableBattVoltageMeasure();
    g_pmu.enableVbusVoltageMeasure();
    g_pmu.enableSystemVoltageMeasure();
    g_pmu.enableTemperatureMeasure();

    /* 中断配置：先全关再清标志，避免上电残留的中断立刻触发。 */
    g_pmu.disableIRQ(XPOWERS_AXP2101_ALL_IRQ);
    g_pmu.clearIrqStatus();
    g_pmu.enableIRQ(XPOWERS_AXP2101_PKEY_SHORT_IRQ | XPOWERS_AXP2101_PKEY_LONG_IRQ |
                    XPOWERS_AXP2101_VBUS_INSERT_IRQ | XPOWERS_AXP2101_VBUS_REMOVE_IRQ |
                    XPOWERS_AXP2101_BAT_INSERT_IRQ | XPOWERS_AXP2101_BAT_REMOVE_IRQ |
                    XPOWERS_AXP2101_BAT_CHG_START_IRQ | XPOWERS_AXP2101_BAT_CHG_DONE_IRQ |
                    XPOWERS_AXP2101_WARNING_LEVEL1_IRQ | XPOWERS_AXP2101_WARNING_LEVEL2_IRQ);

    /* GPIO6 低电平有效，PMIC 内部已有上拉，这里用输入+下降沿。 */
    pinMode(PIN_PMU_IRQ, INPUT_PULLUP);
    attachInterrupt(digitalPinToInterrupt(PIN_PMU_IRQ), pmu_irq_handler, FALLING);

    g_ready = true;
    log_i("%s: AXP2101 初始化完成（只读模式，未修改电源轨）", TAG);
    return true;
}

bool pmu_is_ready(void) {
    return g_ready;
}

static pmu_chg_state_t map_chg_state(void) {
    switch (g_pmu.getChargerStatus()) {
        case XPOWERS_AXP2101_CHG_TRI_STATE: return PMU_CHG_TRICKLE;
        case XPOWERS_AXP2101_CHG_PRE_STATE: return PMU_CHG_PRE;
        case XPOWERS_AXP2101_CHG_CC_STATE: return PMU_CHG_CC;
        case XPOWERS_AXP2101_CHG_CV_STATE: return PMU_CHG_CV;
        case XPOWERS_AXP2101_CHG_DONE_STATE: return PMU_CHG_DONE;
        default: return PMU_CHG_NOT_CHARGING;
    }
}

bool pmu_read(pmu_status_t *out) {
    if (!g_ready || !out) return false;

    out->battery_present = g_pmu.isBatteryConnect();
    /* getBatteryPercent/getBattVoltage 在无电池时返回 -1/0，这里保持该语义。 */
    out->percent = out->battery_present ? g_pmu.getBatteryPercent() : -1;
    out->batt_mv = out->battery_present ? g_pmu.getBattVoltage() : 0;
    out->vbus_present = g_pmu.isVbusIn();
    out->vbus_mv = out->vbus_present ? g_pmu.getVbusVoltage() : 0;
    out->charging = g_pmu.isCharging();
    out->chg_state = out->charging ? map_chg_state() : PMU_CHG_NOT_CHARGING;
    out->die_temp_c = g_pmu.getTemperature();
    return true;
}

int pmu_poll_events(pmu_event_t *buf, int max) {
    if (!g_ready || !buf || max <= 0) return 0;
    if (!g_irq_pending) return 0;

    /* 先清标志再读状态：若在读的过程中来了新中断，标志会被重新置起，
     * 下次调用仍能捕获，不会丢事件。 */
    g_irq_pending = false;

    g_pmu.getIrqStatus();
    int n = 0;

    /* 顺序有意：按键优先，低电告警最后，保证 buf 满时先保住交互事件。 */
    if (n < max && g_pmu.isPekeyShortPressIrq()) buf[n++] = PMU_EVT_PWRKEY_SHORT;
    if (n < max && g_pmu.isPekeyLongPressIrq()) buf[n++] = PMU_EVT_PWRKEY_LONG;
    if (n < max && g_pmu.isVbusInsertIrq()) buf[n++] = PMU_EVT_VBUS_INSERT;
    if (n < max && g_pmu.isVbusRemoveIrq()) buf[n++] = PMU_EVT_VBUS_REMOVE;
    if (n < max && g_pmu.isBatInsertIrq()) buf[n++] = PMU_EVT_BAT_INSERT;
    if (n < max && g_pmu.isBatRemoveIrq()) buf[n++] = PMU_EVT_BAT_REMOVE;
    if (n < max && g_pmu.isBatChargeStartIrq()) buf[n++] = PMU_EVT_CHG_START;
    if (n < max && g_pmu.isBatChargeDoneIrq()) buf[n++] = PMU_EVT_CHG_DONE;
    if (n < max && g_pmu.isDropWarningLevel1Irq()) buf[n++] = PMU_EVT_LOW_WARN;
    if (n < max && g_pmu.isDropWarningLevel2Irq()) buf[n++] = PMU_EVT_LOW_CRIT;

    /* 必须清中断，否则 IRQ 引脚保持低电平，下降沿不再触发。 */
    g_pmu.clearIrqStatus();
    return n;
}

const char *pmu_event_str(pmu_event_t e) {
    switch (e) {
        case PMU_EVT_PWRKEY_SHORT: return "PWR键短按";
        case PMU_EVT_PWRKEY_LONG: return "PWR键长按";
        case PMU_EVT_VBUS_INSERT: return "USB插入";
        case PMU_EVT_VBUS_REMOVE: return "USB拔出";
        case PMU_EVT_BAT_INSERT: return "电池接入";
        case PMU_EVT_BAT_REMOVE: return "电池移除";
        case PMU_EVT_CHG_START: return "开始充电";
        case PMU_EVT_CHG_DONE: return "充电完成";
        case PMU_EVT_LOW_WARN: return "低电告警";
        case PMU_EVT_LOW_CRIT: return "电量危急";
        default: return "未知事件";
    }
}

const char *pmu_chg_state_str(pmu_chg_state_t s) {
    switch (s) {
        case PMU_CHG_TRICKLE: return "涓流充电";
        case PMU_CHG_PRE: return "预充电";
        case PMU_CHG_CC: return "恒流充电";
        case PMU_CHG_CV: return "恒压充电";
        case PMU_CHG_DONE: return "已充满";
        case PMU_CHG_NOT_CHARGING: return "未充电";
        default: return "未知";
    }
}

void pmu_dump_rails(void) {
    if (!g_ready) {
        Serial.println("  PMU 未就绪，无法读取电源轨");
        return;
    }

    Serial.println();
    Serial.println("============== AXP2101 电源轨（只读）==============");
    Serial.println("  实际使用：VRTC → RTC（always-on），DCDC1 → 3.3V 系统轨");
    Serial.println("  其余轨全部悬空，不接负载。");
    Serial.println();
    Serial.printf("  %-8s %-8s %s\n", "轨", "状态", "电压");
    /* VRTC 是 AXP2101 的 always-on RTC 供电轨，固定输出、不可调。
     * XPowersLib 未暴露读取接口（无 isEnableVRTC/getVrtcVoltage，
     * getPowerChannelVoltage 也只支持 DCDC/ALDO），故只标注状态不读电压。 */
    Serial.printf("  %-8s %-8s %s  ← RTC 专用（always-on，固定输出）\n", "VRTC", "开", "--");
    Serial.printf("  %-8s %-8s %u mV  ← 3.3V 系统轨（ESP32/LCD/TP/SD）\n", "DCDC1",
                  g_pmu.isEnableDC1() ? "开" : "关", g_pmu.getDC1Voltage());
    Serial.println();
    Serial.println("  以下轨未使用（悬空）：");
    Serial.printf("    DCDC3  %s %u mV\n", g_pmu.isEnableDC3() ? "开" : "关",
                  g_pmu.getDC3Voltage());
    Serial.printf("    ALDO1  %s %u mV\n", g_pmu.isEnableALDO1() ? "开" : "关",
                  g_pmu.getALDO1Voltage());
    Serial.printf("    ALDO2  %s %u mV\n", g_pmu.isEnableALDO2() ? "开" : "关",
                  g_pmu.getALDO2Voltage());
    Serial.printf("    ALDO3  %s %u mV\n", g_pmu.isEnableALDO3() ? "开" : "关",
                  g_pmu.getALDO3Voltage());
    Serial.printf("    ALDO4  %s %u mV\n", g_pmu.isEnableALDO4() ? "开" : "关",
                  g_pmu.getALDO4Voltage());
    Serial.printf("    BLDO1  %s %u mV\n", g_pmu.isEnableBLDO1() ? "开" : "关",
                  g_pmu.getBLDO1Voltage());
    Serial.printf("    BLDO2  %s %u mV\n", g_pmu.isEnableBLDO2() ? "开" : "关",
                  g_pmu.getBLDO2Voltage());
    Serial.println("=================================================");
}

bool pmu_i2c_is_initialized(void) {
    return g_i2c_initialized;
}

void pmu_shutdown(void) {
    if (!g_ready) return;
    log_w("%s: 执行软关机", TAG);
    delay(50); /* 给串口留出输出时间 */
    g_pmu.shutdown();
}
