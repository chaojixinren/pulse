# 设备配对模块 (pulse_claim)

## 功能

让固件自动从后端换取 `device_token`，无需手动复制粘贴到 `config.json`。

## 流程

1. 用户在 App 上点"添加设备"，后端生成一个 **6 位数字绑定码**（10 分钟有效，一次性）
2. 设备端输入这个 6 位码（可以通过配网 Web 表单、触摸键盘、或预留的物理按键）
3. 固件调用 `pulse_claim_request()`，POST 到 `/api/v1/device/claim`
4. 后端返回 `device_token`，固件自动写入 NVS（覆盖 `config.json`）
5. 重启或调用 `config_reload()`，上传层开始用新 token

## API

### `pulse_claim_request()`

```c
claim_request_t req = {};
strlcpy(req.bind_code, "123456", sizeof(req.bind_code));
// device_id 和 device_name 留空会自动填默认值

claim_result_t result = pulse_claim_request(&req);
switch (result) {
case CLAIM_OK:
    // 成功，req.device_token 已填充，NVS 已写入
    ESP.restart(); // 让新 token 生效
    break;
case CLAIM_INVALID_CODE:
    // 绑定码无效/过期/已使用
    break;
case CLAIM_NETWORK_ERROR:
    // 网络错误，可重试
    break;
case CLAIM_JSON_ERROR:
case CLAIM_NVS_ERROR:
    // 后端响应格式错误 / NVS 写入失败
    break;
}
```

### `pulse_claim_is_needed()`

```c
if (pulse_claim_is_needed()) {
    // device_token 为空，或 auth_scheme 还是 Bearer（临时调试用）
    // 需要配对
}
```

## 集成建议

### 方案 A: 启动时检查

```cpp
// 在 net_task 的 WiFi 连接成功后
if (pulse_claim_is_needed()) {
    // 进入配对 UI，提示用户在 App 生成绑定码
    show_pairing_screen();
}
```

### 方案 B: 配网时一并输入

在 `pulse_wifi_start_provisioning()` 的 Web 表单里加一个"绑定码"输入框，和 SSID/密码一起提交。WiFi 连接成功后立即调用 `pulse_claim_request()`。

优点：用户只需配网一次，不用多一步操作。

### 方案 C: 设置页手动触发

在设置页加一个"绑定设备"按钮，点击后弹出 6 位数字键盘，输入后调用 `pulse_claim_request()`。

适合已经在用但需要重新绑定（设备被解绑、恢复出厂）的场景。

## 文件

- `pulse_claim.h` — 头文件
- `pulse_claim.cpp` — 实现
- `claim_example.cpp` — 集成示例（不编译进主固件）

## 依赖

- `ArduinoJson` — 解析后端响应
- `HTTPClient` — 发起 POST 请求
- `config/pulse_config.h` — 读 `base_url` / 写 NVS
- `net/wifi/pulse_wifi.h` — 检查 WiFi 状态（仅头文件依赖，不调用函数）

## 后端接口

`POST /api/v1/device/claim`

请求:
```json
{
  "device_id": "pulse-aabbccddeeff",
  "bind_code": "123456",
  "name": "我的 Pulse"
}
```

响应 (200):
```json
{
  "device": {
    "id": "uuid",
    "device_id": "pulse-aabbccddeeff",
    "name": "我的 Pulse",
    ...
  },
  "device_token": "abcd1234..."
}
```

错误 (400/401):
```json
{
  "error": "绑定码无效或已过期"
}
```

## 测试

1. 后端生成绑定码:
   ```bash
   curl -X POST http://192.168.1.100:8080/api/v1/devices/bind-code \
     -H "Authorization: Bearer <user_jwt>" \
     -H "Content-Type: application/json"
   ```
   响应中拿到 `code`（6 位数字）。

2. 固件调用:
   ```cpp
   claim_request_t req = {};
   strlcpy(req.bind_code, "123456", sizeof(req.bind_code));
   claim_result_t result = pulse_claim_request(&req);
   ```

3. 检查 NVS:
   ```cpp
   config_dump(); // 应该看到 auth_scheme=Device, auth_token=<长串>
   ```

4. 重启后上传音频，应该不再 401。

## 注意事项

- **绑定码无限流**（按产品决策）：6 位数字 + 10 分钟窗口，理论上可暴力穷举。后端已预留限流入口，固件侧不受影响。
- **传输明文**：`verify_tls: false`，token 在链路上裸传。按产品决策暂不上 TLS。
- **device_id 生成规则**：空则用 `pulse-<MAC>`（12 位小写十六进制）。如果要自定义，在 `config.json` 里填 `cloud.device_id`。
- **重复绑定**：同一用户重复绑定会轮换 token（旧 token 失效），适合恢复出厂后重新配对。设备已属于别人则拒绝绑定。
