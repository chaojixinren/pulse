# Pulse 硬件

可穿戴语音采集设备完整硬件方案，包含 PCB、外壳与 ESP32-S3 固件。

## 整体架构

```
音频采集 → 本地 TF 卡暂存 → 分段上传云端 AI 分析
```

设备基于 ESP32-S3R8 主控，集成 PDM 数字麦克风、1.69" LCD 触摸屏、AXP2101 电源管理、BM8563 RTC 与 Micro SD 卡槽，支持 WiFi 无线联网与 OTA 升级。

## 功能列表

- 🎙️ **语音采集** — I2S PDM 数字麦克风，VAD 语音检测，Pre-roll 缓冲，WAV 输出
- 💾 **本地存储** — Micro SD 卡 spool 队列，自动降级（SD → PSRAM），容量淘汰与断网归档
- 📡 **网络上传** — WiFi 2.4 GHz + AP 配网，HTTP multipart 流式上传，断网补传，OTA 升级
- 🖥️ **用户界面** — 1.69" 触摸屏 + LVGL 8.4，单屏状态展示，背光调节，中文 CJK 字体
- 🔋 **电源管理** — AXP2101 PMU + BM8563 RTC，三级功耗（Active / Idle / DeepSleep），PWR 键息屏/断电
- 🔧 **按键交互** — PWR 息屏断电、BOOT 下载、RST 复位
- ⚙️ **配置系统** — 三层覆盖（固件默认 → TF 卡 JSON → NVS），所有参数无需重编译
- 🧪 **模块化测试** — 12 个独立 PlatformIO 环境（m0~m10），每模块可单独烧写验证

---

## 目录结构

```
hardware/
├── README.md               # 本文件
├── firmware/               # ESP32-S3 固件（PlatformIO 项目）
│   ├── src/
│   │   ├── main.cpp        # Arduino 入口（setup/loop 转发）
│   │   ├── app/            # 应用编排（任务启动、状态机）
│   │   ├── audio/          # I2S PDM 采集 + VAD 语音检测
│   │   ├── config/         # 配置管理（config.json 驱动）
│   │   ├── hal/            # 硬件抽象层
│   │   │   ├── display/    # ST7789 LCD 驱动
│   │   │   ├── pmu/        # AXP2101 电源管理
│   │   │   ├── rtc/        # BM8563 实时时钟
│   │   │   ├── storage/    # SD 卡 / PSRAM 环形缓冲
│   │   │   └── touch/      # CST816 电容触摸
│   │   ├── net/            # 网络通信
│   │   │   ├── wifi/       # WiFi 连接 + AP 配网
│   │   │   └── upload/     # HTTP multipart 上传
│   │   ├── power/          # 三级功耗管理（Active / Idle / DeepSleep）
│   │   ├── ui/             # LVGL 8.4 界面
│   │   │   ├── fonts/      # CJK 字体子集（12/14/16/20/28 px）
│   │   │   ├── lv_port/    # LVGL 平台适配层
│   │   │   └── screens/    # 主界面 + 弹层
│   │   └── tests/          # 模块级独立测试（m0 ~ m10）
│   ├── include/
│   │   ├── board_pins.h    # 板级引脚定义（编译期自检）
│   │   └── lv_conf.h       # LVGL 8.4 配置
│   ├── config.json         # 运行时配置（复制到 TF 卡即可生效）
│   ├── platformio.ini      # PlatformIO 多环境配置
│   ├── partitions_16mb.csv # 16MB Flash 分区表（OTA / SPIFFS / coredump）
│   └── pc_sim/             # PC 模拟运行（SDL2，无需硬件）
├── pcb/                    # PCB 设计（嘉立创 EDA）
│   ├── Pulse.epro2         # 嘉立创 EDA 集成项目
│   ├── pcb/Pulse.pcbdoc    # PCB 布局文件
│   ├── schematic/Pulse.schdoc # 原理图文件
│   ├── gerber/             # Gerber 输出（打样用）
│   ├── docs/
│   │   ├── BOM.csv         # 物料清单
│   │   ├── PickAndPlace.csv # SMT 坐标文件
│   │   └── Schematic.pdf   # 原理图 PDF
│   ├── models/Pulse.step   # PCB 3D 模型
│   └── README.md           # PCB 详细说明
└── enclosure/              # 外壳 3D 设计（Fusion 360）
    ├── Pulse.f3d           # 主体外壳源文件
    ├── Pulse_button.f3d    # 按钮部件源文件
    ├── step/
    │   ├── Pulse.step      # 主体外壳 CAD 工程文件
    │   └── Pulse_button.step # 按钮部件 CAD 工程文件
    ├── stl/
    │   ├── Pulse.stl       # 主体外壳（3D 打印）
    │   └── Pulse_button.stl # 按钮部件（3D 打印）
    └── README.md           # 外壳详细说明
```

