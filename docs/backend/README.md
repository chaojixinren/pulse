# Pulse 后端模块开发文档（总览）

> 本文档是 Pulse 后端的分阶段开发指南，按「跑通闭环 → AI 增强 → 生产化」三个阶段组织。
> 每个阶段一个独立文档，内部按模块拆分，包含职责、数据结构、接口签名、依赖与验收标准。

## 技术栈（已确定）

- **语言/框架**：Go 1.21+ / Gin
- **AI SDK**：adk-go + eino
- **STT**：StepFun StepAudio-2.5-ASR
- **数据库**：MySQL 8.0+（主数据）
- **缓存**：Redis 7+（会话、热点数据）
- **音频存储**：MySQL（音频二进制存于 audio_sessions 表）
- **认证**：JWT

## 阶段总览

| 阶段 | 目标 | 核心模块 | 完成标志 |
|------|------|----------|----------|
| [Phase 1：MVP](phase-1-mvp.md) | 跑通「上传音频 → 转写 → 查看」最小闭环 | 骨架、认证、音频上传、MySQL 存储、语音会话、STT 转写、身份、时间线、日报 | 硬件上传的音频能被转写并在前端时间线查看 |
| [Phase 2：AI 增强](phase-2-ai.md) | 从转写文本提取结构化信息并推送提醒 | AI 分析（身份识别/信息提取）、设备管理、提醒中心 | 系统能自动识别身份、提取待办/承诺并提醒 |
| [Phase 3：生产化](phase-3-production.md) | 达到可上线标准 | 报告增强、加密存储、数据删除/导出、限流配额、可观测性、部署 | 通过安全检查，可灰度上线 |

## 后端目录结构（目标态）

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # 入口：加载配置、初始化依赖、启动 HTTP
├── internal/
│   ├── api/                     # HTTP 处理器（按资源拆分）
│   │   ├── router.go            # 路由注册
│   │   ├── handler.go           # 统一响应 / 错误处理
│   │   ├── auth.go
│   │   ├── audio.go
│   │   ├── identity.go
│   │   ├── timeline.go
│   │   ├── device.go            # Phase 2
│   │   ├── report.go
│   │   └── reminder.go          # Phase 2
│   ├── service/                 # 业务逻辑层
│   │   ├── auth.go
│   │   ├── audio.go
│   │   ├── stt.go               # StepFun STT 封装
│   │   ├── ai.go                # adk-go + eino 编排
│   │   ├── identity.go
│   │   ├── timeline.go
│   │   ├── report.go
│   │   ├── device.go            # Phase 2
│   │   └── reminder.go          # Phase 2
│   ├── repository/              # 数据访问层（与 service 分离）
│   │   ├── user.go
│   │   ├── identity.go
│   │   ├── audio_session.go
│   │   ├── device.go
│   │   └── reminder.go
│   ├── model/                   # 数据模型 struct
│   │   ├── user.go
│   │   ├── identity.go
│   │   ├── audio_session.go
│   │   ├── device.go
│   │   ├── refresh_token.go
│   │   └── reminder.go
│   ├── middleware/              # 中间件
│   │   ├── auth.go              # JWT 认证
│   │   ├── logger.go
│   │   ├── cors.go
│   │   └── ratelimit.go         # Phase 3
│   ├── worker/                  # 后台处理
│   │   └── audio_processor.go
│   └── config/
│       └── config.go            # 配置加载
├── pkg/
│   ├── logger/                  # zap 封装
│   ├── response/                # 统一 JSON 响应
│   └── utils/
├── migrations/                  # SQL 迁移文件
│   ├── 001_init.sql
│   └── ...
├── .env.example
├── go.mod
└── README.md
```

> 说明：现有仓库的 `internal/` 只有 api/config/middleware/model/service 五个空目录。
> 本文档建议新增 `repository`（数据访问层）与 `worker`（后台处理）两个目录，实现更清晰的分层。

## 分层约定

```
HTTP 请求 → api(handler) → service(业务逻辑) → repository(数据访问) → MySQL
                                  │
                                  ├→ stt(StepFun)
                                  └→ ai(adk-go + eino)

后台任务 → worker → service → ...
```

- **api 层**：只做参数解析、校验、调用 service、返回统一响应；不写业务逻辑。
- **service 层**：核心业务逻辑、事务边界、调用外部服务（stt/ai）。
- **repository 层**：纯 SQL 访问，不掺业务判断。
- **model 层**：struct 定义，对应数据库表。

## 通用约定

### 1. 统一响应结构

```go
type Response struct {
    Code    int         `json:"code"`    // 0 表示成功
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

### 2. 错误处理

- service 层返回 Go error，api 层统一映射为 HTTP 状态码 + 错误信息。
- 业务错误用自定义错误类型区分「参数错误 / 未授权 / 未找到 / 内部错误」。

### 3. 命名与风格

- 文件名与包名一致（`auth.go` → `package api` 内 `AuthHandler`）。
- handler 方法签名统一：`func (h *XxxHandler) Yyy(c *gin.Context)`。
- 数据库访问统一走 repository，禁止在 service 里直接写 SQL。

### 4. 时间处理

- 数据库存 UTC，`recorded_at`、`created_at` 等均为 UTC。
- 展示层（前端）再转本地时区；后端 API 统一返回 RFC3339 格式。

## 模块地图

| 模块 | 阶段 | 涉及目录 | 依赖 |
|------|------|----------|------|
| 项目骨架 | 1 | cmd、config、pkg/logger、pkg/response | 无 |
| 认证 | 1 | api/auth、service/auth、model/user、model/refresh_token | 骨架 |
| 音频上传 + 存储 | 1 | api/audio、service/audio、MySQL（audio_data） | 认证、骨架 |
| 语音会话状态机 | 1 | model/audio_session、repository/audio_session | 骨架 |
| STT 转写 | 1 | service/stt、worker/audio_processor | 会话、存储 |
| 身份管理 | 1 | api/identity、service/identity、model/identity | 认证 |
| 时间线 | 1 | api/timeline、service/timeline | 会话、身份 |
| 日报 | 1 | api/report、service/report | 时间线 |
| AI 分析 | 2 | service/ai（adk-go + eino） | STT、身份 |
| 设备管理 | 2 | api/device、service/device、model/device | 认证 |
| 提醒中心 | 2 | api/reminder、service/reminder、model/reminder | AI 分析 |
| 报告增强 | 3 | service/report | 时间线、AI |
| 加密存储 | 3 | service/audio | 存储 |
| 数据删除/导出 | 3 | api/export、service/export | 各模块 |
| 限流配额 | 3 | middleware/ratelimit | 认证 |
| 可观测性 | 3 | pkg/logger、middleware | 骨架 |
| 部署 | 3 | docker-compose、CI | 全部 |

## 开发顺序建议

严格按阶段推进，**上一阶段验收通过再进入下一阶段**：

1. Phase 1 先搭骨架 + 认证，因为所有后续接口都依赖 JWT 中间件。
2. Phase 1 的音频上传 → 存储 → 会话 → STT → 时间线 是一条强依赖链，按此顺序开发。
3. Phase 2 的 AI 分析依赖 Phase 1 的 STT 与身份，提醒中心依赖 AI 分析。
4. Phase 3 可并行推进：加密存储、限流、可观测性相互独立。

## 相关文档

- [Phase 1：MVP 开发文档](phase-1-mvp.md)
- [Phase 2：AI 增强开发文档](phase-2-ai.md)
- [Phase 3：生产化开发文档](phase-3-production.md)
- [后端整体设计](../backend-design.md)
