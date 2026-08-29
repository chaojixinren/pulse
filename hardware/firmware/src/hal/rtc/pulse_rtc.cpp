/**
 * @file pulse_rtc.cpp
 * @brief BM8563 RTC 实现（PCF8563 兼容寄存器布局）
 */
#include "pulse_rtc.h"

#include <Arduino.h>
#include <Wire.h>
#include <esp_sntp.h>
#include <sys/time.h>

#include <string.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "rtc";

/* PCF8563 / BM8563 寄存器映射。
 * 时间字段全为 BCD 编码，且各自有效位宽不同（见掩码）。 */
#define REG_CTRL_STATUS1 0x00
#define REG_CTRL_STATUS2 0x01
#define REG_VL_SECONDS 0x02 /* bit7 = VL：置位表示时钟完整性丢失 */
#define REG_MINUTES 0x03
#define REG_HOURS 0x04
#define REG_DAYS 0x05
#define REG_WEEKDAYS 0x06
#define REG_CENTURY_MONTHS 0x07 /* bit7 = 世纪位 */
#define REG_YEARS 0x08

/* 各字段的有效位掩码。高位是标志位或保留位，读出后必须屏蔽，
 * 否则 BCD 解码会得到荒谬的值。 */
#define MASK_SECONDS 0x7F
#define MASK_MINUTES 0x7F
#define MASK_HOURS 0x3F
#define MASK_DAYS 0x3F
#define MASK_WEEKDAYS 0x07
#define MASK_MONTHS 0x1F

#define VL_BIT 0x80
#define CENTURY_BIT 0x80

static bool g_ready = false;
static pulse_rtc_source_t g_source = PULSE_RTC_SRC_NONE;
static bool g_valid = false;

static inline uint8_t bcd2dec(uint8_t b) {
    return (uint8_t)((b >> 4) * 10 + (b & 0x0F));
}

static inline uint8_t dec2bcd(uint8_t d) {
    return (uint8_t)(((d / 10) << 4) | (d % 10));
}

static bool reg_read(uint8_t reg, uint8_t *buf, size_t len) {
    Wire.beginTransmission(I2C_ADDR_BM8563);
    Wire.write(reg);
    if (Wire.endTransmission(false) != 0) return false;
    size_t got = Wire.requestFrom((uint8_t)I2C_ADDR_BM8563, (uint8_t)len);
    if (got != len) return false;
    for (size_t i = 0; i < len; i++) buf[i] = Wire.read();
    return true;
}

static bool reg_write(uint8_t reg, const uint8_t *buf, size_t len) {
    Wire.beginTransmission(I2C_ADDR_BM8563);
    Wire.write(reg);
    for (size_t i = 0; i < len; i++) Wire.write(buf[i]);
    return Wire.endTransmission() == 0;
}

static bool reg_write1(uint8_t reg, uint8_t val) {
    return reg_write(reg, &val, 1);
}

/* 把 UTC 的 struct tm 转成 time_t（epoch 秒）。
 * mktime 按本地时区解释 tm，而 RTC 内部存的是 UTC，必须先把 TZ 临时切到
 * UTC0 再转换，否则会引入时区偏移（冷启动恢复时间时尤其隐蔽）。
 * timegm 在裁剪版 newlib 中不可用，故用 getenv/setenv 切换。 */
static time_t utc_mktime(const struct tm *t) {
    char *old_tz = getenv("TZ");
    char saved[64] = {0};
    if (old_tz) strncpy(saved, old_tz, sizeof(saved) - 1);

    setenv("TZ", "UTC0", 1);
    tzset();
    struct tm tmp = *t;
    time_t epoch = mktime(&tmp);

    if (saved[0]) setenv("TZ", saved, 1);
    else unsetenv("TZ");
    tzset();
    return epoch;
}

