/**
 * @file pulse_wifi.cpp
 * @brief WiFi 状态机 + AP 配网门户实现
 */
#include "pulse_wifi.h"

#include <Arduino.h>
#include <WebServer.h>
#include <WiFi.h>

#include <string.h>

#include "board_pins.h"
#include "config/pulse_config.h"

static const char *TAG = "wifi";

static pulse_wifi_state_t g_state = PULSE_WIFI_IDLE;
static uint32_t g_attempt_start = 0;   /* 本次连接尝试的起始时刻 */
static uint32_t g_backoff_ms = 0;      /* 当前退避间隔 */
static uint32_t g_backoff_until = 0;   /* 退避结束时刻 */
static uint32_t g_connect_attempts = 0;
static uint32_t g_disconnect_count = 0;
static uint32_t g_connected_since = 0;
static uint32_t g_consecutive_failures = 0;

static WebServer *g_server = nullptr;
static char g_ap_ssid[32] = {0};

/* 配网完成后的「延迟切换」标志：handle_save 运行在 g_server->handleClient()
 * 的调用栈内，若直接 connect → stop_provisioning → delete g_server 会造成
 * use-after-free。故只记录待连接凭据，由 tick 在 handleClient 返回后切换。 */
static volatile bool g_provision_ready = false;
static char g_pending_ssid[64] = {0};
static char g_pending_pass[64] = {0};

/* ─────────── 内部工具 ─────────── */

static void reset_backoff(void) {
    const pulse_config_t *c = config_get();
    g_backoff_ms = c->wifi.reconnect_min_ms;
}

static void bump_backoff(void) {
    const pulse_config_t *c = config_get();
    if (g_backoff_ms == 0) g_backoff_ms = c->wifi.reconnect_min_ms;
    else g_backoff_ms *= 2;
    if (g_backoff_ms > c->wifi.reconnect_max_ms) g_backoff_ms = c->wifi.reconnect_max_ms;
}

static void begin_attempt(const char *ssid, const char *password) {
    g_connect_attempts++;
    g_attempt_start = millis();
    g_state = PULSE_WIFI_CONNECTING;
    WiFi.mode(WIFI_STA);
    /* 关闭 WiFi 省电模式：默认的 modem sleep 会让上传延迟抖动很大。
     * 功耗由三级功耗状态机在 Idle/深睡阶段整体关闭射频来控制。 */
    WiFi.setSleep(false);
    WiFi.begin(ssid, password);
    log_i("%s: 正在连接 \"%s\"（第 %u 次尝试）", TAG, ssid, g_connect_attempts);
}

/* ─────────── 初始化 ─────────── */

bool pulse_wifi_init(void) {
    const pulse_config_t *c = config_get();

    reset_backoff();
    g_consecutive_failures = 0;

    if (c->wifi.ssid[0] == '\0') {
        g_state = PULSE_WIFI_NO_CREDENTIALS;
        log_w("%s: 未配置 SSID，需要配网", TAG);
        return true; /* 不是错误，只是需要配网 */
    }

    begin_attempt(c->wifi.ssid, c->wifi.password);
    return true;
}

/* ─────────── 状态机 ─────────── */

