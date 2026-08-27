/**
 * @file pulse_upload.cpp
 * @brief 上传层实现：multipart 流式上传 + 指数退避
 */
#include "pulse_upload.h"

#include <Arduino.h>
#include <HTTPClient.h>
#include <WiFi.h>
#include <WiFiClient.h>
#include <WiFiClientSecure.h>

#include <stdlib.h>
#include <string.h>

#include "board_pins.h"
#include "config/pulse_config.h"
#include "hal/rtc/pulse_rtc.h"
#include "hal/storage/pulse_storage.h"
#include "net/wifi/pulse_wifi.h"

static const char *TAG = "upload";

static upload_state_t g_state = UPLOAD_IDLE;
static uint32_t g_sent_ok = 0;
static uint32_t g_sent_failed = 0;
static uint32_t g_retry_count = 0;
static uint32_t g_total_bytes = 0;
static int g_last_http = 0;
static char g_last_error[96] = {0};
static char g_current[96] = {0};
static uint32_t g_backoff_ms = 0;
static uint32_t g_backoff_until = 0;

/* 记录上次失败的段路径：换段时重置重试计数 */
static char g_retry_path[96] = {0};

/* ─────────────────────────────────────────────────────────────
 * MultipartStream
 *
 * 把 [前置信封][文件内容][后置信封] 拼成一条虚拟流。
 * HTTPClient 会按需 read()，我们按偏移决定从哪一段取数据 ——
 * 文件部分用 storage_read_chunk 按需拉取，全程不需要整块缓冲。
 * ───────────────────────────────────────────────────────────── */
class MultipartStream : public Stream {
   public:
    MultipartStream(const char *head, size_t head_len, const char *path, size_t file_len,
                    const char *tail, size_t tail_len)
        : _head(head), _head_len(head_len), _path(path), _file_len(file_len), _tail(tail),
          _tail_len(tail_len), _pos(0), _buf_off(0), _buf_len(0), _buf(nullptr) {
        /* 缓冲放堆上：4KB 压栈会在 HTTPS 时与 mbedTLS 握手栈叠加导致爆栈。
         * 上传是单任务串行的，无需考虑并发，malloc 一次即可。 */
        _buf = (uint8_t *)malloc(BUF_SIZE);
    }

    ~MultipartStream() {
        if (_buf) free(_buf);
    }

    size_t total() const { return _head_len + _file_len + _tail_len; }

    int available() override {
        size_t t = total();
        return _pos >= t ? 0 : (int)(t - _pos);
    }

    int read() override {
        uint8_t b;
        return readBytes((char *)&b, 1) == 1 ? b : -1;
    }

    int peek() override {
        /* HTTPClient 不依赖 peek，实现成保守值即可 */
        return -1;
    }

    void flush() override {}

    /* 写方向不使用，但 Print 基类要求实现 */
    size_t write(uint8_t) override { return 0; }
    size_t write(const uint8_t *, size_t) override { return 0; }

    /* HTTPClient 实际调用的是这个批量读取接口 */
    size_t readBytes(char *dst, size_t len) {
        size_t done = 0;
        while (done < len && _pos < total()) {
            /* 1) 前置信封 */
            if (_pos < _head_len) {
                size_t n = _head_len - _pos;
                if (n > len - done) n = len - done;
                memcpy(dst + done, _head + _pos, n);
                _pos += n;
                done += n;
                continue;
            }

            /* 2) 文件体：按 4KB 分块从存储拉取 */
            size_t file_pos = _pos - _head_len;
            if (file_pos < _file_len) {
                /* 内部小缓冲已耗尽则续拉一块 */
                if (_buf_off >= _buf_len) {
                    size_t want = _file_len - file_pos;
                    if (want > BUF_SIZE) want = BUF_SIZE;
                    _buf_len = storage_read_chunk(_path, file_pos, _buf, want);
                    _buf_off = 0;
                    if (_buf_len == 0) break; /* 读取失败，提前结束 */
                }
                size_t n = _buf_len - _buf_off;
                if (n > len - done) n = len - done;
                memcpy(dst + done, _buf + _buf_off, n);
                _buf_off += n;
                _pos += n;
                done += n;
                continue;
            }

            /* 3) 后置信封 */
            size_t tail_pos = _pos - _head_len - _file_len;
            if (tail_pos < _tail_len) {
                size_t n = _tail_len - tail_pos;
                if (n > len - done) n = len - done;
                memcpy(dst + done, _tail + tail_pos, n);
                _pos += n;
                done += n;
                continue;
            }
            break;
        }
        return done;
    }

   private:
    static const size_t BUF_SIZE = 4096; /* 4KB 与 SD 扇区对齐 */

