# Pulse 硬件固件（ESP32-S3）

可穿戴语音记录设备的固件。采集对话音频 → 本地 TF 卡暂存 → 分段上传云端 AI 分析。

## 技术栈

- **MCU**：ESP32-S3R8（8MB octal PSRAM）+ W25Q128JVPIQ（16MB flash）
- **RTOS**：FreeRTOS（Arduino-ESP32 core 2.0.x / IDF 4.4.7）
- **UI**：LVGL 8.4
- **传输**：HTTP multipart（音频）+ SNTP（校时）
- **语言**：C++（Arduino framework）

## 架构

```
┌──────────────────────── ESP32-S3R8 ────────────────────────┐
│  Core 1（实时域）             Core 0（网络/UI 域）           │
│  ┌──────────────────┐       ┌──────────────────┐          │
│  │ audio_task  pr6  │       │ ui_task     pr2  │          │
│  │ I2S PDM RX + VAD │       │ LVGL + 触摸+电源 │          │
│  │ 段写入(spool)     │       └──────────────────┘          │
│  └──────────────────┘       ┌──────────────────┐          │
│  loop（core1，空闲）         │ net_task    pr4  │          │
│  时间/电量低频刷新            │ WiFi + 上传 + SNTP│          │
│                             └──────────────────┘          │
│  跨任务：FreeRTOS Queue + EventGroup + 互斥锁              │
│  PSRAM：音频 pre-roll、降级缓冲；内部 RAM：DMA、LVGL 缓冲   │
└────────────────────────────────────────────────────────────┘
         │ HTTP multipart  │ SNTP
         ▼                 ▼
   POST /api/v1/device/audio/upload
```

## 模块与测试（每个模块一个独立 env）

| env | 模块 | 独立测试内容 |
|---|---|---|
| `app` | **主固件**（全部串起来） | 完整联调 |
| `m0_config` | 板级定义 + 配置系统 | 打印配置 + 引脚自检 + 芯片资源 |
| `m1_pmu` | AXP2101 电源 | I2C 扫描 + 只读诊断（不改电源轨） |
| `m2_display` | ST7789 屏幕 | 彩条/边框/背光，实测 Y 偏移与颜色序 |
| `m3_touch` | CST816 触摸 + LVGL | 裸坐标方向 + LVGL 交互 + FPS |
| `m4_storage` | SD spool 队列 | 吞吐基准 + WAV 头 + 生命周期 + 淘汰 |
| `m5_audio` | I2S PDM + VAD | 通路自检 + 电平表 + 阈值标定 + 落盘 |
| `m6_rtc` | BM8563 + 时间戳 | 寄存器往返 + BCD 进位 + RFC3339 |
| `m7_wifi` | WiFi + 配网 | 扫描 + 状态机 + 退避 + AP 网页配网 |
| `m8_upload` | 上传层 | multipart 流式 + 断网补传 + 退避 |
| `m9_power` | 电源管理 | 三级功耗 + 深睡唤醒 + 低电告警 |
| `m10_ui` | LVGL 界面 | 模拟数据遍历全部场景 |

烧写某个模块：
```bash
pio run -e m5_audio -t upload -t monitor
```

## 配置（config.json，非硬编码）

所有可调参数集中在 `config.json`，复制到 TF 卡根目录即可被设备读取。
字段缺失时回退固件内置默认值。运行时可通过配网写入 NVS 覆盖 WiFi 凭据与 token。

层级覆盖顺序：**内置默认 → TF 卡 config.json → NVS（配网写入）**。

详见 `config.json` 内的逐项注释。

## 上位机工具（tools/）

| 脚本 | 用途 |
|---|---|
| `fake_server.py` | 模拟后端 `/device/audio/upload`，校验逻辑与真后端 audio.go 一致 |
| `selftest_fake_server.py` | 验证假后端本身的判定正确性 |
| `wav_check.py` | 音频质量分析（DC 偏移/削波/SNR），配合 M5 |
| `soak_test.py` | 断网长跑完整性分析（查重复/漏传/乱序） |
| `pin_lint.py` | 引脚与配置静态检查 |
| `gen_font.py` | 用 lv_font_conv 生成中文字体子集（界面中文用） |

## 与后端的契约（读后端代码得出）

- `POST /api/v1/device/audio/upload`，multipart，文件字段名 **`file`**
- 扩展名 `.wav/.mp3/.m4a`，且须与文件头魔数一致（`RIFF/WAVE`）
- 可选字段：`device_id`、`duration`（秒）、`recorded_at`（RFC3339）
- 采样参数被后端 `stt.go` 写死：**16kHz / 16bit / mono**
- 鉴权：后端已有设备级中间件 `middleware.DeviceAuth`，消费 `devices.device_token_hash`。
  固件请求头填 `Authorization: Device <device_token>`，改 `config.json` 即可，无需重编译
  （`auth_scheme` = `Device`，`auth_token` = 设备 token）。