bool pulse_rtc_init(void) {
    /* I2C 总线可能已由其他模块（PMU）初始化过。
     * ESP32 Arduino core 的 Wire.begin() 内部有初始化标志，重复调用是安全的，
     * 不会重置已配置的 I2C 外设寄存器。 */
    Wire.begin(PIN_I2C_SDA, PIN_I2C_SCL, 100000U);

    /* 探测：读控制寄存器。BM8563 上电后 CTRL1 应为 0x00 或 0x08。 */
    uint8_t ctrl = 0;
    if (!reg_read(REG_CTRL_STATUS1, &ctrl, 1)) {
        log_e("%s: BM8563 无应答（地址 0x%02X）", TAG, I2C_ADDR_BM8563);
        return false;
    }

    /* 确保时钟在运行：CTRL1 的 STOP 位（bit5）必须为 0。 */
    if (ctrl & 0x20) {
        log_w("%s: 时钟处于停止状态，正在启动", TAG);
        reg_write1(REG_CTRL_STATUS1, (uint8_t)(ctrl & ~0x20));
    }

    /* 关闭报警与定时器中断，避免 INT 脚被意外拉低。
     * 本项目用 AXP2101 的 IRQ 做唤醒，不用 RTC 中断。 */
    reg_write1(REG_CTRL_STATUS2, 0x00);

    g_ready = true;

    /* 检查 VL 标志。这是判断时间可信度的唯一依据。 */
    uint8_t sec_reg = 0;
    if (reg_read(REG_VL_SECONDS, &sec_reg, 1)) {
        if (sec_reg & VL_BIT) {
            g_valid = false;
            g_source = PULSE_RTC_SRC_NONE;
            log_w("%s: VL 标志置位 —— 时钟完整性丢失（掉电过久或首次上电）", TAG);
            log_w("%s: 时间不可信，需 SNTP 校时后才能上传", TAG);
        } else {
            /* 时间可信，同步到系统时钟。RTC 存的是 UTC，用 utc_mktime 避免
             * mktime 按本地时区解释引入偏移（与 pulse_rtc_set_utc 保持一致）。 */
            struct tm t;
            if (pulse_rtc_get_utc(&t)) {
                time_t epoch = utc_mktime(&t);
                struct timeval tv = {.tv_sec = epoch, .tv_usec = 0};
                settimeofday(&tv, nullptr);
                g_valid = true;
                g_source = PULSE_RTC_SRC_HARDWARE;
                log_i("%s: 从 BM8563 恢复时间 %04d-%02d-%02d %02d:%02d:%02d UTC",
                      TAG, t.tm_year + 1900, t.tm_mon + 1, t.tm_mday, t.tm_hour, t.tm_min,
                      t.tm_sec);
            }
        }
    }

    log_i("%s: BM8563 就绪，时间来源 %s", TAG, pulse_rtc_source_str(g_source));
    return true;
}

bool pulse_rtc_is_ready(void) {
    return g_ready;
}

bool pulse_rtc_time_valid(void) {
    return g_valid;
}

pulse_rtc_source_t pulse_rtc_source(void) {
    return g_source;
}

const char *pulse_rtc_source_str(pulse_rtc_source_t s) {
    switch (s) {
        case PULSE_RTC_SRC_HARDWARE: return "BM8563 硬件时钟";
        case PULSE_RTC_SRC_SNTP: return "SNTP 已校时";
        default: return "不可信（未校时）";
    }
}