void pulse_wifi_tick(void) {
    const pulse_config_t *c = config_get();

    /* AP 配网模式下只处理 HTTP 请求，不做 STA 状态判定 */
    if (g_state == PULSE_WIFI_AP_PROVISION) {
        if (g_server) g_server->handleClient();
        /* handleClient 返回后处理配网完成：此时 delete g_server 是安全的
         * （不在请求回调的调用栈内）。 */
        if (g_provision_ready) {
            g_provision_ready = false;
            pulse_wifi_stop_provisioning();
            pulse_wifi_connect(g_pending_ssid, g_pending_pass);
        }
        return;
    }

    switch (g_state) {
        case PULSE_WIFI_CONNECTING:
            if (WiFi.status() == WL_CONNECTED) {
                g_state = PULSE_WIFI_CONNECTED;
                g_connected_since = millis();
                g_consecutive_failures = 0;
                reset_backoff();
                log_i("%s: 已连接 IP=%s RSSI=%d dBm", TAG, WiFi.localIP().toString().c_str(),
                      WiFi.RSSI());
            } else if (millis() - g_attempt_start >= c->wifi.connect_timeout_ms) {
                /* 超时：进入退避等待，而不是立刻重试 */
                g_consecutive_failures++;
                bump_backoff();
                g_backoff_until = millis() + g_backoff_ms;
                g_state = PULSE_WIFI_RECONNECT_WAIT;
                WiFi.disconnect();
                log_w("%s: 连接超时（连续失败 %u 次），退避 %u ms 后重试", TAG,
                      g_consecutive_failures, g_backoff_ms);
            }
            break;

        case PULSE_WIFI_CONNECTED:
            if (WiFi.status() != WL_CONNECTED) {
                g_disconnect_count++;
                g_state = PULSE_WIFI_RECONNECT_WAIT;
                bump_backoff();
                g_backoff_until = millis() + g_backoff_ms;
                log_w("%s: 连接断开（累计 %u 次），退避 %u ms 后重连", TAG, g_disconnect_count,
                      g_backoff_ms);
            }
            break;

        case PULSE_WIFI_RECONNECT_WAIT:
            /* 用无符号比较判断退避是否到期，避免 millis() 回绕后强转 int32
             * 导致溢出（设备运行约 24.8 天后差值超过 INT32_MAX 会误判为未到期）。 */
            if ((uint32_t)(millis() - g_backoff_until) <= INT32_MAX) {
                if (c->wifi.ssid[0] == '\0') {
                    g_state = PULSE_WIFI_NO_CREDENTIALS;
                } else {
                    begin_attempt(c->wifi.ssid, c->wifi.password);
                }
            }
            break;

        case PULSE_WIFI_NO_CREDENTIALS:
        case PULSE_WIFI_IDLE:
        default:
            /* 等待调用方决定：配网或保持离线。
             * 离线期间录音仍继续，段留在 spool 队列里等联网后上传。 */
            break;
    }
}

bool pulse_wifi_is_connected(void) {
    return g_state == PULSE_WIFI_CONNECTED && WiFi.status() == WL_CONNECTED;
}

pulse_wifi_state_t pulse_wifi_state(void) {
    return g_state;
}

const char *pulse_wifi_state_str(pulse_wifi_state_t s) {
    switch (s) {
        case PULSE_WIFI_NO_CREDENTIALS: return "未配网";
        case PULSE_WIFI_CONNECTING: return "连接中";
        case PULSE_WIFI_CONNECTED: return "已连接";
        case PULSE_WIFI_RECONNECT_WAIT: return "退避等待";
        case PULSE_WIFI_AP_PROVISION: return "AP 配网中";
        default: return "空闲";
    }
}

bool pulse_wifi_get_status(pulse_wifi_status_t *out) {
    if (!out) return false;
    memset(out, 0, sizeof(*out));
    out->state = g_state;
    out->connected = pulse_wifi_is_connected();
    out->connect_attempts = g_connect_attempts;
    out->disconnect_count = g_disconnect_count;
    out->backoff_ms = g_backoff_ms;

    if (out->connected) {
        strncpy(out->ssid, WiFi.SSID().c_str(), sizeof(out->ssid) - 1);
        strncpy(out->ip, WiFi.localIP().toString().c_str(), sizeof(out->ip) - 1);
        out->rssi = WiFi.RSSI();
        out->connected_seconds = (millis() - g_connected_since) / 1000;
    } else if (g_state == PULSE_WIFI_AP_PROVISION) {
        strncpy(out->ssid, g_ap_ssid, sizeof(out->ssid) - 1);
        strncpy(out->ip, WiFi.softAPIP().toString().c_str(), sizeof(out->ip) - 1);
    } else {
        const pulse_config_t *c = config_get();
        strncpy(out->ssid, c->wifi.ssid, sizeof(out->ssid) - 1);
    }
    return true;
}

bool pulse_wifi_connect(const char *ssid, const char *password) {
    if (!ssid || ssid[0] == '\0') return false;
    if (g_state == PULSE_WIFI_AP_PROVISION) pulse_wifi_stop_provisioning();
    reset_backoff();
    g_consecutive_failures = 0;
    begin_attempt(ssid, password ? password : "");
    return true;
}

/* ─────────── AP 配网门户 ─────────── */

/* 配网页面。内联在固件里，不依赖 SD 卡上的文件 ——
 * 否则「没插卡」和「没配网」会同时发生，形成死锁。 */
