/**
 * @file pulse_claim.cpp
 * @brief 设备配对实现
 */
#include "pulse_claim.h"

#include <Arduino.h>
#include <ArduinoJson.h>
#include <HTTPClient.h>
#include <WiFi.h>
#include <WiFiClient.h>
#include <WiFiClientSecure.h>

#include "config/pulse_config.h"

static const char *TAG = "claim";

claim_result_t pulse_claim_request(claim_request_t *req) {
    if (!req || !req->bind_code[0]) {
        log_e("%s: 绑定码为空", TAG);
        return CLAIM_INVALID_CODE;
    }

    pulse_config_t *cfg = config_get();
    if (!cfg->cloud.base_url[0]) {
        log_e("%s: base_url 未配置", TAG);
        return CLAIM_NETWORK_ERROR;
    }

    // device_id: 优先用请求里的，空则从 config 取，还是空就用 MAC
    if (!req->device_id[0]) {
        if (cfg->cloud.device_id[0]) {
            strlcpy(req->device_id, cfg->cloud.device_id, sizeof(req->device_id));
        } else {
            uint8_t mac[6];
            WiFi.macAddress(mac);
            snprintf(req->device_id, sizeof(req->device_id),
                     "pulse-%02x%02x%02x%02x%02x%02x",
                     mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
        }
    }

    // device_name: 空则用默认
    if (!req->device_name[0]) {
        strlcpy(req->device_name, "我的 Pulse", sizeof(req->device_name));
    }

    char url[256];
    snprintf(url, sizeof(url), "%s/api/v1/device/claim", cfg->cloud.base_url);

    // 构造 JSON: {"device_id": "...", "bind_code": "123456", "name": "..."}
    DynamicJsonDocument doc(512);
    doc["device_id"] = req->device_id;
    doc["bind_code"] = req->bind_code;
    doc["name"] = req->device_name;

    String payload;
    serializeJson(doc, payload);

    log_i("%s: POST %s", TAG, url);
    log_i("%s: device_id=%s, bind_code=%s", TAG, req->device_id, req->bind_code);

    HTTPClient http;
    WiFiClient *client = nullptr;
    WiFiClientSecure *sclient = nullptr;

    // 根据 verify_tls 决定用 WiFiClient 还是 WiFiClientSecure
    if (cfg->cloud.verify_tls) {
        sclient = new WiFiClientSecure();
        if (!sclient) {
            log_e("%s: WiFiClientSecure 分配失败", TAG);
            return CLAIM_NETWORK_ERROR;
        }
        sclient->setInsecure(); // 不校验证书，按产品决策
        client = sclient;
    } else {
        client = new WiFiClient();
        if (!client) {
            log_e("%s: WiFiClient 分配失败", TAG);
            return CLAIM_NETWORK_ERROR;
        }
    }

    if (!http.begin(*client, url)) {
        log_e("%s: http.begin 失败", TAG);
        delete client;
        return CLAIM_NETWORK_ERROR;
    }

    http.addHeader("Content-Type", "application/json");
    http.setTimeout(cfg->cloud.request_timeout_ms);

    int code = http.POST(payload);

    claim_result_t result;

    if (code <= 0) {
        log_e("%s: HTTP 错误: %s", TAG, HTTPClient::errorToString(code).c_str());
        result = CLAIM_NETWORK_ERROR;
        goto cleanup;
    }

    log_i("%s: HTTP %d", TAG, code);

    if (code == 401 || code == 400) {
        String body = http.getString();
        log_w("%s: 绑定码无效: %s", TAG, body.c_str());
        result = CLAIM_INVALID_CODE;
        goto cleanup;
    }

    if (code != 200 && code != 201) {
        log_e("%s: 后端返回 %d", TAG, code);
        result = CLAIM_NETWORK_ERROR;
        goto cleanup;
    }

    // 解析响应: {"device": {...}, "device_token": "..."}
    String resp = http.getString();
    http.end();
    delete client;

    DynamicJsonDocument respDoc(2048);
    DeserializationError err = deserializeJson(respDoc, resp);
    if (err) {
        log_e("%s: JSON 解析失败: %s", TAG, err.c_str());
        return CLAIM_JSON_ERROR;
    }

    const char *token = respDoc["device_token"];
    if (!token || !token[0]) {
        log_e("%s: device_token 字段缺失", TAG);
        return CLAIM_JSON_ERROR;
    }

    strlcpy(req->device_token, token, sizeof(req->device_token));
    log_i("%s: 收到 device_token (长度=%d)", TAG, strlen(token));

    // 写入 NVS，覆盖 config.json
    if (!config_save_token("Device", token)) {
        log_e("%s: NVS 写入失败", TAG);
        return CLAIM_NVS_ERROR;
    }

    log_i("%s: 配对成功，device_token 已写入 NVS", TAG);
    return CLAIM_OK;

cleanup:
    http.end();
    delete client;
    return result;
}

bool pulse_claim_is_needed(void) {
    pulse_config_t *cfg = config_get();
    // token 为空，或 scheme 是 Bearer（说明还在用临时 JWT 调试）
    if (!cfg->cloud.auth_token[0]) return true;
    if (strcmp(cfg->cloud.auth_scheme, "Device") != 0) return true;
    return false;
}
