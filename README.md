# Pulse · 时笺

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)
![Status](https://img.shields.io/badge/status-active-success.svg)
![Go Version](https://img.shields.io/badge/Go-1.26.5-00ADD8.svg?logo=go)
![React](https://img.shields.io/badge/React-18.2-61DAFB.svg?logo=react)
![ESP32](https://img.shields.io/badge/ESP32-S3-lightgrey.svg)

> **通过语音记录生活，让 AI 帮你在不同身份间自由切换**

[文档](./docs/) · [快速开始](#-快速开始) · [贡献指南](./CONTRIBUTING.md)

---

## 🌟 为什么选择 Pulse

在忙碌的现代生活中，我们每个人都在扮演着多重角色——
作为员工处理工作事务，作为父母陪伴孩子成长，作为朋友分享欢笑，
作为子女倾听父母的唠叨。

每一个身份，都是一段独特的人生旅程。
每一次对话，都承载着重要的承诺与待办。

**Pulse（时笺）** 致力于温柔地记录这些瞬间，
用智能技术帮助你理解自己的生活轨迹，
让重要的事情不再被遗忘。

---

## 💫 核心理念

### 身份流转，智能记录

Pulse 不仅仅是一个语音记录工具，它理解你——
理解你此刻是工作状态还是家庭时间，
理解你在会议中做出的承诺，
理解你和朋友的闲聊中约定的下一次相聚。

### 数据主权，隐私优先

你的声音数据只属于你。端到端加密确保
即使是我们也无法窥探你的私密对话。

### 科技服务于人

我们相信技术应该是温和的、无形的。
Pulse 安静地记录，智能地分析，
让你专注于真正重要的事情——**活在当下**。

---

## 🎯 核心功能

### 🎤 智能语音采集
通过可穿戴设备（ESP32-S3）全天候记录重要对话，
支持离线缓存、智能上传，
确保关键时刻不被遗漏。

### 🧠 AI 身份识别
基于上下文理解自动识别你的当前身份：
工作、家庭、社交……无需手动切换，
Pulse 知道你此刻是谁。

### 📝 智能信息提取
从对话中自动提取：
- ✅ 待办事项（To-dos）
- 💬 承诺与约定（Commitments）
- 💡 重要笔记（Notes）
- 🔑 关键决策点（Decisions）

### 📊 多维报告生成
按身份、时间段、重要性生成个性化报告：
- 日报：今天做了什么
- 周报：本周各身份的时间分配
- 统计：时间都去哪了

### 🔒 隐私保护
- 音频数据 AES-256-GCM 加密存储
- 支持数据导出与账号注销
- 完整的 GDPR 合规设计

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    🎙️ 可穿戴硬件 (ESP32-S3)                    │
│                  语音采集 → WiFi 加密上传                       │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTPS
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                   ⚖️ 负载均衡 (Nginx/ALB)                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────────────┐
│              🚀 Gin Web Server (Go 1.26.5)                   │
│  ┌──────────┬──────────┬──────────┬──────────────────────┐ │
│  │ 认证中间件 │ 日志中间件 │ CORS 中间件│ 限流中间件           │ │
│  └──────────┴──────────┴──────────┴──────────────────────┘ │
│  ┌──────────┬──────────┬──────────┬──────────────────────┐ │
│  │ Auth API │Audio API │Identity  │    Report API        │ │
│  ├──────────┼──────────┼──────────┼──────────────────────┤ │
│  │Device API│Timeline  │   AI     │     Metrics          │ │
│  └──────────┴──────────┴──────────┴──────────────────────┘ │
└──────────────────────┬──────────────────────────────────────┘
                       │
            ┌──────────┴──────────┐
            │                     │
            ↓                     ↓
┌──────────────────┐       ┌──────────────────┐
│   🗄️ MySQL 8.0   │       │   🤖 AI Service  │
│  (主数据库+音频)   │       │  (adk-go + STT)  │
└──────────────────┘       └────────┬─────────┘
                                     │
                                     ↓
                            ┌──────────────────┐
                            │ StepFun ASR API  │
                            │ (语音转文字)       │
                            └──────────────────┘

┌─────────────────────────────────────────────────────────────┐
│              💻 React 前端 (TypeScript + Vite)               │
│  身份管理 · 时间线 · 设备管理 · 智能报告 · 数据可视化           │
└─────────────────────────────────────────────────────────────┘
```

---

## ⚡ 快速开始

### 🚀 Docker 全栈（推荐）

一键启动所有服务，最简单的方式：

```bash
git clone https://github.com/chaojixinren/pulse.git
cd pulse
docker compose up -d --build
```

访问 `http://localhost:5173` 即可体验完整系统 ✨

### 🛠️ 本地开发

#### 前置要求

- **Go** 1.26.5+
- **Node.js** 20+
- **MySQL** 8.0+
- **ESP32-S3**（可选，用于硬件开发）

#### 1️⃣ 后端服务

```bash
cd backend
go mod download
cp .env.example .env
# 配置环境变量（数据库、API keys 等）
go run cmd/server/main.go
```

后端服务将在 `http://localhost:8080` 运行

#### 2️⃣ 前端应用

```bash
cd frontend
npm install
npm run dev
```

前端应用将在 `http://localhost:5173` 运行

#### 3️⃣ 硬件设备

硬件采用 ESP32-S3，开发指南详见 [`hardware/README.md`](hardware/README.md)

---

## 🛠️ 技术栈

### 后端

| 层级 | 技术选型 |
|------|---------|
| 语言 | [Go 1.26.5](https://go.dev/) |
| 框架 | [Gin](https://gin-gonic.com/) |
| AI SDK | [adk-go](https://github.com/google/adk-go) (Google Agent Development Kit) |
| STT 服务 | [StepFun StepAudio-2.5-ASR](https://platform.stepfun.com/) |
| 数据库 | [MySQL 8.0+](https://www.mysql.com/) |
| 音频存储 | MySQL LONGBLOB (AES-256-GCM 加密) |
| 可观测性 | Prometheus Metrics + OpenTelemetry Tracing |

### 前端

| 层级 | 技术选型 |
|------|---------|
| 框架 | [React 18](https://react.dev/) + [TypeScript](https://www.typescriptlang.org/) |
| 构建工具 | [Vite](https://vitejs.dev/) |
| 路由 | [React Router](https://reactrouter.com/) |
| HTTP 客户端 | [Axios](https://axios-http.com/) |
| 测试 | [Vitest](https://vitest.dev/) + [Playwright](https://playwright.dev/) |
| 代码质量 | ESLint + Prettier |

### 硬件

| 组件 | 规格 |
|------|------|
| 单片机 | ESP32-S3 (Xtensa® 32 位 LX7 双核) |
| 无线 | 2.4 GHz WiFi + Bluetooth® 5 (LE) |
| 实时系统 | FreeRTOS / RT-Thread |
| 界面框架 | [LVGL](https://lvgl.io/) |
| 开发环境 | VSCode + PlatformIO |

---

## 📚 文档导航

### 📖 核心文档

- [项目状态看板](PROJECT_STATUS.md) - 当前进度与里程碑
- [贡献指南](CONTRIBUTING.md) - 如何参与项目开发
- [后端设计文档](docs/backend-design.md) - 系统架构与 API 设计
- [前端设计文档](docs/frontend-design.md) - 组件架构与 UI 规范

### 📂 分阶段文档

**后端开发文档**
- [后端 README](docs/backend/README.md) - 快速开始
- [Phase 1: MVP](docs/backend/phase-1-mvp.md) - 核心功能闭环
- [Phase 2: AI 增强](docs/backend/phase-2-ai.md) - 智能身份识别
- [Phase 3: 生产化](docs/backend/phase-3-production.md) - 安全与可观测性

**前端开发文档**
- [前端 README](docs/frontend/README.md) - 快速开始
- [Phase 1: MVP](docs/frontend/phase-1-mvp.md) - 身份管理时间线
- [Phase 2: 设备与 AI](docs/frontend/phase-2-device-ai.md) - 智能展示
- [Phase 3: 生产化](docs/frontend/phase-3-production.md) - 测试与部署

**CI/CD 文档**
- [后端 CI/CD](docs/backend/ci-cd.md) - GitHub Actions 工作流
- [前端 CI/CD](docs/frontend/ci-cd.md) - 自动化测试部署
- [硬件开发指南](hardware/README.md) - ESP32-S3 固件开发

---

## 🗺️ 开发路线图

- [x] 项目架构设计
- [x] 后端 Phase 1-3（MVP + AI + 生产化）
- [x] 前端 Phase 1-3（MVP + AI + 生产化）
- [x] 全栈 Docker Compose 部署
- [ ] 硬件原型开发（MVP）
- [ ] 内测版本（Beta）
- [ ] 正式发布（v1.0.0）

> **当前状态**：后端与前端已完成三阶段开发，具备完整功能。硬件固件正在开发中，预计下一个里程碑将迎来完整的内测版本。

---

## 🤝 参与贡献

我们欢迎所有形式的贡献！无论是：

- 🐛 报告 Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 提交代码修复
- 🌍 多语言翻译

请查看 [贡献指南](CONTRIBUTING.md) 了解详情。

### 🏗️ 开发流程

```bash
# 1. Fork 本仓库
# 2. 创建特性分支
git checkout -b feature/AmazingFeature

# 3. 提交更改
git commit -m 'Add some AmazingFeature'

# 4. 推送到分支
git push origin feature/AmazingFeature

# 5. 提交 Pull Request
```

---

## 🧪 测试

### 后端测试

```bash
cd backend

# 运行单元测试 + 集成测试
go test ./... -race

# 运行真实 MySQL e2e 测试
go test -tags e2e ./test/
```

### 前端测试

```bash
cd frontend

# 运行单元测试
npm test

# 运行 E2E 测试
npm run test:e2e

# 测试覆盖率报告
npm run test:coverage
```

> **测试状态**：后端 100% 通过 · 前端 209 单元 + 33 E2E 测试全绿 ✅

---

## 📦 部署

### Docker Compose

```bash
# 启动全栈
docker compose up -d --build

# 查看日志
docker compose logs -f

# 停止服务
docker compose down
```

### 手动部署

详细部署指南请参考 [后端 CI/CD](docs/backend/ci-cd.md) 和 [前端 CI/CD](docs/frontend/ci-cd.md)

---

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

## 💬 联系方式

- **项目主页**：[GitHub](https://github.com/chaojixinren/pulse)
- **问题反馈**：[Issues](https://github.com/chaojixinren/pulse/issues)
- **讨论交流**：[Discussions](https://github.com/chaojixinren/pulse/discussions)

---

## 🙏 致谢

感谢所有为这个项目付出时间和精力的贡献者们。
特别感谢以下开源项目和服务：

- [StepFun](https://platform.stepfun.com/) - 提供优质的语音识别服务
- [Google ADK](https://github.com/google/adk-go) - AI Agent 开发框架
- [LVGL](https://lvgl.io/) - 优秀的嵌入式 GUI 库

---

## ⭐ 支持我们

如果 **Pulse · 时笺** 对你有所帮助，或者你认同我们的理念，
请给我们一个 ⭐ **Star**！

你的支持是我们持续前进的动力 💪

---

<div align="center">

### 记录每一刻 · 理解每一种身份

**让科技温柔地记住你的生活轨迹**

[回到顶部](#pulse--时笺) · [开始使用](#-快速开始) · [加入我们](#-参与贡献)

</div>
