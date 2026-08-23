# Pulse Backend

云端服务，负责语音处理、AI 分析、数据存储等核心功能。

## 技术栈

- **语言**：Go 1.21+
- **框架**：Gin
- **AI SDK**：adk-go + eino
- **STT 服务**：StepFun StepAudio-2.5-ASR
- **数据库**：MySQL 8.0+
- **缓存**：Redis 7+
- **对象存储**：七牛云

## 项目结构

```
backend/
├── cmd/
│   └── server/
│       └── main.go       # 应用入口
├── internal/
│   ├── api/              # API 路由
│   │   ├── auth.go       # 用户认证
│   │   ├── audio.go      # 语音上传
│   │   ├── identity.go   # 身份管理
│   │   └── report.go     # 报告生成
│   ├── service/          # 业务逻辑
│   │   ├── ai.go         # AI 分析服务 (adk-go + eino)
│   │   └── storage.go    # 存储服务
│   ├── model/            # 数据模型
│   │   ├── user.go
│   │   ├── identity.go
│   │   └── audio_session.go
│   ├── middleware/       # 中间件
│   └── config/           # 配置文件
├── pkg/                  # 公共包
│   └── utils/            # 工具函数
├── migrations/           # 数据库迁移
├── .env.example          # 环境变量模板
├── go.mod                # Go 模块依赖
└── README.md            # 本文件
```

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 文件，填入以下配置：

```env
# 服务配置
PORT=8080
GIN_MODE=debug

# 数据库
DATABASE_URL=postgresql://user:password@localhost:5432/pulse
REDIS_URL=redis://localhost:6379

# 对象存储
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
S3_BUCKET=pulse-audio

# AI 服务 (adk-go + eino)
AI_API_KEY=your_ai_api_key
AI_BASE_URL=https://api.example.com

# JWT 密钥
JWT_SECRET=your_secret_key
```

### 3. 初始化数据库

```bash
# 运行迁移
go run cmd/migrate/main.go
```

### 4. 启动服务

```bash
# 开发模式
go run cmd/server/main.go

# 或使用 air 热重载
air
```

服务将在 `http://localhost:8080` 运行。

## API 端点

### 认证

- `POST /api/auth/register` - 用户注册
- `POST /api/auth/login` - 用户登录
- `POST /api/auth/refresh` - 刷新 token

### 语音处理

- `POST /api/audio/upload` - 上传语音文件
- `GET /api/audio/:id` - 获取语音记录
- `GET /api/audio/sessions` - 获取语音会话列表

### 身份管理

- `GET /api/identities` - 获取用户的所有身份
- `POST /api/identities` - 创建新身份
- `PUT /api/identities/:id` - 更新身份
- `DELETE /api/identities/:id` - 删除身份

### 报告生成

- `GET /api/reports/daily/:date` - 获取日报
- `GET /api/reports/weekly/:date` - 获取周报
- `GET /api/reports/stats` - 获取统计数据

完整 API 文档请查看 [API Documentation](../docs/api.md)。

## 开发

### 运行测试

```bash
npm test
```

### 代码检查

```bash
npm run lint
```

### 数据库迁移

```bash
# 创建新迁移
npm run migrate:create migration_name

# 运行迁移
npm run migrate

# 回滚迁移
npm run migrate:rollback
```

## 部署

详见 [部署指南](../docs/deployment.md)。

## 许可证

MIT