static const char PROVISION_PAGE[] PROGMEM =
    "<!DOCTYPE html><html lang=\"zh\"><head>"
    "<meta charset=\"utf-8\">"
    "<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">"
    "<title>Pulse 配网</title><style>"
    "body{font-family:-apple-system,system-ui,sans-serif;margin:0;padding:24px;"
    "background:#101418;color:#e8eaed}"
    "h1{font-size:20px;color:#4fc3f7;margin:0 0 4px}"
    "p.sub{color:#78909c;font-size:13px;margin:0 0 20px}"
    "label{display:block;margin:14px 0 6px;font-size:14px;color:#b0bec5}"
    "input{width:100%;box-sizing:border-box;padding:12px;font-size:16px;"
    "border:1px solid #37474f;border-radius:8px;background:#182026;color:#e8eaed}"
    "button{width:100%;margin-top:22px;padding:14px;font-size:16px;font-weight:600;"
    "border:0;border-radius:8px;background:#4fc3f7;color:#08131a}"
    "button:active{background:#29b6f6}"
    ".n{margin-top:18px;font-size:12px;color:#546e7a;line-height:1.6}"
    "</style></head><body>"
    "<h1>Pulse 设备配网</h1>"
    "<p class=\"sub\">填写要连接的 WiFi（仅支持 2.4GHz）</p>"
    "<form method=\"POST\" action=\"/save\">"
    "<label>WiFi 名称 (SSID)</label>"
    "<input name=\"ssid\" required maxlength=\"63\" autocapitalize=\"off\" autocomplete=\"off\">"
    "<label>密码</label>"
    "<input name=\"pass\" type=\"password\" maxlength=\"63\" autocomplete=\"off\">"
    "<button type=\"submit\">保存并连接</button>"
    "</form>"
    "<p class=\"n\">ESP32-S3 不支持 5GHz 频段。<br>"
    "保存后设备会自动重连，本热点将关闭。</p>"
    "</body></html>";

static void handle_root(void) {
    g_server->send_P(200, "text/html; charset=utf-8", PROVISION_PAGE);
}

static void handle_save(void) {
    String ssid = g_server->arg("ssid");
    String pass = g_server->arg("pass");

    if (ssid.length() == 0) {
        g_server->send(400, "text/html; charset=utf-8",
                       "<meta charset=\"utf-8\"><p>SSID 不能为空</p>");
        return;
    }

    /* 先回响应再切模式：否则切到 STA 会立刻断开 AP，
     * 手机拿不到确认页面，用户不知道是否成功。 */
    g_server->send(200, "text/html; charset=utf-8",
                   "<!DOCTYPE html><meta charset=\"utf-8\">"
                   "<body style=\"font-family:system-ui;background:#101418;color:#e8eaed;"
                   "padding:24px\"><h2 style=\"color:#4fc3f7\">已保存</h2>"
                   "<p>设备正在连接，本热点即将关闭。</p>"
                   "<p style=\"color:#78909c;font-size:13px\">若指示灯或屏幕显示连接失败，"
                   "请重新进入配网并检查密码。</p></body>");

    log_i("%s: 收到配网凭据 SSID=\"%s\"", TAG, ssid.c_str());

    /* 持久化到 NVS，下次开机直接用 */
    config_save_wifi(ssid.c_str(), pass.c_str());

    /* 不在此直接 connect（见 g_provision_ready 注释）。记录凭据并置标志，
     * 由 tick 在 handleClient 返回后安全地关闭 AP 并连接目标 WiFi。 */
    strncpy(g_pending_ssid, ssid.c_str(), sizeof(g_pending_ssid) - 1);
    g_pending_ssid[sizeof(g_pending_ssid) - 1] = '\0';
    strncpy(g_pending_pass, pass.c_str(), sizeof(g_pending_pass) - 1);
    g_pending_pass[sizeof(g_pending_pass) - 1] = '\0';
    g_provision_ready = true;
}

/* 扫描结果 JSON，供页面可选地做下拉列表（当前页面未用，留作扩展）*/
static void handle_scan(void) {
    int n = WiFi.scanNetworks();
    String json = "[";
    for (int i = 0; i < n && i < 20; i++) {
        if (i) json += ",";
        json += "{\"ssid\":\"" + WiFi.SSID(i) + "\",\"rssi\":" + String(WiFi.RSSI(i)) + "}";
    }
    json += "]";
    g_server->send(200, "application/json", json);
    WiFi.scanDelete();
}