- 设备态上传时后端以 token 反解出的设备为准，表单里的 `device_id` 会被忽略。
- **设备配对**（`src/net/claim`）：`POST /api/v1/device/claim` 已实现，
  `{device_id, bind_code, name}` → `{device, device_token}`。用户在 App 生成 6 位数字码
  （10 分钟、一次性），固件调用 `pulse_claim_request()` 换取 token，自动写入 NVS。
  详见 `src/net/claim/README.md`。**需集成到 UI 或配网流程**，当前仅提供 API。
- 其余设备接口（固件**尚未接入**）：
  - `POST /api/v1/device/heartbeat` — 响应捎带待执行指令与 `server_time`。
  - `POST /api/v1/device/commands/<id>/ack` — 回执 `done` / `failed`。
- 传输仍是明文 HTTP（`verify_tls: false`），按产品决策暂不上 TLS。

## 关键设计决策

1. **纯 HTTP 不引 MQTT**：后端零 MQTT 实现，音频走 upload 接口，
   控制面由心跳响应捎带（后端已打通；指令 pending-until-ack，
   `start/stop_recording` 幂等，10 分钟未取走即过期）。
2. **三级功耗**（Active/Idle/DeepSleep），默认不自动深睡，避免静音期"聋掉"。
3. **存储可降级**：SD 4-bit → 1-bit → PSRAM 环形缓冲，GPIO45/46 有问题时也能联调。
4. **流式上传**：960KB 的段绝不整块进内存，用自定义 Stream 边读边发。
5. **legacy I2S 驱动**：core 2.0.x 下 `i2s_pdm.h` 不存在，必须用 `driver/i2s.h`。

## 界面（LVGL 8.4，单屏穿戴式）

240×280 深色主题，单屏状态界面 + 弹层，全部用 LVGL 原生控件实现（无图标字体/图片依赖）。

```
┌──────────────────────────────┐
│ 14:30          ▂▄▆█  🔋 85% │  WiFi 信号格 + 电池图标
├──────────────────────────────┤
│ ╭──────────────────────────╮ │
│ │ ● 录音中               ● │ │  红点脉冲呼吸动画
│ │      01:13               │ │
│ ╰──────────────────────────╯ │
├──────────────────────────────┤
│ ╭──────────────────────────╮ │
│ │ 上传中…                ● │ │  状态彩色圆点
│ │ 队列18 已传42 失败1      │ │
│ ╰──────────────────────────╯ │
├──────────────────────────────┤
│ SD 卡 剩余 28.0 GB           │
│ ▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░   │  存储容量条
│                      [SET]  │  设置入口
└──────────────────────────────┘
```

| 元素 | 状态映射 |
|---|---|
| WiFi 信号格（4 竖条）| RSSI 分 4 档，断网全灰 |
| 电池图标 + 百分比 | 填充按电量，>20% 绿 / >10% 橙 / ≤10% 红 |
| 录音脉冲点 | 录音中红色呼吸动画，非录音灰 |
| 上传状态点 | 绿=已同步 蓝=上传中 橙=离线/退避 红=未配置 |
| 存储容量条 | 已用空间占比 |

弹层：

- **设置页**：亮度滑块（拖动预览、松开持久化）+ WiFi 重配 + Close
- **配网提示**：显示真实热点名与配网地址
- **toast**：低电/异常等一次性提示，3 秒自动消失

界面数据全部由 `pulse_ui_state_t` 结构体注入，UI 层不直接读硬件（M10 测试可纯模拟遍历所有状态）。

## 按键与功耗

### PWR 键（GPIO7，同时是 AXP2101_PWRON）

| 状态 | 短按 | 长按 |
|---|---|---|
| Active（亮屏录音） | 灭屏进 Idle，录音/传输继续 | AXP2101 硬件断电 |
| Idle（灭屏录音） | 亮屏回 Active | AXP2101 硬件断电 |
| DeepSleep | 硬件唤醒 | — |

- 短按是「息屏」而非「关机」，不打断录音和上传。
- 长按是 AXP2101 的硬件强制断电（固件死机时的兜底），软件不介入。

### 三级功耗

| 状态 | 屏幕 | 录音 | 进入条件 |
|---|---|---|---|
| Active | 亮 | 开 | 触摸 / PWR 短按唤醒 |
| Idle | 灭（背光=0） | **继续** | 无触摸超时 或 PWR 短按 |
| DeepSleep | 关 | 停 | 电量 <5% 或 `deep_sleep_ms` 静置超时 |

深睡不绑按键：深睡期间麦克风停止，且板上无「人声自动唤醒」通路，
故默认 `deep_sleep_ms=0`（永不自动深睡），避免白天静音期聋掉。

### 其他按键

- **BOOT**（GPIO0）：低电平进串口下载模式（配合 RST 使用）。
- **RST**：复位芯片。

## 已知风险（上板前必读）

- **GPIO46** 是 strapping 引脚，焊了外部上拉会导致无法下载（已在新版PCB中修复）
- **中文界面**默认用内置英文字体，中文需跑 `gen_font.py`。
