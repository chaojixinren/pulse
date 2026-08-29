# Pulse Backend

云端服务，负责语音处理、AI 分析、数据存储等核心功能。

## 技术栈

- **语言**：Go 1.26.5
- **框架**：Gin
- **AI SDK**：adk-go（Google Agent Development Kit）
- **STT 服务**：StepFun StepAudio-2.5-ASR
- **数据库**：MySQL 8.0+
- **音频存储**：MySQL（音频二进制存于 audio_sessions 表）

## 项目结构

```
backend/
├── cmd/
│   ├── server/           # 应用入口
│   └── migrate/          # 数据库迁移命令
├── internal/
│   ├── api/              # HTTP 处理器与路由（auth/audio/identity/timeline/report/device）
│   ├── service/          # 业务逻辑（auth/audio/stt/ai/identity/timeline/report/device）
│   ├── repository/       # 数据访问层（user/refresh_token/identity/audio_session/device）
│   ├── model/            # 数据模型（user/refresh_token/identity/audio_session/device/extraction）
│   ├── middleware/       # 中间件（auth/logger/cors/error_handler）
│   ├── worker/           # 后台处理（转写 → AI 分析流水线）
│   └── config/           # 配置加载（config/database）
├── pkg/                  # 公共包（errors/logger/prompt/response/utils）
├── test/                 # 真实基础设施 e2e（-tags e2e）
├── migrations/           # SQL 迁移（001_init.sql、002_phase2.sql）
├── Dockerfile
├── .env.example          # 环境变量模板
├── go.mod                # Go 模块依赖
└── README.md             # 本文件
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
DATABASE_DSN=user:password@tcp(localhost:3306)/pulse?charset=utf8mb4&parseTime=True&loc=Local

# AI 服务 (adk-go)
AI_API_KEY=your_ai_api_key
AI_BASE_URL=https://api.openai.com/v1
AI_MODEL=gpt-4o-mini
# 身份识别置信度阈值（低于该值不自动绑定身份）
AI_CONFIDENCE_THRESHOLD=0.6

# STT 服务 (StepFun StepAudio-2.5-ASR)
STEPFUN_API_KEY=your_stepfun_api_key
STEPFUN_STT_MODEL=stepaudio-2.5-asr
STEPFUN_API_BASE=https://api.stepfun.com/v1

# JWT 密钥
JWT_SECRET=your_random_secret_key_change_in_production
# access token 有效期（JWT）
JWT_EXPIRES_IN=1h
# refresh token 有效期
REFRESH_TOKEN_TTL=168h

# CORS 配置
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# 日志级别
LOG_LEVEL=info

# 语音处理配置
MAX_AUDIO_SIZE=52428800
AUDIO_RETENTION_DAYS=30
```

### 3. 初始化数据库

```bash
# 运行迁移
go run ./cmd/migrate
```

### 4. 启动服务

```bash
# 开发模式
go run ./cmd/server

# 或使用 air 热重载
air
```

服务将在 `http://localhost:8080` 运行。

## API 端点

### 健康检查

- `GET /health` - 健康检查（MySQL 可用性）

### 认证

- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新 token
- `POST /api/v1/auth/logout` - 登出
- `GET /api/v1/auth/me` - 当前用户信息

### 语音处理

- `POST /api/v1/audio/upload` - 上传语音文件
- `POST /api/v1/audio/:id/retry` - 重试转写

### 身份管理

- `GET /api/v1/identities` - 获取用户的所有身份
- `POST /api/v1/identities` - 创建新身份
- `PUT /api/v1/identities/:id` - 更新身份
- `DELETE /api/v1/identities/:id` - 删除身份
- `PUT /api/v1/identities/:id/default` - 设为默认身份

### 时间线

- `GET /api/v1/timeline` - 按身份查看语音会话时间线

### 报告

- `GET /api/v1/reports/daily` - 获取日报

### 设备管理（Phase 2）

- `POST /api/v1/devices` - 创建设备绑定，一次性返回 device_token 供手抄到硬件
- `GET /api/v1/devices` - 设备列表
- `GET /api/v1/devices/:id` - 设备详情
- `DELETE /api/v1/devices/:id` - 解绑设备
- `POST /api/v1/devices/:id/heartbeat` - 设备心跳
- `POST /api/v1/devices/:id/command` - 下发指令
- `POST /api/v1/device/audio/upload` - 设备态上传（Authorization: Device <token>）
- `POST /api/v1/device/heartbeat` - 设备态心跳（捎带待执行指令）
- `POST /api/v1/device/commands/:id/ack` - 设备回执指令

详细接口约定见 [后端设计](../docs/backend-design.md) 与 [分阶段开发文档](../docs/backend/README.md)。

## 开发

### 运行测试

```bash
go test ./... -race        # 单元 + 集成测试（sqlmock，无需外部依赖）
go test -tags e2e ./test/  # 真实 MySQL e2e
```

### 代码检查

```bash
go vet ./...
golangci-lint run
```

### 数据库迁移

```bash
# 运行数据库迁移
go run ./cmd/migrate
```

## 部署与 CI/CD

### Docker Compose 一键启动

```bash
docker compose up -d --build
```

会拉起 MySQL + 迁移 + 后端服务，详见 [CI/CD 说明](../docs/backend/ci-cd.md)。

### 常用命令

```bash
make build        # 编译
make test         # 单元/集成测试
make test-e2e     # 真实 MySQL e2e
make docker-up    # 启动全栈
```

### CI/CD

- CI：`.github/workflows/ci.yml`（gofmt + go vet + 单测/竞态 + Phase 1/2/3 真实 MySQL e2e 矩阵 + 镜像构建）
- CD：`.github/workflows/release.yml`（打 `v*` tag 推送镜像到 GHCR 并创建 Release）

详见 [docs/backend/ci-cd.md](../docs/backend/ci-cd.md)。

## 许可证

MIT
