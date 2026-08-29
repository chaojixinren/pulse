/**
 * @file claim_example.cpp
 * @brief 设备配对集成示例
 *
 * 这个文件演示如何在固件中集成 pulse_claim。
 * 不直接编译进主固件，供开发者参考。
 */
#include "pulse_claim.h"

#include <Arduino.h>

#include "config/pulse_config.h"
#include "net/wifi/pulse_wifi.h"

/**
 * 示例 1: 启动时检查是否需要配对
 *
 * 在 pulse_app_bootstrap() 或 net_task 里调用，
 * WiFi 连接成功后执行。
 */
void example_check_claim_on_boot() {
    if (!pulse_claim_is_needed()) {
        Serial.println("[claim] device_token 已存在，跳过配对");
        return;
    }

    // 需要配对，但此时还没有绑定码
    // 方案 1: 进入配对 UI，提示用户在 App 生成绑定码
    // 方案 2: 先跳过，让设备正常启动，用户稍后在设置页手动触发
    Serial.println("[claim] 需要配对，等待用户输入绑定码");

    // 假设 UI 提示用户，用户在 App 生成了 "123456"，
    // 然后在设备上输入（物理键盘、触摸键盘、或通过配网 Web 表单）
}

/**
 * 示例 2: UI 触发的配对流程
 *
 * 用户在设置页点"绑定设备"，进入绑定码输入界面。
 * 输入 6 位数字后调用此函数。
 */
bool example_bind_with_code(const char *bind_code_input) {
    if (!pulse_wifi_is_connected()) {
        Serial.println("[claim] WiFi 未连接，无法配对");
        // UI 提示: "请先连接 WiFi"
        return false;
    }

    claim_request_t req = {};
    // device_id: 空则自动从 config 或 MAC 生成
    // device_name: 空则用默认 "我的 Pulse"
    // 也可以让用户自己输入名字:
    // strlcpy(req.device_name, user_input_name, sizeof(req.device_name));
    strlcpy(req.bind_code, bind_code_input, sizeof(req.bind_code));

    Serial.printf("[claim] 发起配对请求: bind_code=%s\n", req.bind_code);

    claim_result_t result = pulse_claim_request(&req);

    switch (result) {
    case CLAIM_OK:
        Serial.println("[claim] 配对成功！");
        Serial.printf("[claim] device_token: %s\n", req.device_token);
        // UI 提示: "绑定成功，即将重启"
        // 重启让新 token 生效（或调用 config_reload() 如果有的话）
        delay(2000);
        ESP.restart();
        return true;

    case CLAIM_INVALID_CODE:
        Serial.println("[claim] 绑定码无效或已过期");
        // UI 提示: "绑定码错误，请重新输入"
        return false;

    case CLAIM_NETWORK_ERROR:
        Serial.println("[claim] 网络错误，请稍后重试");
        // UI 提示: "网络错误"
        return false;

    case CLAIM_JSON_ERROR:
        Serial.println("[claim] 后端响应格式错误");
        // UI 提示: "服务器错误"
        return false;

    case CLAIM_NVS_ERROR:
        Serial.println("[claim] NVS 写入失败（拿到 token 了但存不进去）");
        // UI 提示: "存储错误，请重启后重试"
        return false;

    default:
        Serial.printf("[claim] 未知错误: %d\n", result);
        return false;
    }
}

/**
 * 示例 3: 在配网 Web 表单里直接输入绑定码
 *
 * pulse_wifi_start_provisioning() 开启的 AP 门户页面，
 * 可以加一个"绑定码"输入框，和 SSID/密码一起提交。
 *
 * 流程：用户填 SSID、密码、绑定码 → 保存 WiFi 凭据到 NVS → 连接 WiFi →
 * WiFi 连接成功后立即调用 pulse_claim_request()。
 *
 * 这样用户只需要在配网时一并输入绑定码，不用多一步操作。
 */
void example_provision_with_claim(const char *ssid, const char *password, const char *bind_code) {
    // 1. 保存 WiFi 凭据
    if (!config_save_wifi(ssid, password)) {
        Serial.println("[provision] WiFi 凭据保存失败");
        return;
    }

    // 2. 连接 WiFi
    pulse_wifi_stop_provisioning();
    if (!pulse_wifi_connect(ssid, password)) {
        Serial.println("[provision] WiFi 连接失败");
        return;
    }

    // 3. 等待连接成功（轮询或在 net_task 里）
    for (int i = 0; i < 30; i++) {
        if (pulse_wifi_is_connected()) break;
        delay(1000);
    }

    if (!pulse_wifi_is_connected()) {
        Serial.println("[provision] WiFi 连接超时");
        return;
    }

    // 4. 如果用户填了绑定码，立即配对
    if (bind_code && bind_code[0]) {
        example_bind_with_code(bind_code);
    }
}

/**
 * 示例 4: 401 上传失败后重新配对
 *
 * 上传层返回 HTTP 401 时（token 过期/被撤销/设备被解绑），
 * 可以触发重新配对流程。
 */
void example_handle_upload_401() {
    Serial.println("[claim] 检测到 401，device_token 可能失效");
    // UI 提示: "设备已解绑，请重新绑定"
    // 跳转到绑定码输入界面
}