    const char *_head;
    size_t _head_len;
    const char *_path;
    size_t _file_len;
    const char *_tail;
    size_t _tail_len;
    size_t _pos;

    /* 文件分块缓冲。堆分配（见构造函数注释），避免压栈。 */
    uint8_t *_buf;
    size_t _buf_off;
    size_t _buf_len;
};

/* ─────────── 退避 ─────────── */

static void reset_backoff(void) {
    const pulse_config_t *c = config_get();
    g_backoff_ms = c->cloud.retry_backoff_ms;
}

static void bump_backoff(void) {
    const pulse_config_t *c = config_get();
    if (g_backoff_ms == 0) g_backoff_ms = c->cloud.retry_backoff_ms;
    else g_backoff_ms *= 2;
    /* 上限借用 WiFi 的 reconnect_max_ms，避免再加一个配置项 */
    if (g_backoff_ms > c->wifi.reconnect_max_ms) g_backoff_ms = c->wifi.reconnect_max_ms;
}

bool pulse_upload_init(void) {
    const pulse_config_t *c = config_get();
    reset_backoff();

    if (c->cloud.base_url[0] == '\0') {
        g_state = UPLOAD_DISABLED;
        log_e("%s: cloud.base_url 未配置，上传功能关闭", TAG);
        return false;
    }
    if (c->cloud.auth_token[0] == '\0') {
        /* 不阻止启动：让用户能先跑通链路再补 token */
        log_w("%s: cloud.auth_token 为空，后端会返回 401", TAG);
    }

    g_state = UPLOAD_IDLE;
    log_i("%s: 上传层就绪 %s%s", TAG, c->cloud.base_url, c->cloud.upload_path);
    return true;
}

/* ─────────── 单段上传 ─────────── */

