# PCB Design

Pulse 语音采集设备的 PCB 设计文件，使用 **嘉立创EDA** 设计。

## 目录结构

```
pcb/
├── pcb/Pulse.pcbdoc          # PCB 布局文件（Altium）
├── schematic/Pulse.schdoc    # 原理图文件（Altium）
├── Pulse.epro2               # 嘉立创EDA 集成项目文件
├── gerber/                   # Gerber 输出文件（用于打样）
│   ├── Gerber_TopLayer.GTL
│   ├── Gerber_BottomLayer.GBL
│   ├── Gerber_InnerLayer1.G1
│   ├── Gerber_InnerLayer2.G2
│   ├── Gerber_TopSolderMaskLayer.GTS
│   ├── Gerber_BottomSolderMaskLayer.GBS
│   ├── Gerber_TopSilkscreenLayer.GTO
│   ├── Gerber_BottomSilkscreenLayer.GBO
│   ├── Gerber_TopPasteMaskLayer.GTP
│   ├── Gerber_BottomPasteMaskLayer.GBP
│   ├── Gerber_BoardOutlineLayer.GKO
│   ├── Gerber_DocumentLayer.GDL
│   ├── Gerber_DrillDrawingLayer.GDD
│   ├── Drill_PTH_Through.DRL
│   ├── Drill_NPTH_Through.DRL
│   ├── Drill_PTH_Through_Via.DRL
│   └── FlyingProbeTesting.json
├── docs/
│   ├── BOM.csv               # 物料清单
│   ├── PickAndPlace.csv      # SMT 坐标文件
│   └── Schematic.pdf         # 原理图 PDF 导出
└── models/Pulse.step         # PCB 3D 模型（STEP 格式）
```

## 设计规格

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

## 板卡功能模块

### 主控与存储

| 元件 | 型号 | 说明 |
|------|------|------|
| U1 | ESP32-S3R8 | 主控，Xtensa LX7 双核，8MB Flash / 8MB PSRAM |
| U2 | W25Q128JVPIQ | 16MB SPI Flash（外部存储） |
| X1 | 40 MHz | 主时钟晶振 |
| X2 | 32.768 kHz | RTC 时钟晶振（接 BM8563） |

### 电源管理

| 元件 | 型号 | 说明 |
|------|------|------|
| U4 | AXP2101 | 电源管理 IC（PMIC），支持 PMU + 充放电管理 |
| U5 | BM8563EMA | 实时时钟（RTC），I2C 接口 |
| USB1 | MC-121-L119 | USB-C 连接器，支持充电与数据传输 |
| D1 | USBLC6-2SC6 | USB ESD 保护器件 |

### 音频采集

| 元件 | 型号 | 说明 |
|------|------|------|
| MIC1 | MSM261D4030H1CPM | MEMS 数字麦克风，I2S 接口，全向拾音 |
| Q1 | AO3400A | 麦克风电源控制 MOSFET |

### 显示与交互

| 元件 | 型号 | 说明 |
|------|------|------|
| FPC1 | HC-FPPC-05-09-18RLTAG | 18-pin FPC 连接器（接 LCD 显示屏） |
| KEY_DWN1 | TS24CA | 录音按键 |
| KEY_PWR1 | TS24CA | 电源按键 |
| KEY_RST1 | TS24CA | 复位按键 |

### 无线连接

| 元件 | 型号 | 说明 |
|------|------|------|
| U3 | ANT3216LL00R2400A | 2.4 GHz 陶瓷天线（WiFi / BLE） |

### 外部存储

| 元件 | 型号 | 说明 |
|------|------|------|
| SIM1 | SDCARD-1 (Molex) | Micro SD 卡座（SMD 焊接） |

## 板卡尺寸

PCB 外形约为 **35 mm × 31 mm**，四角有 M2 螺丝安装孔（SREW1/2/3）。

## 版本记录

| 版本 | 日期 | 变更内容 |
|------|------|---------|
| v1.0 | 2026-08 | 初始版本，完成核心功能布局 |

## 关联文档

- [硬件总 README](../README.md) — 整体硬件设计说明
- [固件代码](../firmware/) — ESP32-S3 固件
- [外壳设计](../enclosure/) — 3D 打印外壳