---

## 固件（firmware/）

固件位于 [firmware/](firmware/)，使用 **PlatformIO + Arduino-ESP32 core** 构建。

### 技术栈

| 项目 | 规格 |
|------|------|
| MCU | ESP32-S3R8（Xtensa LX7 双核，8MB octal PSRAM） |
| 外部 Flash | W25Q128JVPIQ（16MB SPI） |
| RTOS | FreeRTOS（Arduino-ESP32 core 2.0.x / IDF 4.4.7） |
| UI | LVGL 8.4 |
| 显示 | ST7789（240×280，SPI） + CST816 电容触摸 |
| 音频 | MSM261D4030H1CPM（PDM 数字麦克风，I2S） |
| 电源 | AXP2101 PMU + BM8563 RTC |
| 存储 | Micro SD 卡（SDMMC 4-bit）+ PSRAM 降级缓冲 |
| 传输 | HTTP multipart 上传 + SNTP 校时 |
| 语言 | C++（Arduino framework） |

### 任务架构

```
Core 1（实时域）                        Core 0（网络/UI 域）
┌──────────────────┐                  ┌──────────────────┐
│ audio_task  pr6  │                  │ ui_task     pr2  │
│ I2S PDM RX + VAD │                  │ LVGL + 触摸+电源 │
│ 段写入(spool)     │                  └──────────────────┘
└──────────────────┘                  ┌──────────────────┐
loop（core1，空闲）                     │ net_task    pr4  │
时间/电量低频刷新                       │ WiFi + 上传 + SNTP│
                                       └──────────────────┘
跨任务通信：FreeRTOS Queue + EventGroup + 互斥锁
PSRAM：音频 pre-roll、降级缓冲；内部 RAM：DMA、LVGL 缓冲
```

### 模块化测试

每个功能模块有独立的 PlatformIO 环境，可单独烧写验证：

| 环境 | 模块 | 测试内容 |
|------|------|----------|
| `app` | 主固件（全链路联调） | 完整集成测试 |
| `m0_config` | 板级定义 + 配置系统 | 打印配置、引脚自检、芯片资源 |
| `m1_pmu` | AXP2101 电源 | I2C 扫描 + 只读诊断 |
| `m2_display` | ST7789 屏幕 | 彩条/边框/背光，Y 偏移与颜色序 |
| `m3_touch` | CST816 触摸 + LVGL | 裸坐标方向 + LVGL 交互 + FPS |
| `m4_storage` | SD spool 队列 | 吞吐基准 + WAV 头 + 生命周期 + 淘汰 |
| `m5_audio` | I2S PDM + VAD | 通路自检 + 电平表 + 阈值标定 + 落盘 |
| `m6_rtc` | BM8563 + 时间戳 | 寄存器往返 + BCD 进位 + RFC3339 |
| `m7_wifi` | WiFi + 配网 | 扫描 + 状态机 + 退避 + AP 网页配网 |
| `m8_upload` | 上传层 | multipart 流式 + 断网补传 + 退避 |
| `m9_power` | 电源管理 | 三级功耗 + 深睡唤醒 + 低电告警 |
| `m10_ui` | LVGL 界面 | 模拟数据遍历全部场景 |

烧写示例：
```bash
pio run -e m5_audio -t upload -t monitor
```

### 配置方式

所有可调参数集中在 `config.json`，复制到 TF 卡根目录即可被设备读取。字段缺失时回退固件内置默认值。运行时可通过配网写入 NVS 覆盖 WiFi 凭据与 token。

覆盖优先级：**内置默认 → TF 卡 config.json → NVS（配网写入）**

详见 [config.json](firmware/config.json) 内的逐项注释。

### 按键与功耗

| 按键 | 短按 | 长按 |
|------|------|------|
| PWR（GPIO7） | Active ↔ Idle 息屏切换，不中断录音 | AXP2101 硬件断电 |
| BOOT（GPIO0） | 低电平进串口下载模式 | — |
| RST | 复位芯片 | — |