bool pulse_rtc_get_utc(struct tm *out) {
    if (!g_ready || !out) return false;

    /* 一次性连读 7 个寄存器：避免跨秒读取造成的时间撕裂
     * （例如读到 10:59:59 的秒和 11:00:00 的分）。 */
    uint8_t b[7] = {0};
    if (!reg_read(REG_VL_SECONDS, b, sizeof(b))) return false;

    if (b[0] & VL_BIT) return false; /* 时间不可信 */

    memset(out, 0, sizeof(*out));
    out->tm_sec = bcd2dec(b[0] & MASK_SECONDS);
    out->tm_min = bcd2dec(b[1] & MASK_MINUTES);
    out->tm_hour = bcd2dec(b[2] & MASK_HOURS);
    out->tm_mday = bcd2dec(b[3] & MASK_DAYS);
    out->tm_wday = bcd2dec(b[4] & MASK_WEEKDAYS);
    out->tm_mon = (int)bcd2dec(b[5] & MASK_MONTHS) - 1; /* tm_mon 从 0 起 */

    /* 世纪位：0 表示 20xx，1 表示 19xx。
     * 本项目不可能运行在 19xx，故只处理 20xx/21xx。 */
    int year = bcd2dec(b[6]);
    out->tm_year = (b[5] & CENTURY_BIT) ? year : (year + 100);
    out->tm_isdst = 0;

    /* 基本合法性检查：BCD 解码出界说明读到脏数据 */
    if (out->tm_sec > 59 || out->tm_min > 59 || out->tm_hour > 23 || out->tm_mday < 1 ||
        out->tm_mday > 31 || out->tm_mon < 0 || out->tm_mon > 11) {
        log_w("%s: 读到非法时间值，寄存器可能未初始化", TAG);
        return false;
    }
    return true;
}

bool pulse_rtc_set_utc(const struct tm *t) {
    if (!g_ready || !t) return false;

    uint8_t b[7];
    /* 写秒寄存器时 VL 位写 0，即声明「时间现在可信」。 */
    b[0] = dec2bcd((uint8_t)t->tm_sec) & MASK_SECONDS;
    b[1] = dec2bcd((uint8_t)t->tm_min) & MASK_MINUTES;
    b[2] = dec2bcd((uint8_t)t->tm_hour) & MASK_HOURS;
    b[3] = dec2bcd((uint8_t)t->tm_mday) & MASK_DAYS;
    b[4] = dec2bcd((uint8_t)t->tm_wday) & MASK_WEEKDAYS;

    int full_year = t->tm_year + 1900;
    uint8_t month = dec2bcd((uint8_t)(t->tm_mon + 1)) & MASK_MONTHS;
    if (full_year >= 2100) month |= CENTURY_BIT;
    b[5] = month;
    b[6] = dec2bcd((uint8_t)(full_year % 100));

    if (!reg_write(REG_VL_SECONDS, b, sizeof(b))) {
        log_e("%s: 写入时间失败", TAG);
        return false;
    }

    /* 同步系统时钟：RTC 存的是 UTC，用 utc_mktime 转 epoch。 */
    time_t epoch = utc_mktime(t);
    struct timeval tv = {.tv_sec = epoch, .tv_usec = 0};
    settimeofday(&tv, nullptr);

    g_valid = true;
    log_i("%s: 已写入 %04d-%02d-%02d %02d:%02d:%02d UTC", TAG, full_year, t->tm_mon + 1,
          t->tm_mday, t->tm_hour, t->tm_min, t->tm_sec);
    return true;
}

uint32_t pulse_rtc_unix(void) {
    if (!g_valid) return 0;
    time_t now = time(nullptr);
    /* 系统时钟已在 init/set 时同步为 UTC epoch */
    if (now < 1700000000) return 0; /* 早于 2023-11，显然未校时 */
    return (uint32_t)now;
}

bool pulse_rtc_rfc3339(char *buf, size_t cap) {
    if (!buf || cap < 21) return false;
    if (!g_valid) {
        buf[0] = '\0';
        return false;
    }

    time_t now = time(nullptr);
    if (now < 1700000000) {
        buf[0] = '\0';
        return false;
    }

    struct tm t;
    gmtime_r(&now, &t); /* 强制 UTC，与后端 time.Now().UTC() 对齐 */
    /* 后端用 time.Parse(time.RFC3339)，Z 后缀表示 UTC 零偏移 */
    snprintf(buf, cap, "%04d-%02d-%02dT%02d:%02d:%02dZ", t.tm_year + 1900, t.tm_mon + 1,
             t.tm_mday, t.tm_hour, t.tm_min, t.tm_sec);
    return true;
}