bool pulse_wifi_start_provisioning(void) {
    if (g_state == PULSE_WIFI_AP_PROVISION) return true;

    /* AP 名带 MAC 后缀，多台设备同时配网时不会混淆 */
    uint8_t mac[6] = {0};
    esp_read_mac(mac, ESP_MAC_WIFI_SOFTAP);
    snprintf(g_ap_ssid, sizeof(g_ap_ssid), "Pulse-%02X%02X", mac[4], mac[5]);

    WiFi.disconnect(true);
    WiFi.mode(WIFI_AP);
    /* 开放热点：设备无输入手段，让用户先连上再填凭据。
     * 这是配网期的临时热点，保存后立即关闭。 */
    if (!WiFi.softAP(g_ap_ssid)) {
        log_e("%s: 启动 AP 失败", TAG);
        return false;
    }

    if (!g_server) g_server = new WebServer(80);
    if (!g_server) {
        log_e("%s: WebServer 分配失败", TAG);
        return false;
    }

    g_server->on("/", HTTP_GET, handle_root);
    g_server->on("/save", HTTP_POST, handle_save);
    g_server->on("/scan", HTTP_GET, handle_scan);
    /* 任意路径都返回配网页，配合手机的 captive portal 探测 */
    g_server->onNotFound(handle_root);
    g_server->begin();

    g_state = PULSE_WIFI_AP_PROVISION;
    log_i("%s: 配网门户已启动", TAG);
    log_i("%s:   热点 \"%s\"（无密码）", TAG, g_ap_ssid);
    log_i("%s:   浏览器访问 http://%s", TAG, WiFi.softAPIP().toString().c_str());
    return true;
}

void pulse_wifi_stop_provisioning(void) {
    if (g_server) {
        g_server->stop();
        delete g_server;
        g_server = nullptr;
    }
    WiFi.softAPdisconnect(true);
    if (g_state == PULSE_WIFI_AP_PROVISION) g_state = PULSE_WIFI_IDLE;
    log_i("%s: 配网门户已关闭", TAG);
}

void pulse_wifi_shutdown(void) {
    if (g_server) pulse_wifi_stop_provisioning();
    WiFi.disconnect(true);
    WiFi.mode(WIFI_OFF);
    g_state = PULSE_WIFI_IDLE;
    log_i("%s: 射频已关闭", TAG);
}

int pulse_wifi_scan(void) {
    Serial.println();
    Serial.println("================ WiFi 扫描 ================");
    WiFi.mode(WIFI_STA);
    int n = WiFi.scanNetworks();
    if (n <= 0) {
        Serial.println("  未发现任何网络");
        Serial.println("=========================================");
        return 0;
    }

    Serial.printf("  发现 %d 个网络（ESP32-S3 仅支持 2.4GHz）:\n", n);
    Serial.printf("  %-32s %6s  %s\n", "SSID", "RSSI", "加密");
    for (int i = 0; i < n; i++) {
        const char *enc;
        switch (WiFi.encryptionType(i)) {
            case WIFI_AUTH_OPEN: enc = "开放"; break;
            case WIFI_AUTH_WEP: enc = "WEP"; break;
            case WIFI_AUTH_WPA_PSK: enc = "WPA"; break;
            case WIFI_AUTH_WPA2_PSK: enc = "WPA2"; break;
            case WIFI_AUTH_WPA_WPA2_PSK: enc = "WPA/2"; break;
            case WIFI_AUTH_WPA3_PSK: enc = "WPA3"; break;
            default: enc = "其他"; break;
        }
        Serial.printf("  %-32s %4d dBm  %s\n", WiFi.SSID(i).c_str(), WiFi.RSSI(i), enc);
    }
    WiFi.scanDelete();
    Serial.println("=========================================");
    return n;
}

void pulse_wifi_dump(void) {
    const pulse_config_t *c = config_get();
    pulse_wifi_status_t st;
    pulse_wifi_get_status(&st);

    Serial.println();
    Serial.println("================ WiFi 状态 ================");
    Serial.printf("  状态       %s\n", pulse_wifi_state_str(st.state));
    Serial.printf("  配置 SSID  %s\n", c->wifi.ssid[0] ? c->wifi.ssid : "(未设置)");

    if (st.connected) {
        Serial.printf("  已连接     %s\n", st.ssid);
        Serial.printf("  IP 地址    %s\n", st.ip);
        Serial.printf("  网关       %s\n", WiFi.gatewayIP().toString().c_str());
        Serial.printf("  信号       %d dBm %s\n", st.rssi,
                      st.rssi > -60   ? "(强)"
                      : st.rssi > -75 ? "(中)"
                                      : "(弱，上传可能超时)");
        Serial.printf("  已保持     %u 秒\n", st.connected_seconds);
    } else if (st.state == PULSE_WIFI_AP_PROVISION) {
        Serial.printf("  AP 热点    %s\n", st.ssid);
        Serial.printf("  配网地址   http://%s\n", st.ip);
    }

    Serial.printf("  连接尝试   %u 次\n", st.connect_attempts);
    Serial.printf("  断开次数   %u 次\n", st.disconnect_count);
    Serial.printf("  当前退避   %u ms（上限 %u）\n", st.backoff_ms, c->wifi.reconnect_max_ms);
    Serial.println("=========================================");
}
