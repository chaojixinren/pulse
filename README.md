# Pulse - 时笺

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

> 通过语音记录生活，让 AI 帮你在不同身份间自由切换

## 项目简介

时笺 是一个智能身份管理系统，帮助用户在多重生活角色中自由切换。通过可穿戴硬件设备采集语音，云端 AI 分析理解你的对话内容，自动识别当前身份（员工、父母、朋友等），并生成个性化的报告。

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

- **后端**：Go 1.21+
- **前端**：Node.js 18+
- **硬件**：ESP32-S3

### 安装步骤

#### 1. 后端服务

```bash
cd backend
go mod download
cp .env.example .env
# 配置环境变量（数据库、Redis、API keys 等）
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

- [系统架构设计](docs/architecture.md)
- [API 文档](docs/api.md)
- [硬件开发指南](docs/hardware-guide.md)
- [AI Prompt 设计](docs/ai-prompts.md)
- [部署指南](docs/deployment.md)

## 技术栈

### 后端
- **语言**：Go
- **框架**：Gin
- **AI SDK**：adk-go（Google Agent Development Kit）
- **STT 服务**：StepFun StepAudio-2.5-ASR
- **数据库**：MySQL + Redis
- **对象存储**：七牛云

### 前端
- **框架**：React
- **移动端**：待定（未来可能使用 React Native）

### 硬件
- **单片机**：ESP32-S3（Xtensa® 32 位 LX7 双核）
- **无线连接**：2.4 GHz Wi-Fi (IEEE 802.11b/g/n) + Bluetooth® 5 (LE)
- **RTOS**：FreeRTOS / RT-Thread
- **界面**：LVGL
- **开发环境**：VSCode + PlatformIO
- **语言**：C / Arduino

## 开发路线图

- [x] 项目架构设计
- [ ] 硬件原型开发（MVP）
- [ ] 云端 API 服务
- [ ] AI 分析引擎
- [ ] 用户端 App（基础功能）
- [ ] 内测版本
- [ ] 正式发布

## 开发指南

### 快速 Setup

```bash
# 1. 运行配置脚本
./setup.sh

# 2. 启动开发环境
make docker-up

# 3. 访问服务
# - 后端 API: http://localhost:8080
# - 前端: http://localhost:5173
```

### 常用命令

```bash
# 后端
make build          # 编译后端
make test           # 运行测试
make lint           # Lint 检查
make docker-up      # 启动开发环境
make docker-down    # 停止开发环境

# 前端
cd frontend
npm run dev         # 启动开发服务器
npm run lint        # ESLint 检查
npm run type-check  # TypeScript 类型检查
npm run build       # 生产构建
npm test            # 运行测试
```

### 分支策略

- `main` / `master` - 生产环境
- `develop` - 开发分支
- `feature/xxx` - 功能开发分支
- `fix/xxx` - Bug 修复分支
- `docs/xxx` - 文档更新分支

### Git 工作流程

我们遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```bash
# 创建功能分支
git checkout -b feature/user-authentication

# 提交（会自动触发 Git hooks 检查）
git commit -m "feat(auth): add OAuth2 authentication"

# 推送并创建 PR
git push origin feature/user-authentication
```

### Code Review

所有 PR 都需要经过：

1. ✅ CI/CD 自动化检查
2. 👀 至少一位维护者审查
3. ✅ LGTM（Looks Good To Me）
4. 🔀 Squash Merge

详细流程请查看 [CONTRIBUTING.md](CONTRIBUTING.md)。

## CI/CD

项目使用 GitHub Actions 进行自动化：

- **Backend CI**: Lint → Test → E2E → Docker Build
- **Frontend CI**: Lint → Type-Check → Test → Build
- **Release**: 自动创建 Release 和 Changelog

查看 [.github/workflows/](.github/workflows/) 了解详情。

## 贡献指南

我们欢迎任何形式的贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解：

- 📖 完整的开发流程
- 💻 代码规范（Go / TypeScript / C++）
- 📝 提交信息规范
- 🔀 Code Review 流程
- 🌟 社区准则

### 贡献者

感谢所有为 Pulse 项目做出贡献的人！ 🙏

[贡献者列表](https://github.com/yourusername/pulse/graphs/contributors)

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 联系方式

- 项目主页：[GitHub](https://github.com/yourusername/pulse)
- 问题反馈：[Issues](https://github.com/yourusername/pulse/issues)

## 致谢

- 感谢所有贡献者

---

⭐ 如果这个项目对你有帮助，请给我们一个 Star！
