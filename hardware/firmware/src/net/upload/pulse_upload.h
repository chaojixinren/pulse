/**
 * @file pulse_upload.h
 * @brief 上传层：spool 队列 → multipart/form-data → POST /api/v1/device/audio/upload
 *
 * 与后端的契约（读代码得来，非猜测）
 * ----------------------------------
 * backend/internal/api/audio.go 的 Upload handler 要求：
 *   - multipart 表单，文件字段名必须是 "file"
 *   - 文件名扩展必须是 .wav/.mp3/.m4a 之一，且**与文件头魔数一致**
 *     （detectAudioExt 会校验 RIFF/WAVE，不一致直接 400）
 *   - 可选字段：device_id、duration（整数秒）、recorded_at（RFC3339）
 *   - 鉴权见下方说明
 * 成功返回 {"session_id": "...", "status": "pending"}
 *
 * 鉴权现状
 * --------
 * 后端已补上设备级鉴权（middleware.DeviceAuth，消费 devices.device_token_hash）。
 * 设备走 POST /api/v1/device/audio/upload，请求头 "Authorization: Device <device_token>"，
 * 正好落在这里预留的 "<scheme> <token>" 两段可配设计上 —— 只改 config.json，无需重新编译。
 * 设备态上传时后端以 token 反解出的设备为准，表单里的 device_id 会被忽略。
 *
 * 尚未实现（固件侧）
 * ------------------
 * device_token 目前仍需人工填进 config.json。后端已提供免手抄的自助配对：
 *   POST /api/v1/device/claim  {device_id, bind_code, name} -> {device, device_token}
 * bind_code 是用户在 App 上生成的 6 位数字码（10 分钟、一次性）。
 * 拿到 token 后用 pulse_config 的 NVS 接口落盘即可。
 * 另有 POST /api/v1/device/heartbeat（响应捎带待执行指令与 server_time）
 * 与 POST /api/v1/device/commands/<id>/ack，固件均尚未接入。
 *
 * 为什么必须流式上传
 * ------------------
 * 一段 30 秒音频 = 960KB，而 S3 内部 RAM 只有 320KB。
 * 整块读入必然 OOM，所以用自定义 Stream 把 multipart 信封与文件体
 * 拼成一条虚拟流，交给 HTTPClient::sendRequest(type, Stream*, size)
 * 按 4KB 粒度边读边发。
 *
 * 断网语义
 * --------
 * 未联网时 upload_tick() 直接返回，段留在 spool 里不动。
 * 联网后按文件名（时间戳）顺序补传，保证云端时间线不乱序。
 */
#ifndef PULSE_UPLOAD_H
#define PULSE_UPLOAD_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    UPLOAD_IDLE = 0,      /* 队列为空 */
    UPLOAD_OFFLINE,       /* 未联网，暂停上传 */
    UPLOAD_SENDING,
    UPLOAD_BACKOFF,       /* 失败后退避等待 */
    UPLOAD_DISABLED,      /* 未配置 base_url 或 token */
} upload_state_t;

typedef struct {
    upload_state_t state;
    uint32_t sent_ok;         /* 成功上传段数 */
    uint32_t sent_failed;     /* 移入 failed 的段数 */
    uint32_t retry_count;     /* 当前段已重试次数 */
    uint32_t total_bytes_sent;
    int last_http_code;       /* 最近一次 HTTP 状态码，0 表示连接失败 */
    char last_error[96];
    char current_path[96];    /* 正在上传的段 */
    uint32_t backoff_ms;
} upload_status_t;

/** 初始化上传层。不发起网络请求。 */
bool pulse_upload_init(void);

/**
 * 驱动上传状态机。需周期调用（建议 200~1000ms）。
 * 内部自行判断是否联网、队列是否为空、是否在退避期。
 * 单次调用最多上传一个段，避免长时间占用调用线程。
 */
void pulse_upload_tick(void);

/**
 * 立刻尝试上传一个指定段（测试用）。
 * @param path storage_pending_at 返回的路径
 * @param http_code_out 输出 HTTP 状态码
 * @return true 表示后端接受（2xx）
 */
bool pulse_upload_one(const char *path, int *http_code_out);

bool pulse_upload_get_status(upload_status_t *out);
upload_state_t pulse_upload_state(void);
const char *pulse_upload_state_str(upload_state_t s);

/** 队列是否已清空（深睡前检查 sleep_requires_empty_spool 用）*/
bool pulse_upload_queue_empty(void);

void pulse_upload_dump(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_UPLOAD_H */
