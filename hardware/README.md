# Pulse Hardware

基于可穿戴设备的语音采集硬件。

## 技术栈

- **单片机**: ESP32-S3 系列芯片（Xtensa® 32 位 LX7 双核微处理器）
- **RTOS**: FreeRTOS / RT-Thread
- **界面框架**: LVGL
- **开发环境**: VSCode + PlatformIO
- **编程语言**: C / Arduino

### 芯片规格（ESP32-S3）

- **处理器**: Xtensa® 32 位 LX7 双核微处理器
- **无线连接**: 2.4 GHz Wi-Fi (IEEE 802.11b/g/n) 和 Bluetooth® 5 (LE)
- **存储扩展**: 芯片封装内可叠封 1.8 V 或 3.3 V flash 和 PSRAM
- **GPIO**: 45 个 GPIO 管脚
- **封装**: QFN56 (7×7 mm) 封装

## 规划中的功能

- 🎤 语音采集
- 🖥️ LVGL 图形界面显示
- 📡 网络连接（WiFi/蓝牙/4G）
- 🔋 电源管理
- 💾 本地缓存
- 🔒 数据加密

## 目录结构

```
hardware/
├── firmware/          # 固件代码
│   ├── platformio.ini # PlatformIO 配置
│   ├── src/           # 源代码
│   └── include/       # 头文件
├── pcb/              # PCB 设计文件（待定）
├── enclosure/        # 外壳设计（待定）
└── docs/             # 硬件文档
```

## 开发环境搭建

### 1. 安装 VSCode

