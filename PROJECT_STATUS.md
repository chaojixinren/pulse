# Pulse 项目架构（更新版）

## 🎯 项目概述

Pulse 是一个智能身份空间系统，通过可穿戴硬件设备采集用户的日常对话，利用 AI 自动识别用户当前的身份角色，并生成个性化的报告和提醒。

## 📚 技术栈（已确定）

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **AI SDK**: adk-go + eino
- **STT 服务**: StepFun StepAudio-2.5-ASR
- **数据库**: MySQL 8.0+ (主数据)
- **缓存**: Redis 7+ (会话、热点数据)
- **存储**: 七牛云 (语音文件)

### 前端
- **框架**: React 18+
- **语言**: TypeScript
- **构建工具**: Vite
- **路由**: React Router
- **HTTP 客户端**: Axios
- **移动端**: 待定（未来可能使用 React Native）

### 硬件
- **单片机**: ESP32-S3 系列芯片（Xtensa® 32 位 LX7 双核微处理器）
- **无线连接**: 2.4 GHz Wi-Fi (IEEE 802.11b/g/n) 和 Bluetooth® 5 (LE)
- **存储扩展**: 封装内可叠封 1.8 V / 3.3 V flash 和 PSRAM
- **GPIO**: 45 个 GPIO 管脚（QFN56，7×7 mm）
- **RTOS**: FreeRTOS / RT-Thread
- **界面框架**: LVGL
- **开发环境**: VSCode + PlatformIO
- **编程语言**: C / Arduino

## 📁 项目结构

```
Pulse/
├── backend/                    # Go 后端服务
│   ├── cmd/
│   │   └── server/
│   │       └── main.go        # 应用入口
│   ├── internal/
│   │   ├── api/               # API 路由
│   │   ├── service/           # 业务逻辑（AI 分析等）
│   │   ├── model/             # 数据模型
│   │   ├── middleware/        # 中间件
│   │   └── config/            # 配置
│   ├── pkg/
│   │   └── utils/             # 工具函数
│   ├── migrations/            # 数据库迁移
│   ├── go.mod                 # Go 依赖
│   ├── .env.example           # 环境变量模板
│   └── README.md
│
├── frontend/                   # React 前端应用
│   ├── src/
│   │   ├── main.tsx           # 应用入口
│   │   ├── App.tsx            # 根组件
│   │   ├── components/        # 通用组件
│   │   ├── pages/             # 页面组件
│   │   │   ├── Auth/
│   │   │   ├── Device/
│   │   │   ├── Identity/
│   │   │   ├── Timeline/
│   │   │   └── Report/
│   │   ├── services/          # API 服务
│   │   ├── hooks/             # 自定义 Hooks
│   │   ├── utils/             # 工具函数
│   │   ├── types/             # TypeScript 类型
│   │   └── styles/            # 样式文件
│   ├── public/                # 静态资源
│   ├── package.json           # 依赖
│   └── README.md
│
├── hardware/                   # 硬件设备（ESP32-S3）
│   ├── firmware/              # 固件代码
│   ├── pcb/                   # PCB 设计
│   ├── enclosure/             # 外壳设计
│   └── docs/                  # 硬件文档
│
├── docs/                       # 项目文档
│   ├── architecture.md        # 系统架构
│   ├── api.md                 # API 文档
│   ├── ai-prompts.md          # AI Prompt 设计
│   ├── hardware-guide.md      # 硬件开发指南
│   ├── deployment.md          # 部署指南
│   └── PROJECT_OVERVIEW.md    # 项目总览
│
├── README.md                   # 项目主文档
├── LICENSE                     # MIT 许可证
├── CONTRIBUTING.md             # 贡献指南
└── .gitignore                 # Git 忽略配置
```

## 🚀 快速开始

### 后端（Go）

```bash
cd backend

# 下载依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件

# 运行服务
go run cmd/server/main.go
```

### 前端（React）

```bash
cd frontend

# 安装依赖
npm install

# 配置环境变量
# 创建 .env 文件，设置 VITE_API_BASE_URL

# 运行开发服务器
npm run dev
```