三级功耗：Active（亮屏录音）→ Idle（灭屏背光，录音继续）→ DeepSleep（停录）。默认不自动深睡（`deep_sleep_ms=0`），避免静音期聋掉。

### 分区表

16MB Flash 分区（W25Q128JVPIQ）：NVS / OTA（app0+app1 各 6MB）/ SPIFFS（LVGL 资源）/ coredump。

详见 [partitions_16mb.csv](firmware/partitions_16mb.csv)。

### 上位机工具

| 脚本 | 用途 |
|------|------|
| `fake_server.py` | 模拟后端 `/audio/upload` |
| `selftest_fake_server.py` | 验证假后端判定正确性 |
| `wav_check.py` | 音频质量分析（DC 偏移/削波/SNR） |
| `soak_test.py` | 断网长跑完整性分析 |
| `pin_lint.py` | 引脚与配置静态检查 |
| `gen_font.py` | 生成中文字体子集 |

### 快速开始

```bash
cd hardware/firmware

# 1. 安装依赖
pio lib install

# 2. 编译主固件
pio run -e app

# 3. 上传到设备
pio run -e app -t upload

# 4. 串口监视
pio device monitor
```

#### 配置 WiFi

设备首次启动时自动进入 AP 配网模式：

1. 手机连接到 `Pulse-XXXX` WiFi 热点
2. 浏览器访问 `http://192.168.4.1`
3. 输入家庭/办公室 WiFi 账号密码
4. 保存后设备自动重启并连接

WiFi 凭据通过 NVS 持久化，后续启动自动重连。也可在 TF 卡的 `config.json` 中配置。

#### 配置云端 API

将 TF 卡格式化为 FAT32，将 `firmware\config.json` 复制到 TF 卡根目录：

```json
{
  "cloud": {
    "base_url": "http://your-server:8080",
    "upload_path": "/api/v1/audio/upload",
    "device_id": "",
    "auth_scheme": "Bearer",
    "auth_token": "your_jwt_token",
    "request_timeout_ms": 30000,
    "max_retry": 5,
    "retry_backoff_ms": 2000
  }
}
```

设备启动时按优先级读取配置：**固件内置默认 → TF 卡 config.json → NVS（配网写入）**。`auth_scheme` 与 `auth_token` 组成请求头 `Authorization: <scheme> <token>`，后端补齐设备级鉴权后只需修改 `config.json`，无需重编译。

---

## PCB（pcb/）

PCB 设计文件位于 [pcb/](pcb/)，使用 **嘉立创 EDA** 设计。

### 设计规格

| 参数 | 规格 |
|------|------|
| 层数 | 4 层（信号 - GND - 电源 - 信号） |
| 板厚 | 1.0 mm |
| 铜厚 | 1 oz（35 μm） |
| 最小线宽/线距 | 5 mil / 3 mil |
| 最小过孔 | 0.3 mm（机械孔） |
| 表面处理 | ENIG（沉金） |
| 阻焊颜色 | 黑色（Top / Bottom） |
| 丝印颜色 | 白色 |
| 板材 | FR-4（Tg ≥ 150°C） |

### 板卡尺寸

PCB 外形约为 **35 mm × 31 mm**，四角有 M2 螺丝安装孔。

### 核心元件

| 元件 | 型号 | 功能 |
|------|------|------|
| U1 | ESP32-S3R8 | 主控，8MB Flash / 8MB PSRAM |
| U2 | W25Q128JVPIQ | 16MB 外部 SPI Flash |
| U3 | ANT3216LL00R2400A | 2.4 GHz 陶瓷天线 |
| U4 | AXP2101 | 电源管理 IC（PMU） |
| U5 | BM8563EMA | 实时时钟（RTC） |
| MIC1 | MSM261D4030H1CPM | MEMS 数字麦克风（PDM） |
| FPC1 | HC-FPPC-05-09-18RLTAG | 18-pin FPC 连接器（LCD） |
| USB1 | MC-121-L119 | USB-C 连接器 |
| KEY_DWN/PWR/RST | TS24CA | 录音 / 电源 / 复位按键 |
| SIM1 | SDCARD-1 (Molex) | Micro SD 卡座（SMD） |
| D1 | USBLC6-2SC6 | USB ESD 保护 |

详见 [pcb/README.md](pcb/README.md) 和 [pcb/docs/BOM.csv](pcb/docs/BOM.csv)。

---

## 外壳（enclosure/）

