# Pulse - 拾笺

<p align="center">
  <img src="banner.svg" alt="Pulse 拾笺" width="100%" />
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

> 通过语音记录生活，让 AI 帮你在不同身份间自由切换

## 项目简介

拾笺 是一个智能身份管理系统，帮助用户在多重生活角色中自由切换。通过可穿戴硬件设备采集语音，云端 AI 分析理解你的对话内容，自动识别当前身份（员工、父母、朋友等），并生成个性化的报告。

### 核心功能

- 🎤 **智能语音采集**：可穿戴硬件设备，全天候记录重要对话
- 🧠 **AI 身份识别**：自动识别当前角色（工作、家庭、社交等）
- 📝 **智能信息提取**：提取待办事项、承诺、重要笔记
- 📊 **多维度报告**：按身份生成日报、周报、时间统计
- 🔒 **隐私优先**：端到端加密，用户完全掌控数据

## 项目架构

```
Pulse/
├── backend/          # 云端服务（Go）
├── frontend/         # 用户端 Web 应用（React）
├── hardware/         # 硬件设备（ESP32-S3）
├── docs/            # 项目文档
└── README.md        # 本文件
```

## 快速开始

### 前置要求

- **Docker 全栈（推荐 / 生产部署）**：仅需 Docker + Docker Compose，无需安装 Go / Node（构建在镜像内完成）
- **本地开发后端**：Go 1.26.x
- **本地开发前端**：Node.js 20+
- **硬件**：ESP32-S3

### 安装步骤

#### 0. Docker 全栈（推荐，生产部署亦用此方式）

```bash
# 1) 准备环境变量（复制后填写生产密钥，见下方说明）
cp .env.example .env

# 2) 构建并启动（依次 mysql → migrate → backend → frontend）
docker compose up -d --build

# 前端：http://localhost:5173 （FRONTEND_PORT 可改）
# 后端健康检查：curl http://localhost:8080/health
```

> **部署到服务器**：把仓库 clone 到服务器后执行上述两步即可（Dockerfile 为多阶段构建，服务器无需预装 Go / Node）。`.env` 中**至少**要改这些生产项：
> - `JWT_SECRET`、`MYSQL_ROOT_PASSWORD`、`MYSQL_PASSWORD` —— 默认是开发弱口令，务必替换
> - `STEPFUN_API_KEY`、`AI_API_KEY` —— 不填则语音转写与 AI 分析不可用
> - `AUDIO_ENCRYPTION_KEY` —— 生产建议配置（生成：`openssl rand -base64 32`）
>
> 注意：`docker compose` 读取的是**仓库根目录**的 `.env`（不是 `backend/.env`，后者仅用于 `go run` 本地开发）。

#### 1. 后端服务

```bash
cd backend
go mod download
cp .env.example .env
# 配置环境变量（数据库、API keys 等）
go run cmd/server/main.go
```

#### 2. 前端 App

```bash
cd frontend
npm install
# 配置环境变量（创建 .env 文件）
npm run dev
```

#### 3. 硬件设备

硬件采用 ESP32-S3，开发指南见 `hardware/README.md`。

详细文档请查看各模块的 README。

## 文档

- [后端设计](docs/backend-design.md) · [后端分阶段文档](docs/backend/)
- [前端设计](docs/frontend-design.md) · [前端分阶段文档](docs/frontend/)
- [后端 CI/CD](docs/backend/ci-cd.md) · [前端 CI/CD](docs/frontend/ci-cd.md)
- [项目状态](PROJECT_STATUS.md) · [硬件开发指南](hardware/README.md)

## 技术栈

### 后端
- **语言**：Go
- **框架**：Gin
- **AI SDK**：adk-go（Google Agent Development Kit）
- **STT 服务**：StepFun StepAudio-2.5-ASR
- **数据库**：MySQL
- **音频存储**：MySQL（audio_sessions.audio_data，LONGBLOB）

### 前端
- **框架**：React 18 + TypeScript
- **构建工具**：Vite
- **路由**：React Router
- **HTTP 客户端**：Axios
- **移动端**：待定（未来可能使用 React Native）

### 硬件
- **单片机**：ESP32-S3（Xtensa® 32 位 LX7 双核）
- **无线连接**：2.4 GHz Wi-Fi (IEEE 802.11b/g/n) + Bluetooth® 5 (LE)
- **RTOS**：FreeRTOS / RT-Thread
- **界面**：LVGL
- **开发环境**：VSCode + PlatformIO
- **语言**：C / Arduino

## 贡献指南

我们欢迎任何形式的贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详情。

### 开发流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 联系方式

- 项目主页：[GitHub](https://github.com/chaojixinren/pulse)
- 问题反馈：[Issues](https://github.com/chaojixinren/pulse/issues)

## 致谢

- 感谢所有贡献者

---

⭐ 如果这个项目对你有帮助，请给我们一个 Star！