bool pulse_upload_one(const char *path, int *http_code_out) {
    const pulse_config_t *c = config_get();
    if (http_code_out) *http_code_out = 0;
    if (!path || path[0] == '\0') return false;

    size_t file_len = storage_file_size(path);
    if (file_len == 0) {
        snprintf(g_last_error, sizeof(g_last_error), "文件为空或不存在");
        return false;
    }

    /* 取文件名（不含目录）作为上传文件名。
     * 必须以 .wav 结尾：后端会校验扩展名与魔数一致。 */
    const char *base = strrchr(path, '/');
    base = base ? base + 1 : path;

    char filename[64];
    if (strncmp(path, "psram:", 6) == 0) {
        /* PSRAM 模式的路径是 psram:N，不是合法文件名，需另造一个 */
        snprintf(filename, sizeof(filename), "%010u.wav",
                 pulse_rtc_unix() ? pulse_rtc_unix() : (uint32_t)(millis() / 1000));
    } else {
        snprintf(filename, sizeof(filename), "%s", base);
    }

    /* recorded_at：优先用 RTC 的 RFC3339。时间不可信时不发该字段，
     * 让后端用它自己的 time.Now()，避免写入错误时间污染时间线。 */
    char recorded_at[40] = {0};
    bool have_time = pulse_rtc_rfc3339(recorded_at, sizeof(recorded_at));

    /* 时长（秒）：由 PCM 字节数反推，减去 44 字节 WAV 头 */
    uint32_t duration = 0;
    if (file_len > 44) {
        duration = (uint32_t)((file_len - 44) / AUDIO_BYTES_PER_SEC);
    }

    /* 组装 multipart 信封。boundary 用固定串即可 ——
     * 内容是二进制音频，不会恰好包含这个串。 */
    static const char BOUNDARY[] = "----PulseFormBoundary7MA4YWxkTrZu0gW";

    char head[768];
    int hn = 0;
    hn += snprintf(head + hn, sizeof(head) - hn,
                   "--%s\r\n"
                   "Content-Disposition: form-data; name=\"device_id\"\r\n\r\n"
                   "%s\r\n",
                   BOUNDARY, c->cloud.device_id);
    hn += snprintf(head + hn, sizeof(head) - hn,
                   "--%s\r\n"
                   "Content-Disposition: form-data; name=\"duration\"\r\n\r\n"
                   "%u\r\n",
                   BOUNDARY, duration);
    if (have_time) {
        hn += snprintf(head + hn, sizeof(head) - hn,
                       "--%s\r\n"
                       "Content-Disposition: form-data; name=\"recorded_at\"\r\n\r\n"
                       "%s\r\n",
                       BOUNDARY, recorded_at);
    }
    /* 文件字段必须叫 "file"，见后端 c.FormFile("file") */
    hn += snprintf(head + hn, sizeof(head) - hn,
                   "--%s\r\n"
                   "Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n"
                   "Content-Type: audio/wav\r\n\r\n",
                   BOUNDARY, filename);

    char tail[64];
    int tn = snprintf(tail, sizeof(tail), "\r\n--%s--\r\n", BOUNDARY);

    MultipartStream body(head, (size_t)hn, path, file_len, tail, (size_t)tn);

    /* 拼 URL */
    char url[256];
    snprintf(url, sizeof(url), "%s%s", c->cloud.base_url, c->cloud.upload_path);

    bool is_https = strncmp(c->cloud.base_url, "https://", 8) == 0;

    HTTPClient http;
    WiFiClient *client = nullptr;
    WiFiClientSecure *sclient = nullptr;

    if (is_https) {
        sclient = new WiFiClientSecure();
        if (!sclient) {
            snprintf(g_last_error, sizeof(g_last_error), "TLS 客户端分配失败");
            return false;
        }
        if (!c->cloud.verify_tls) {
            /* 跳过证书校验。仅用于自签名证书的开发环境，
             * 生产必须配置 CA 并把 verify_tls 设为 true。 */
            sclient->setInsecure();
        }
        client = sclient;
    } else {
        client = new WiFiClient();
        if (!client) {
            snprintf(g_last_error, sizeof(g_last_error), "客户端分配失败");
            return false;
        }
    }

    http.setTimeout((uint16_t)c->cloud.request_timeout_ms);
    http.setReuse(false);

    if (!http.begin(*client, url)) {
        snprintf(g_last_error, sizeof(g_last_error), "http.begin 失败: %s", url);
        delete client;
        return false;
    }

    char ctype[128];
    snprintf(ctype, sizeof(ctype), "multipart/form-data; boundary=%s", BOUNDARY);
    http.addHeader("Content-Type", ctype);

    if (c->cloud.auth_token[0]) {
        char auth[192];
        snprintf(auth, sizeof(auth), "%s %s", c->cloud.auth_scheme, c->cloud.auth_token);
        http.addHeader("Authorization", auth);
    }

    log_i("%s: 上传 %s (%u 字节, %us%s)", TAG, filename, (unsigned)file_len, duration,
          have_time ? "" : ", 无时间戳");

    int code = http.sendRequest("POST", (Stream *)&body, body.total());
    g_last_http = code;

    if (code > 0) {
        String resp = http.getString();
        if (code >= 200 && code < 300) {
            g_total_bytes += file_len;
            g_last_error[0] = '\0';
            log_i("%s: 成功 HTTP %d %s", TAG, code, resp.c_str());
        } else {
            /* 截断响应体，避免长错误信息挤爆日志 */
            snprintf(g_last_error, sizeof(g_last_error), "HTTP %d: %.60s", code, resp.c_str());
            log_w("%s: 失败 %s", TAG, g_last_error);
            /* 针对性诊断：401 几乎一定是鉴权缺口 */
            if (code == 401) {
                log_w("%s: 401 说明 token 无效。后端目前只认用户级 JWT，", TAG);
                log_w("%s: 设备级 device_token 尚无中间件消费（需后端补齐）", TAG);
            } else if (code == 400) {
                log_w("%s: 400 多为 WAV 头与扩展名不符，用 M4 测试校验文件结构", TAG);
            }
        }
    } else {
        snprintf(g_last_error, sizeof(g_last_error), "连接失败: %s",
                 HTTPClient::errorToString(code).c_str());
        log_w("%s: %s", TAG, g_last_error);
    }

    http.end();
    delete client;

    if (http_code_out) *http_code_out = code;
    return code >= 200 && code < 300;
}

/* ─────────── 状态机 ─────────── */