外壳设计文件位于 [enclosure/](enclosure/)，使用 **Fusion 360** 设计。

### 文件清单

| 文件 | 说明 |
|------|------|
| `Pulse.f3d` | 主体外壳 Fusion 360 源文件 |
| `Pulse_button.f3d` | 按钮部件 Fusion 360 源文件 |
| `step/Pulse.step` | 主体外壳 CAD 工程文件 |
| `step/Pulse_button.step` | 按钮部件 CAD 工程文件 |
| `stl/Pulse.stl` | 主体外壳（3D 打印） |
| `stl/Pulse_button.stl` | 按钮部件（3D 打印） |

### 组装说明

1. 将按钮部件安装到主体外壳对应位置
2. 将 PCB 放入主体外壳对应的卡槽
3. 盖上屏幕，确认所有接口（USB-C、麦克风孔）露出
4. 必要时使用胶水或螺丝固定

### 3D 打印参数

- **材料**：PLA / PETG
- **层高**：0.2 mm
- **填充**：20%
- **支撑**：需要

### 注意事项

- 外壳尺寸与 PCB（35 mm × 31 mm）匹配，如有差异请调整 Fusion 360 源文件
- 麦克风开孔需保证收音效果，避免被外壳遮挡
- 按钮行程空间需预留充足，避免卡死
- USB-C 充电接口开口需对准设备接口位置

详见 [enclosure/README.md](enclosure/README.md)。

---

## 引脚总览

详见 [firmware/include/board_pins.h](firmware/include/board_pins.h)。

| 外设 | 引脚 | 说明 |
|------|------|------|
| BOOT 键 | GPIO0 | 同时是 strapping 引脚 |
| 麦克风 Data | GPIO1 | I2S0 PDM RX |
| 麦克风 CLK | GPIO2 | I2S0 PDM 时钟 |
| I2C SCL | GPIO4 | AXP2101 + BM8563 + CST816 共用 |
| I2C SDA | GPIO5 | 同上 |
| PMU IRQ | GPIO6 | AXP2101 中断，低电平有效 |
| PWR 键 | GPIO7 | 同时是 AXP2101_PWRON |
| LCD 背光 | GPIO8 | 经 AO3400A，LEDC PWM 调光 |
| LCD DC | GPIO9 | |
| LCD CS | GPIO10 | |
| LCD SCLK | GPIO11 | SPI2（FSPI） |
| LCD MOSI | GPIO12 | |
| LCD / TP RST | GPIO13 | 共用，复位 LCD 和触摸屏 |
| TP INT | GPIO14 | 低电平有效 |
| USB DM | GPIO19 | |
| USB DP | GPIO20 | |
| SD D0 | GPIO42 | SDMMC 4-bit |
| SD CLK | GPIO41 | |
| SD CMD | GPIO40 | |
| SD D3 | GPIO39 | |
| SD D2 | GPIO38 | |
| SD D1 | GPIO45 | |

> **注意**：GPIO26~37 为 ESP32-S3 保留引脚（接封装内 Flash / octal PSRAM），不可他用。`board_pins.h` 包含编译期静态自检，引脚冲突会在编译阶段报错。

---

## 硬件需求总览

| 组件 | 规格 |
|------|------|
| 麦克风 | MSM261D4030H1CPM，PDM 数字输出，全向拾音 |
| 显示屏 | 1.69" ST7789，240×280，SPI + CST816 电容触摸 |
| 处理器 | ESP32-S3R8，Xtensa LX7 双核，8MB PSRAM |
| 电源 | AXP2101 PMU，USB-C 充电，锂电池供电 |
| 存储 | 16MB 外部 Flash + Micro SD 卡 + 8MB PSRAM |
| 网络 | WiFi 2.4 GHz（802.11b/g/n）+ BLE 5 |
| 实时时钟 | BM8563EMA，I2C 接口，电池备份 |

---

## 打样与制造

### PCB 打样

Gerber 文件位于 [pcb/gerber/](pcb/gerber/)，可直接上传嘉立创 / 捷配等平台打样。

### 外壳打印

STL 文件位于 [enclosure/stl/](enclosure/stl/)，支持 FDM 3D 打印。

### SMT 贴片

参考 [pcb/docs/PickAndPlace.csv](pcb/docs/PickAndPlace.csv) 和 [pcb/docs/BOM.csv](pcb/docs/BOM.csv) 进行贴片。

---

## 许可证

MIT
