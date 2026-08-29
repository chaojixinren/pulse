/**
 * @file pulse_claim.h
 * @brief 设备配对：用 6 位绑定码换取 device_token
 *
 * 调用时机
 * --------
 * 1. 首次开机，device_token 为空
 * 2. 恢复出厂设置，NVS 被清空
 * 3. 用户在 UI 手动触发"重新绑定"
 *
 * 流程
 * ----
 * 1. 用户在 App 生成 6 位绑定码（10 分钟、一次性）
 * 2. 固件调用 pulse_claim_request(bind_code)
 * 3. 成功后 device_token 自动写入 NVS（覆盖 config.json）
 * 4. 需重启或调用 config_reload() 让上传层生效
 *
 * 错误处理
 * ---------
 * - 401: 绑定码无效/过期/已使用/被其他设备占用
 * - 网络错误: 返回 false，调用方可重试
 * - 成功: 返回 true，token 已落盘
 */
#ifndef PULSE_CLAIM_H
#define PULSE_CLAIM_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    char device_id[64];      /* 读自 config，空则用 MAC 生成 */
    char device_name[64];    /* 例: "我的 Pulse"，可用户输入 */
    char bind_code[8];       /* 6 位数字码 */
    char device_token[128];  /* 输出：换到的 token */
} claim_request_t;

typedef enum {
    CLAIM_OK = 0,            /* 成功，token 已写入 NVS */
    CLAIM_INVALID_CODE,      /* 401: 绑定码无效/过期 */
    CLAIM_NETWORK_ERROR,     /* 网络或 HTTP 错误 */
    CLAIM_JSON_ERROR,        /* 响应解析失败 */
    CLAIM_NVS_ERROR,         /* token 拿到了但 NVS 写入失败 */
} claim_result_t;

/**
 * 发起配对请求。
 * 成功时自动调用 config_save_token("Device", token)，无需调用方再写 NVS。
 *
 * @param req  输入：device_id、device_name、bind_code；输出：device_token
 * @return claim_result_t
 */
claim_result_t pulse_claim_request(claim_request_t *req);

/**
 * 判断是否需要配对（device_token 为空或 upload 层报 401）。
 * 供启动逻辑使用。
 */
bool pulse_claim_is_needed(void);

#ifdef __cplusplus
}
#endif

#endif // PULSE_CLAIM_H