void pulse_upload_tick(void) {
    const pulse_config_t *c = config_get();

    if (g_state == UPLOAD_DISABLED) return;

    /* 断网：暂停上传，段留在 spool 等重连。
     * 这正是需求里「断网时暂停上传，等待重连后统一上传」的实现点。 */
    if (!pulse_wifi_is_connected()) {
        g_state = UPLOAD_OFFLINE;
        return;
    }

    /* 退避期内不动作（用无符号比较避免 millis 回绕溢出，见 pulse_wifi.cpp）*/
    if (g_state == UPLOAD_BACKOFF && (uint32_t)(millis() - g_backoff_until) > INT32_MAX) {
        return;
    }

    /* 取队首段。storage_pending_at(0) 每次都返回当前最旧的一个，
     * 文件名是零填充时间戳，字典序即时间序，因此天然按录制顺序补传。 */
    char path[96];
    size_t size = 0;
    if (!storage_pending_at(0, path, sizeof(path), &size)) {
        g_state = UPLOAD_IDLE;
        g_current[0] = '\0';
        return;
    }

    /* 换段则重置重试计数 */
    if (strcmp(path, g_retry_path) != 0) {
        strncpy(g_retry_path, path, sizeof(g_retry_path) - 1);
        g_retry_path[sizeof(g_retry_path) - 1] = '\0';
        g_retry_count = 0;
        reset_backoff();
    }

    strncpy(g_current, path, sizeof(g_current) - 1);
    g_current[sizeof(g_current) - 1] = '\0';
    g_state = UPLOAD_SENDING;

    int code = 0;
    if (pulse_upload_one(path, &code)) {
        storage_delete(path);
        g_sent_ok++;
        g_retry_count = 0;
        g_retry_path[0] = '\0';
        reset_backoff();
        g_state = UPLOAD_IDLE;
        return;
    }

    g_retry_count++;

    /* 4xx（除 401/408/429）是客户端错误，重试无意义，直接移入 failed。
     * 401 例外：token 可能稍后被补上，值得重试。 */
    bool permanent = (code >= 400 && code < 500 && code != 401 && code != 408 && code != 429);

    if (permanent || g_retry_count >= c->cloud.max_retry) {
        log_w("%s: %s 放弃重试（%s），移入 failed", TAG, path,
              permanent ? "客户端错误" : "超过重试上限");
        storage_move_to_failed(path);
        g_sent_failed++;
        g_retry_count = 0;
        g_retry_path[0] = '\0';
        reset_backoff();
        g_state = UPLOAD_IDLE;
        return;
    }

    bump_backoff();
    g_backoff_until = millis() + g_backoff_ms;
    g_state = UPLOAD_BACKOFF;
    log_w("%s: 第 %u/%u 次失败，退避 %u ms", TAG, g_retry_count, c->cloud.max_retry, g_backoff_ms);
}

bool pulse_upload_get_status(upload_status_t *out) {
    if (!out) return false;
    memset(out, 0, sizeof(*out));
    out->state = g_state;
    out->sent_ok = g_sent_ok;
    out->sent_failed = g_sent_failed;
    out->retry_count = g_retry_count;
    out->total_bytes_sent = g_total_bytes;
    out->last_http_code = g_last_http;
    out->backoff_ms = g_backoff_ms;
    strncpy(out->last_error, g_last_error, sizeof(out->last_error) - 1);
    strncpy(out->current_path, g_current, sizeof(out->current_path) - 1);
    return true;
}

upload_state_t pulse_upload_state(void) {
    return g_state;
}

const char *pulse_upload_state_str(upload_state_t s) {
    switch (s) {
        case UPLOAD_OFFLINE: return "离线暂停";
        case UPLOAD_SENDING: return "上传中";
        case UPLOAD_BACKOFF: return "退避等待";
        case UPLOAD_DISABLED: return "未配置";
        default: return "空闲";
    }
}

bool pulse_upload_queue_empty(void) {
    return storage_pending_count() == 0;
}

void pulse_upload_dump(void) {
    const pulse_config_t *c = config_get();
    upload_status_t st;
    pulse_upload_get_status(&st);

    Serial.println();
    Serial.println("================ 上传层状态 ================");
    Serial.printf("  状态       %s\n", pulse_upload_state_str(st.state));
    Serial.printf("  目标       %s%s\n", c->cloud.base_url, c->cloud.upload_path);
    Serial.printf("  鉴权       %s %s\n", c->cloud.auth_scheme,
                  c->cloud.auth_token[0] ? "(已设置)" : "(未设置，将 401)");
    Serial.printf("  device_id  %s\n", c->cloud.device_id);
    Serial.printf("  超时       %u ms  最大重试 %u\n", c->cloud.request_timeout_ms,
                  c->cloud.max_retry);
    Serial.println();
    Serial.printf("  成功       %u 段, %.2f MB\n", st.sent_ok, st.total_bytes_sent / 1048576.0);
    Serial.printf("  失败归档   %u 段\n", st.sent_failed);
    Serial.printf("  待上传     %d 段\n", storage_pending_count());
    if (st.current_path[0]) {
        Serial.printf("  当前       %s (重试 %u)\n", st.current_path, st.retry_count);
    }
    if (st.last_http_code) {
        Serial.printf("  最近 HTTP  %d\n", st.last_http_code);
    }
    if (st.last_error[0]) {
        Serial.printf("  最近错误   %s\n", st.last_error);
    }

    char rfc[40];
    if (pulse_rtc_rfc3339(rfc, sizeof(rfc))) {
        Serial.printf("  recorded_at %s\n", rfc);
    } else {
        Serial.println("  recorded_at 不可用（RTC 未校时，将由后端填当前时间）");
    }
    Serial.println("==========================================");
}
