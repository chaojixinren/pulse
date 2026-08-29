# Pulse · 拾笺

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)
![Status](https://img.shields.io/badge/status-active-success.svg)
![Go Version](https://img.shields.io/badge/Go-1.26.5-00ADD8.svg?logo=go)
![React](https://img.shields.io/badge/React-18.2-61DAFB.svg?logo=react)
![ESP32](https://img.shields.io/badge/ESP32-S3-lightgrey.svg)

> **拾起每一个瞬间 · 笺记每一段人生**

[文档](./docs/) · [快速开始](#-快速开始) · [贡献指南](./CONTRIBUTING.md)

---

## 🌟 为什么选择拾笺

在忙碌的现代生活中，我们每个人都在扮演着多重角色——
作为员工处理工作事务，作为父母陪伴孩子成长，作为朋友分享欢笑，
作为子女倾听父母的唠叨，作为伴侣经营彼此的感情。

每一个身份，都是一段独特的人生旅程。
每一次对话，都承载着重要的承诺与待办。
每一个瞬间，都值得被温柔地记住。

**拾笺（Pulse）** 诞生于一个简单的信念：
**科技应该是温暖的，而不是冰冷的。**

我们用智能语音技术帮你拾起那些容易被遗忘的瞬间，
用 AI 理解你此刻的身份，
用精美的报告让你看见自己的生活轨迹。

让重要的事情不再被遗忘，
让每一份承诺都被铭记，
让每一种身份都被尊重。

---

## 💫 我们的理念

### 🎭 身份流转，智能记录

我们相信，最好的科技是**无形的**。

拾笺不仅仅是一个语音记录工具，它理解你——
理解你此刻是工作状态还是家庭时间，
理解你在会议中做出的承诺，
理解你和朋友的闲聊中约定的下一次相聚，
理解你对家人说"今晚回家吃饭"的温柔。

它知道你此刻是谁，所以你可以**真正地活在当下**，
无需分心去记录，无需费心去管理。

### 🔐 数据主权，隐私至上

你的声音，只属于你。

我们支持对录音进行 **AES-256-GCM** 加密存储，
为每一段录音提供落库保护。

你拥有完整的数据主权：
- 随时导出你的所有数据
- 随时注销你的账户
- 完全掌控你的数字生活

我们相信：**隐私不是功能，而是权利。**

### 🌱 科技服务于人

我们追求的不是更复杂的技术，
而是更简单的体验。

拾笺安静地记录，智能地分析，
让你专注于真正重要的事情——
与家人相处时的欢笑，
与朋友相聚时的畅谈，
与自己独处时的思考。

**技术在后台运行，生活在台前继续。**

---

## 🎯 核心能力

### 🎙️ 智能语音采集

通过可穿戴设备（ESP32-S3）全天候记录重要对话，
支持**离线缓存**和**智能上传**，
确保在 WiFi 信号不佳时也不会丢失任何珍贵瞬间。

**关键特性**：
- 7×24 小时持续录音
- 智能静音检测（仅在对话时录制）
- 本地加密存储，安全同步
- 一键快速回放

### 🧠 AI 身份识别

基于深度上下文理解，自动识别你的当前身份：
工作、家庭、社交、独处……**无需手动切换**。

拾笺知道你此刻是谁，因为：
- 分析对话中的关键词和语境
- 学习你的行为模式和时间规律
- 识别对话对象的关系属性
- 持续优化识别的准确度

**你只需要做自己，剩下的交给拾笺。**

### 📝 智能信息提取

从自然对话中自动提取结构化信息：

| 类型 | 说明 | 示例 |
|------|------|------|
| ✅ **待办事项** | 需要完成的任务 | "明天要把报告发给客户" |
| 💬 **承诺约定** | 对他人的承诺 | "周末一起吃饭" |
| 💡 **重要笔记** | 值得记住的点子 | "这个方案想法不错" |
| 🔑 **关键决策** | 做出的重要决定 | "就选第三个方案" |
| 📅 **日程安排** | 未来的活动 | "下周三下午开会" |

所有信息自动关联到对应身份，帮你构建完整的**个人知识图谱**。

### 📊 多维报告系统

按身份、时间段、重要性生成个性化报告：

**📋 日报** — 今天发生了什么
- 各身份时间分布
- 重要对话摘要
- 待办事项清单

**📈 周报** — 本周的生活轨迹
- 时间分配可视化
- 身份切换频率
- 承诺完成情况

**📊 统计** — 时间的去向
- 各身份时间占比
- 活动趋势分析
- 生活平衡指数

### 🔒 企业级隐私保护

- 音频数据支持 **AES-256-GCM** 加密存储（配置密钥后生效）
- 支持完整数据**导出**（JSON 格式）
- 支持**账号注销**（软删除并吊销全部登录凭证）
- 提供符合 GDPR / 个保法要求的数据导出与注销机制

---

## 🗺️ 使用场景

### 👔 职场人士
- **会议记录**：自动记录会议内容，提取行动项
- **客户沟通**：记住客户的偏好和承诺
- **项目跟进**：追踪项目进展和待办事项

### 👨‍👩‍👧 家庭生活
- **亲子时光**：记录孩子的成长瞬间和重要约定
- **夫妻沟通**：记住重要的纪念日和承诺
- **家庭决策**：追踪家庭讨论的决策点

### 👥 社交达人
- **朋友聚会**：记住朋友的近况和下次相聚的约定
- **社交网络**：维护重要的人际关系
- **活动策划**：追踪活动筹备的细节

### 🎯 自我提升
- **个人反思**：通过报告了解自己的生活模式
- **习惯养成**：追踪目标完成情况
- **时间管理**：优化各身份的时间分配

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

**包含服务**：
- ✅ MySQL 8.0 数据库
- ✅ 后端 API 服务（Go）
- ✅ 前端 Web 应用（React）
- ✅ 自动数据库迁移

### 🛠️ 本地开发环境

#### 前置要求

- **Go** 1.26.5+ — [下载地址](https://go.dev/dl/)
- **Node.js** 20+ — [下载地址](https://nodejs.org/)
- **MySQL** 8.0+ — [下载地址](https://dev.mysql.com/downloads/)
- **ESP32-S3**（可选）— 用于硬件开发

#### 1️⃣ 后端服务

```bash
cd backend

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入数据库、API keys 等配置

# 初始化数据库
go run ./cmd/migrate

# 启动服务（开发模式）
go run cmd/server/main.go
```

后端服务将在 `http://localhost:8080` 运行

**后端文档**：[docs/backend/README.md](docs/backend/README.md)

#### 2️⃣ 前端应用

```bash
cd frontend

# 安装依赖
npm install

# 配置环境变量（可选）
cp .env.example .env

# 启动开发服务器
npm run dev
```

前端应用将在 `http://localhost:5173` 运行

**前端文档**：[docs/frontend/README.md](docs/frontend/README.md)

#### 3️⃣ 硬件设备（可选）

硬件采用 **ESP32-S3** 单片机，开发指南详见 [`hardware/README.md`](hardware/README.md)

---

## 🛠️ 技术栈

### 后端技术

| 层级 | 技术选型 | 用途 |
|------|---------|------|
| **语言** | [Go 1.26.5](https://go.dev/) | 高性能后端服务 |
| **框架** | [Gin](https://gin-gonic.com/) | Web 框架与路由 |
| **AI SDK** | [adk-go](https://github.com/google/adk-go) | Google Agent 开发框架 |
| **STT 服务** | [StepFun StepAudio-2.5-ASR](https://platform.stepfun.com/) | 语音转文字 |
| **数据库** | [MySQL 8.0+](https://www.mysql.com/) | 主数据库与音频存储 |
| **加密** | AES-256-GCM | 音频数据加密 |
| **可观测性** | Prometheus + OpenTelemetry |  metrics、tracing、日志 |

### 前端技术

| 层级 | 技术选型 | 用途 |
|------|---------|------|
| **框架** | [React 18](https://react.dev/) + [TypeScript](https://www.typescriptlang.org/) | 现代 Web 应用 |
| **构建工具** | [Vite](https://vitejs.dev/) | 极速开发体验 |
| **路由** | [React Router](https://reactrouter.com/) | 客户端路由管理 |
| **HTTP 客户端** | [Axios](https://axios-http.com/) | API 请求封装 |
| **测试** | [Vitest](https://vitest.dev/) + [Playwright](https://playwright.dev/) | 单元 + E2E 测试 |
| **代码质量** | ESLint + Prettier | 代码规范与格式化 |

### 硬件技术

| 组件 | 规格 | 说明 |
|------|------|------|
| **单片机** | ESP32-S3 (Xtensa® 32 位 LX7 双核) | 主控芯片 |
| **无线连接** | 2.4 GHz WiFi + Bluetooth® 5 (LE) | 网络与通信 |
| **实时系统** | FreeRTOS / RT-Thread | 任务调度 |
| **界面框架** | [LVGL](https://lvgl.io/) | 嵌入式 GUI 库 |
| **开发环境** | VSCode + PlatformIO | 嵌入式开发 |

---

## 📚 文档导航

### 📖 核心文档

- [项目状态看板](PROJECT_STATUS.md) — 当前进度与里程碑
- [贡献指南](CONTRIBUTING.md) — 如何参与项目开发
- [后端设计文档](docs/backend-design.md) — 系统架构与 API 设计
- [前端设计文档](docs/frontend-design.md) — 组件架构与 UI 规范

### 📂 分阶段文档

**后端开发**
- [快速开始](docs/backend/README.md)
- [Phase 1: MVP](docs/backend/phase-1-mvp.md) — 核心功能闭环
- [Phase 2: AI 增强](docs/backend/phase-2-ai.md) — 智能身份识别
- [Phase 3: 生产化](docs/backend/phase-3-production.md) — 安全与可观测性

**前端开发**
- [快速开始](docs/frontend/README.md)
- [Phase 1: MVP](docs/frontend/phase-1-mvp.md) — 身份管理时间线
- [Phase 2: 设备与 AI](docs/frontend/phase-2-device-ai.md) — 智能展示
- [Phase 3: 生产化](docs/frontend/phase-3-production.md) — 测试与部署

**CI/CD 文档**
- [后端 CI/CD](docs/backend/ci-cd.md) — GitHub Actions 工作流
- [前端 CI/CD](docs/frontend/ci-cd.md) — 自动化测试部署
- [硬件开发指南](hardware/README.md) — ESP32-S3 固件开发

---

## 🗺️ 开发路线图

- [x] 项目架构设计
- [x] 后端 Phase 1-3（MVP + AI + 生产化）
- [x] 前端 Phase 1-3（MVP + AI + 生产化）
- [x] 全栈 Docker Compose 部署
- [ ] 硬件原型开发（MVP）
- [ ] **内测版本（Beta）** — 即将推出 🎉
- [ ] 正式发布（v1.0.0）

> **当前状态**：后端与前端已完成三阶段开发，具备完整功能。硬件固件正在开发中，预计下一个里程碑将迎来完整的内测版本。欢迎加入我们的测试计划！🚀

---

## 🤝 参与贡献

我们真诚欢迎所有形式的贡献！无论是：

- 🐛 **报告 Bug** — 发现问题请告诉我们
- 💡 **功能建议** — 有好点子欢迎分享
- 📝 **改进文档** — 让文档更清晰易懂
- 🔧 **代码贡献** — 提交 PR 帮助我们改进
- 🌍 **多语言翻译** — 让拾笺走向世界
- ⭐ **项目推荐** — 给项目一个 Star 就是最大支持

请查看 [贡献指南](CONTRIBUTING.md) 了解详情。

### 🏗️ 开发流程

```bash
# 1. Fork 本仓库到你的 GitHub 账号

# 2. 克隆到本地
git clone https://github.com/YOUR_USERNAME/pulse.git
cd pulse

# 3. 创建特性分支
git checkout -b feature/AmazingFeature

# 4. 开始开发，提交更改
git commit -m 'feat: add AmazingFeature'

# 5. 推送到你的 Fork
git push origin feature/AmazingFeature

# 6. 提交 Pull Request
# 访问 https://github.com/chaojixinren/pulse/compare
```

---

## 🧪 测试

### 后端测试

```bash
cd backend

# 运行单元测试 + 集成测试（sqlmock，无需外部依赖）
go test ./... -race

# 运行真实 MySQL 端到端测试（需要数据库）
go test -tags e2e ./test/
```

### 前端测试

```bash
cd frontend

# 运行单元测试
npm test

# 监听模式（开发时使用）
npm run test:watch

# 运行 E2E 测试
npm run test:e2e

# 生成测试覆盖率报告
npm run test:coverage
```

> **测试状态**：后端 100% 通过 · 前端 **209** 单元测试 + **33** E2E 测试全绿 ✅

---

## 📦 部署

### Docker Compose（生产推荐）

```bash
# 启动全栈
docker compose up -d --build

# 查看所有服务日志
docker compose logs -f

# 只查看后端日志
docker compose logs -f backend

# 停止服务
docker compose down

# 停止并删除数据卷（谨慎使用）
docker compose down -v
```

**服务端口**：
- 🌐 **前端应用**：`http://localhost:5173`（可配置 `FRONTEND_PORT`）
- 🔧 **后端 API**：`http://localhost:8080`
- 🗄️ **MySQL 数据库**：`localhost:3306`

### 手动部署

详细部署指南请参考：
- [后端 CI/CD 文档](docs/backend/ci-cd.md)
- [前端 CI/CD 文档](docs/frontend/ci-cd.md)

---

## 📄 许可证

本项目采用 [MIT](LICENSE) 许可证 — 自由使用，友好分享。

---

## 💬 联系我们

- **项目主页**：[GitHub](https://github.com/chaojixinren/pulse)
- **问题反馈**：[Issues](https://github.com/chaojixinren/pulse/issues)
- **讨论交流**：[Discussions](https://github.com/chaojixinren/pulse/discussions)
- **更新日志**：[Releases](https://github.com/chaojixinren/pulse/releases)

---

## 🙏 致谢

感谢所有为这个项目付出时间和精力的贡献者们。

特别感谢以下开源项目和服务：

- 🤖 [StepFun](https://platform.stepfun.com/) — 提供优质的语音识别服务
- 🧠 [Google ADK](https://github.com/google/adk-go) — 强大的 AI Agent 开发框架
- 🎨 [LVGL](https://lvgl.io/) — 优秀的嵌入式 GUI 库
- 🌟 所有为我们提供反馈和建议的用户

---

## ⭐ 支持我们

如果 **拾笺（Pulse）** 对你有所帮助，
或者你认同我们的理念，
请给我们一个 ⭐ **Star**！

你的支持是我们持续前进的动力 💪

---

<div align="center">

### 🌟 拾起每一个瞬间 · 笺记每一段人生

**让科技温柔地记住你的生活轨迹**

---

**拾笺 · Pulse** — 你的智能身份空间

[🌐 官方网站](https://github.com/chaojixinren/pulse) ·
[📖 文档](./docs/) ·
[💬 社区](https://github.com/chaojixinren/pulse/discussions) ·
[🐛 报告问题](https://github.com/chaojixinren/pulse/issues)

---

*Made with ❤️ by Pulse Team*

</div>