bool pulse_rtc_sync_sntp(uint32_t timeout_ms) {
    const pulse_config_t *c = config_get();

    /* SNTP 只能在联网后使用。这里不主动连 WiFi，由调用方保证。 */
    log_i("%s: 开始 SNTP 校时（服务器 %s）", TAG, c->ui.ntp_server);

    /* 用 UTC 拉取，本地时区仅在显示层应用。 */
    configTzTime("UTC0", c->ui.ntp_server, "time.nist.gov", "cn.pool.ntp.org");

    uint32_t t0 = millis();
    while (millis() - t0 < timeout_ms) {
        if (sntp_get_sync_status() == SNTP_SYNC_STATUS_COMPLETED) break;
        time_t now = time(nullptr);
        if (now > 1700000000) break; /* 已拿到合理时间 */
        delay(200);
    }

    time_t now = time(nullptr);
    if (now < 1700000000) {
        log_w("%s: SNTP 校时超时（%u ms）", TAG, timeout_ms);
        return false;
    }

    struct tm t;
    gmtime_r(&now, &t);

    /* 写回 RTC，这样下次断电重启也有正确时间 */
    if (!pulse_rtc_set_utc(&t)) {
        log_w("%s: SNTP 成功但写回 RTC 失败", TAG);
        /* 系统时钟已正确，仍算部分成功 */
        g_valid = true;
        g_source = PULSE_RTC_SRC_SNTP;
        return true;
    }

    g_source = PULSE_RTC_SRC_SNTP;
    log_i("%s: SNTP 校时完成并已写回 BM8563", TAG);
    return true;
}

void pulse_rtc_dump(void) {
    const pulse_config_t *c = config_get();

    Serial.println();
    Serial.println("================ RTC 状态 ================");
    Serial.printf("  芯片       BM8563 @ I2C 0x%02X\n", I2C_ADDR_BM8563);
    Serial.printf("  就绪       %s\n", g_ready ? "是" : "否（I2C 无应答）");

    if (!g_ready) {
        Serial.println("=========================================");
        return;
    }

    /* 打印原始寄存器，便于核对 BCD 解码是否正确 */
    uint8_t raw[9] = {0};
    if (reg_read(REG_CTRL_STATUS1, raw, sizeof(raw))) {
        Serial.print("  寄存器     ");
        for (size_t i = 0; i < sizeof(raw); i++) {
            Serial.printf("%02X ", raw[i]);
        }
        Serial.println();
        Serial.printf("             CTRL1=%02X CTRL2=%02X SEC=%02X(VL=%d)\n", raw[0], raw[1],
                      raw[2], (raw[2] & VL_BIT) ? 1 : 0);
    }

    Serial.printf("  VL 标志    %s\n",
                  g_valid ? "清除（时间可信）" : "置位（时间不可信，需校时）");
    Serial.printf("  时间来源   %s\n", pulse_rtc_source_str(g_source));

    struct tm t;
    if (pulse_rtc_get_utc(&t)) {
        Serial.printf("  RTC UTC    %04d-%02d-%02d %02d:%02d:%02d (周%d)\n", t.tm_year + 1900,
                      t.tm_mon + 1, t.tm_mday, t.tm_hour, t.tm_min, t.tm_sec, t.tm_wday);
    } else {
        Serial.println("  RTC UTC    读取失败或时间不可信");
    }

    char rfc[32];
    if (pulse_rtc_rfc3339(rfc, sizeof(rfc))) {
        Serial.printf("  RFC3339    %s  <- 上传用的 recorded_at\n", rfc);
    } else {
        Serial.println("  RFC3339    不可用（时间不可信，上传会被拒绝或时间错乱）");
    }

    /* 顺带显示本地时间，确认时区配置生效 */
    time_t now = time(nullptr);
    if (now > 1700000000) {
        setenv("TZ", c->ui.timezone, 1);
        tzset();
        struct tm lt;
        localtime_r(&now, &lt);
        Serial.printf("  本地时间   %04d-%02d-%02d %02d:%02d:%02d (TZ=%s)\n", lt.tm_year + 1900,
                      lt.tm_mon + 1, lt.tm_mday, lt.tm_hour, lt.tm_min, lt.tm_sec,
                      c->ui.timezone);
    }
    Serial.println("=========================================");
}
