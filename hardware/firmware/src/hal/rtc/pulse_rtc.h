/**
 * @file pulse_rtc.h
 * @brief BM8563 RTC HAL (I2C 0x51) + SNTP 校时
 *
 * 为什么需要外部 RTC
 * ------------------
 * 上传时后端要求 recorded_at 为 RFC3339 时间戳。ESP32 内部时钟在深睡与
 * 断电后会丢失，若只靠开机后 SNTP 校时，那么「断网 + 重启」的场景下
 * 录音的时间戳会全错，云端时间线就乱了。BM8563 带纽扣电池，掉电保时。
 *
 * BM8563 与 PCF8563 寄存器兼容，本驱动按 PCF8563 布局实现。
 * 关键点：所有时间字段都是 **BCD 编码**，且秒寄存器的最高位是
 * VL（Voltage Low）标志 —— 该位为 1 表示时钟完整性无保证（掉电过久），
 * 此时读到的时间不可信，必须重新校时。这是最容易被忽略的一点。
 *
 * 时区处理
 * --------
 * RTC 内部一律存 **UTC**，避免夏令时与时区变更造成的歧义。
 * 本地时间显示由 ui.timezone (POSIX TZ) 在展示层转换。
 * 上传的 recorded_at 也用 UTC，与后端 time.Now().UTC() 一致。
 */
#ifndef PULSE_RTC_H
#define PULSE_RTC_H

#include <stdbool.h>
#include <stdint.h>
#include <time.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    PULSE_RTC_SRC_NONE = 0,  /* 时间不可信 */
    PULSE_RTC_SRC_HARDWARE,  /* 来自 BM8563 */
    PULSE_RTC_SRC_SNTP,      /* 本次开机已 SNTP 校时 */
} pulse_rtc_source_t;

/**
 * 初始化 RTC。会读取 VL 标志判断时间是否可信，
 * 可信则同步到系统时钟（settimeofday）。
 * @return false 表示 I2C 无应答
 */
bool pulse_rtc_init(void);

bool pulse_rtc_is_ready(void);

/**
 * 时间是否可信。
 * false 表示 VL 标志置位（掉电过久）或从未校时，
 * 此时 recorded_at 不应上传，或需标记为不可信。
 */
bool pulse_rtc_time_valid(void);

pulse_rtc_source_t pulse_rtc_source(void);
const char *pulse_rtc_source_str(pulse_rtc_source_t s);

/** 读取 RTC 的 UTC 时间。失败或时间不可信返回 false。 */
bool pulse_rtc_get_utc(struct tm *out);

/** 写入 RTC（入参为 UTC），同时清除 VL 标志并同步系统时钟。 */
bool pulse_rtc_set_utc(const struct tm *t);

/** 当前 Unix 时间戳（UTC 秒）。时间不可信时返回 0。 */
uint32_t pulse_rtc_unix(void);

/**
 * 生成 RFC3339 UTC 字符串，用于上传的 recorded_at 字段。
 * 格式: 2026-08-26T11:22:33Z （后端用 time.Parse(time.RFC3339) 解析）
 * @return false 表示时间不可信
 */
bool pulse_rtc_rfc3339(char *buf, size_t cap);

/**
 * 通过 SNTP 校时并写回 RTC。需已联网。
 * @param timeout_ms 等待 SNTP 响应的超时
 * @return false 表示超时或未联网
 */
bool pulse_rtc_sync_sntp(uint32_t timeout_ms);

/** 串口打印 RTC 诊断（含 VL 标志与寄存器原始值）*/
void pulse_rtc_dump(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_RTC_H */