下载并安装 [Visual Studio Code](https://code.visualstudio.com/)

### 2. 安装 PlatformIO 插件

在 VSCode 中安装 PlatformIO IDE 扩展

### 3. 创建项目

```bash
cd hardware/firmware
# PlatformIO 会自动识别 platformio.ini
```

### 4. 编译和上传

```bash
# 编译
pio run

# 上传到设备
pio run --target upload

# 串口监视
pio device monitor
```

## 技术要点

### RTOS 任务管理
- 音频采集任务
- 网络通信任务
- UI 更新任务
- 电源管理任务

### LVGL 界面
- 状态显示（录音中/待机/充电）
- 设置界面（WiFi 配置等）
- 电量显示
- 简洁的交互设计

### 硬件需求

- 麦克风：高质量音频采集
- 显示屏：支持 LVGL（LCD/OLED）
- 处理器：能处理音频、RTOS、LVGL 和网络通信
- 电源：续航 > 8 小时
- 存储：临时缓存音频数据
- 网络：能够上传数据到云端

---

**当前状态**：技术栈已确定（ESP32-S3），可开始固件开发与原型验证

## 项目结构

```
hardware/
├── firmware/                 # 固件代码
│   ├── src/
│   │   ├── main.cpp         # 主程序
│   │   ├── audio.cpp        # 音频采集
│   │   ├── wifi.cpp         # WiFi 连接
│   │   ├── upload.cpp       # 云端上传
│   │   └── config.cpp       # 配置管理
│   ├── include/
│   └── platformio.ini       # PlatformIO 配置
├── pcb/                     # PCB 设计文件 (KiCad)
│   ├── schematic.pdf        # 原理图
│   └── gerber/              # Gerber 文件
├── enclosure/               # 外壳设计 (3D 打印)
│   ├── case.stl
│   └── case.step
├── docs/
│   ├── assembly.md          # 组装指南
│   └── bom.xlsx             # 物料清单
└── README.md               # 本文件
```

## 快速开始

### 开发环境

推荐使用 **PlatformIO** (支持 VSCode 插件)：

```bash
# 安装 PlatformIO CLI
pip install platformio

# 或使用 VSCode 安装 PlatformIO IDE 扩展
```

也可以使用 **Arduino IDE** (需要额外配置 ESP32 支持)。

### 硬件准备

**方案 A：使用开发板（快速原型）**
- ESP32-S3-DevKitC-1 开发板
- INMP441 麦克风模块
- 面包板 + 杜邦线

**方案 B：定制 PCB（量产）**
- 焊接完整的 PCB 板
- 参考 `pcb/` 目录中的设计文件

### 接线方式

| INMP441 引脚 | ESP32-S3 引脚 |
|-------------|--------------|
| VDD         | 3.3V         |
| GND         | GND          |
| SD          | GPIO 4       |
| WS          | GPIO 5       |
| SCK         | GPIO 6       |
| L/R         | GND (左声道) |

| 其他组件     | ESP32-S3 引脚 |
|-------------|--------------|
| RGB LED     | GPIO 48      |
| 录音按钮     | GPIO 0       |

### 编译和上传

```bash
cd firmware

# 安装依赖
pio lib install

# 编译固件
pio run

# 上传到设备
pio run --target upload

# 查看串口输出
pio device monitor
```

### 配置 WiFi

首次启动时，设备会进入 AP 模式：

1. 手机连接到 `Pulse-XXXX` WiFi 热点
2. 浏览器访问 `http://192.168.4.1`
3. 输入家庭/办公室 WiFi 账号密码
4. 保存后设备自动重启并连接

或者通过串口配置：

```cpp
// 修改 firmware/src/config.cpp
const char* WIFI_SSID = "YourWiFiName";
const char* WIFI_PASSWORD = "YourPassword";
```

### 配置云端 API

```cpp
// firmware/src/config.cpp
const char* API_ENDPOINT = "https://api.yourserver.com";
const char* DEVICE_TOKEN = "your_device_token";
```

## 功能说明

### 语音采集
- **VAD (Voice Activity Detection)**：自动检测人声，静音时不录音
- **音频格式**：16kHz, 16bit, 单声道 WAV
- **缓冲机制**：32KB 循环缓冲区

### 数据上传
- **分段上传**：每 30 秒或达到 100KB 时上传
- **断网缓存**：本地存储最多 1MB 数据
- **重试机制**：失败后指数退避重试

### 电源管理
- **深度睡眠**：无活动 5 分钟后进入睡眠
- **按钮唤醒**：按下录音键唤醒设备
- **电量监测**：低于 10% 时闪烁红灯提醒

### 状态指示

| LED 状态 | 含义 |
|---------|------|
| 绿色常亮 | 正常录音中 |
| 蓝色闪烁 | 正在上传数据 |
| 红色闪烁 | 电量低 / 错误 |
| 黄色常亮 | WiFi 配置模式 |

## 开发指南

### 添加新功能

1. 在 `firmware/include/` 创建头文件
2. 在 `firmware/src/` 实现功能
3. 在 `main.cpp` 中调用

### OTA 更新

固件支持无线更新（OTA）：

```cpp
// 启用 OTA
#define ENABLE_OTA true
```

### 调试

```cpp
// 启用调试输出
#define DEBUG_MODE true

// 串口输出
Serial.println("Debug message");
```

## 物料成本（100 台批量）

| 组件 | 单价 (¥) | 备注 |
|-----|---------|------|
| ESP32-S3-WROOM-1 | 18 | 16MB Flash |
| INMP441 麦克风 | 8 | I2S 数字麦克风 |
| 500mAh 锂电池 | 12 | 带保护板 |
| TP4056 充电模块 | 2 | USB-C |
| RGB LED | 0.5 | WS2812 |
| 按钮开关 | 0.5 | 轻触开关 |
| PCB 板 | 15 | 双层板 |
| 外壳 | 25 | 3D 打印或注塑 |
| 其他电阻电容 | 3 | 被动元件 |
| **总计** | **~84** | 不含组装人工 |

## 安全与隐私

- 所有语音数据使用 **HTTPS** 加密传输
- 设备端不永久存储语音（上传后删除）
- 支持**本地加密**：设备生成 AES 密钥，云端存储加密数据
- 物理开关：用户可随时关闭麦克风

## PCB 设计

PCB 设计文件使用 KiCad 7.0 创建，位于 `pcb/` 目录。

### 打样说明
- **层数**：2 层
- **板厚**：1.6mm
- **铜厚**：1oz
- **表面处理**：HASL 无铅
- **阻焊颜色**：黑色

推荐供应商：嘉立创、捷配

## 外壳设计

3D 打印文件位于 `enclosure/` 目录。

### 打印参数
- **材料**：PLA / PETG
- **层高**：0.2mm
- **填充**：20%
- **支撑**：需要

## 故障排除

### 无法连接 WiFi
1. 检查 SSID 和密码是否正确
2. 确认路由器支持 2.4GHz
3. 查看串口输出的错误信息

### 无声音 / 声音失真
1. 检查麦克风接线是否正确
2. 确认 I2S 引脚配置正确
3. 调整采样率（16kHz 推荐）

### 上传失败
1. 检查 API_ENDPOINT 配置
2. 查看网络连接状态
3. 确认设备 token 有效

### 续航时间短
1. 启用 VAD 功能减少录音时间
2. 降低上传频率
3. 启用深度睡眠模式

## 许可证

MIT

## 参考资料

- [ESP32-S3 技术文档](https://www.espressif.com/sites/default/files/documentation/esp32-s3_datasheet_en.pdf)
- [INMP441 数据手册](https://invensense.tdk.com/wp-content/uploads/2015/02/INMP441.pdf)
- [ESP-IDF 编程指南](https://docs.espressif.com/projects/esp-idf/en/latest/)