### 硬件

✅ 硬件技术栈已确定（ESP32-S3），参考 `hardware/README.md` 开始原型开发

## 🔗 Git 仓库

```bash
# 初始化 Git（如果还没有）
git init

# 添加远程仓库
git remote add origin https://github.com/chaojixinren/pulse.git

# 提交代码
git add .
git commit -m "Initial commit: project architecture"
git branch -M main
git push -u origin main
```

## 📋 开发路线图

### Phase 1: 后端 API 开发（3-4 周）
- [ ] 搭建 Go 项目基础架构
- [ ] 实现用户认证（JWT）
- [ ] 实现数据库模型和迁移
- [ ] 实现语音上传接口
- [ ] 集成 adk-go + eino 进行 AI 分析
- [ ] 实现身份管理 CRUD
- [ ] 实现时间线和报告 API

### Phase 2: 前端 Web 应用（3-4 周）
- [ ] 搭建 React + TypeScript 项目
- [ ] 实现认证页面（登录/注册）
- [ ] 实现身份管理界面
- [ ] 实现时间线查看
- [ ] 实现报告中心
- [ ] 实现设备管理（预留）

### Phase 3: 硬件开发（ESP32-S3）
- [x] 确定硬件技术栈（ESP32-S3）
- [ ] 采购开发板和组件
- [ ] 实现音频采集
- [ ] 实现网络通信
- [ ] 实现云端上传

### Phase 4: 集成与测试（2-3 周）
- [ ] 前后端集成测试
- [ ] 硬件与云端集成
- [ ] 用户验收测试
- [ ] 性能优化
- [ ] 部署上线

## 📝 待决策事项

1. **前端状态管理方案**
   - Redux Toolkit?
   - Zustand?
   - Jotai?
   - React Context?

2. **前端 UI 组件库**
   - Material-UI?
   - Ant Design?
   - Chakra UI?
   - Tailwind CSS + Headless UI?

3. ~~**硬件技术栈**~~ ✅ 已确定：ESP32-S3 系列

4. **部署方案**
   - Docker + Kubernetes?
   - 云服务提供商?
   - 单服务器部署?

## 🎯 下一步行动

### 立即可做
1. ✅ 项目架构已设计完成
2. ✅ 技术栈已明确（后端 Go + 前端 React）
3. ⬜ 推送代码到 GitHub
4. ⬜ 创建项目看板（GitHub Projects）

### 本周可做
1. ⬜ 开始后端 Go 项目开发
   - 实现基础的 Gin 服务器
   - 连接 PostgreSQL 数据库
   - 实现用户认证

2. ⬜ 开始前端 React 项目开发
   - 搭建项目框架
   - 实现路由结构
   - 创建基础页面布局

### 后续待办
1. ⬜ 集成 adk-go + eino AI SDK
2. ⬜ 实现完整的 API 接口
3. ✅ 决定硬件技术栈（ESP32-S3）
4. ⬜ 制定详细的开发计划

## 📖 重要文档

- **系统架构**: [docs/architecture.md](docs/architecture.md)
- **API 文档**: [docs/api.md](docs/api.md)
- **AI Prompt 设计**: [docs/ai-prompts.md](docs/ai-prompts.md)
- **项目总览**: [docs/PROJECT_OVERVIEW.md](docs/PROJECT_OVERVIEW.md)
- **部署指南**: [docs/deployment.md](docs/deployment.md)

## ⚠️ 重要提醒

1. **技术栈由项目负责人决定** - 不要自行更改已确定的技术选型
2. **硬件技术栈已确定** - 采用 ESP32-S3 系列芯片
3. **文档需要更新** - 部分文档仍基于旧技术栈（Node.js/Flutter），需要逐步更新

---

**项目状态**: 架构设计完成，技术栈已确定（后端 Go，前端 React）  
**当前进度**: 15%（架构设计 + 技术栈确定）  
**下一里程碑**: 后端 API 基础框架搭建

**最后更新**: 2024-08-23
