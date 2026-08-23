# Phase 1：MVP 开发文档

> **目标**：跑通「硬件上传音频 → MySQL 存储 → STT 转写 → 前端时间线查看」的最小闭环。
> **完成标志**：硬件上传的音频能被转写，并在前端时间线按身份查看。

## 模块依赖关系

```
[1 项目骨架] → [2 认证] → [3 音频上传+存储] → [4 语音会话] → [5 STT 转写] → [7 时间线]
                                    ↘                              ↗
                                     [6 身份管理] → [8 日报]
```

按顺序开发：骨架和认证是地基，之后 3→4→5→7 是主链路，6 可与 3/4 并行，8 依赖 7。

---

## 模块 1：项目骨架

### 职责
搭建可运行的 Gin 服务、配置加载、日志、统一响应，为所有后续模块提供基础。

### 目录 / 文件
```
cmd/server/main.go          # 入口：加载配置 → 初始化 logger/DB/Redis → 注册路由 → 启动
internal/config/config.go   # 从 .env 读取配置，封装 Config struct
pkg/logger/logger.go        # zap 封装，暴露全局 Log
pkg/response/response.go    # 统一 JSON 响应 + 错误映射
internal/api/router.go      # 路由注册入口（先挂 /health）
internal/api/handler.go     # Handler 基类 / 通用辅助
```

### 数据结构
```go
// internal/config/config.go
type Config struct {
    Port        string
    GINMode     string
    MySQLDSN    string
    RedisURL    string
    StepFunAPIKey  string
    StepFunBaseURL string
    JWTSecret      string
    JWTExpiresIn   time.Duration
}
```

### 关键接口
- `GET /health` → `{"code":0,"message":"ok"}`，同时 ping MySQL/Redis，任一失败返回 503。

### 验收标准
- [ ] `go run cmd/server/main.go` 可启动，`/health` 返回 200。
- [ ] .env 缺失时启动报错并给出明确提示。
- [ ] 请求日志（method、path、status、耗时）通过 zap 输出。

---

## 模块 2：认证

### 职责
用户注册、登录、JWT 签发与校验、Refresh Token 续期。所有后续接口依赖 JWT 中间件。

### 数据模型
```sql
CREATE TABLE users (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,   -- bcrypt
    name VARCHAR(100) NOT NULL,
    avatar_url TEXT,
    settings JSON DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE refresh_tokens (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME
);
```

### 目录 / 文件
```
internal/api/auth.go
internal/service/auth.go
internal/repository/user.go
internal/repository/refresh_token.go
internal/model/user.go
internal/model/refresh_token.go
internal/middleware/auth.go   # JWT 校验中间件
```

### 接口签名
```go
// service/auth.go
func (s *AuthService) Register(ctx, email, password, name string) (*model.User, error)
func (s *AuthService) Login(ctx, email, password string) (*TokenPair, error)
func (s *AuthService) Refresh(ctx, refreshToken string) (*TokenPair, error)
func (s *AuthService) Logout(ctx, refreshToken string) error

// model 返回
type TokenPair struct {
    AccessToken  string
    RefreshToken string
}
```

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/register | 注册 |
| POST | /api/v1/auth/login | 登录，返回 access + refresh token |
| POST | /api/v1/auth/refresh | 用 refresh token 换新 token 对 |
| POST | /api/v1/auth/logout | 注销（吊销 refresh token） |
| GET  | /api/v1/auth/me | 当前用户信息（需认证） |

### 安全要点
- 密码用 bcrypt（cost ≥ 10）。
- Access token 短期（如 1h），refresh token 长期（如 7 天），refresh token 只存哈希。
- JWT 中间件解析后把 `user_id` 写入 context，供后续 handler 读取。

### 验收标准
- [ ] 注册后密码在库里为 bcrypt 哈希，不存明文。
- [ ] 未带 token 访问受保护接口返回 401。
- [ ] refresh token 可续期，logout 后 refresh token 失效。

---

## 模块 3：音频上传 + MySQL 存储

### 职责
接收硬件上传的音频文件，校验格式，把音频二进制直接写入 MySQL，并创建语音会话记录。

### 目录 / 文件
```
internal/api/audio.go
internal/service/audio.go
internal/model/audio_session.go
```

### 存储方式
音频二进制不经过对象存储，由 `AudioSessionRepo.Create` 直接写入 `audio_sessions.audio_data`（LONGBLOB）；转写时 worker 从该字段读取字节。Phase 3 在落库前增加 AES-256-GCM 加密（见 phase-3-production.md）。

### 上传流程
```
1. 从 multipart 取 file，读 header 参数：device_id、duration、recorded_at
2. 校验文件格式（WAV/MP3/M4A）与大小上限
3. 生成 session_id（UUID），推断 audio_mime（audio/wav、audio/mpeg 等）
4. 读入字节 → AudioSessionRepo.Create 写入 audio_sessions.audio_data
5. 创建 audio_session 记录（status=pending），进入异步转写队列
6. 同步返回 session_id
```

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/audio/upload | multipart 上传音频（需认证） |

### 验收标准
- [ ] 上传成功返回 session_id，audio_sessions.audio_data 中能查到对应音频二进制。
- [ ] 非法格式 / 超大文件返回 400，不落库。
- [ ] 落库失败返回 5xx，且不产生孤儿记录。

---

## 模块 4：语音会话状态机

### 职责
定义语音处理的生命周期，提供状态流转、重试、错误记录。

### 数据模型
```sql
CREATE TABLE audio_sessions (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    identity_id CHAR(36) REFERENCES identities(id) ON DELETE SET NULL,
    device_id VARCHAR(100),
    audio_data LONGBLOB NOT NULL,      -- 音频二进制
    audio_mime VARCHAR(100) NULL,
    transcript TEXT,
    duration INTEGER NOT NULL,              -- 秒
    file_size BIGINT,                       -- 字节
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    extracted_data JSON DEFAULT '{}',
    ai_confidence DECIMAL(3,2),
    recorded_at DATETIME NOT NULL,
    processed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 状态机
```
pending → processing → completed
                 ↘ failed → (重试) → processing
```

状态取值：`pending` / `processing` / `completed` / `failed`。

### repository 接口签名
```go
func (r *AudioSessionRepo) Create(ctx, s *model.AudioSession) error
func (r *AudioSessionRepo) GetByID(ctx, id string) (*model.AudioSession, error)
func (r *AudioSessionRepo) UpdateStatus(ctx, id, status, errMsg string) error
func (r *AudioSessionRepo) UpdateTranscript(ctx, id, transcript string) error
func (r *AudioSessionRepo) ListByUser(ctx, userID string, filter Filter, page, size int) ([]model.AudioSession, int64, error)
```

### 验收标准
- [ ] 状态流转合法（不允许 completed → failed 等非法跳转）。
- [ ] 失败记录 error_message，可重试。
- [ ] 所有状态变更更新 updated_at。

---

## 模块 5：STT 转写

### 职责
封装 StepFun StepAudio-2.5-ASR，把 MySQL 中的音频二进制转为文本，回写会话记录。

### 目录 / 文件
```
internal/service/stt.go
internal/worker/audio_processor.go   # 异步：拉取 pending 会话 → 转写 → 回写
```

### stt 接口签名
```go
type SttService struct { ... }

// Transcribe 音频字节 → 转写文本
func (s *SttService) Transcribe(ctx, data []byte, filename string) (text string, err error)
```

### 处理流程（异步 worker）
```
1. 定时/轮询取 status=pending 的会话（单 worker 轮询 DB）
2. 置 status=processing
3. 读 audio_data，调 stt.Transcribe(bytes, filename)
4. 成功：写 transcript，置 completed；失败：置 failed + error_message
5. 结果写 Redis 缓存（key: session:{id}）
```

### 注意
- Phase 1 为跑通闭环，采用**简单轮询 + 单 worker** 即可。
- STT 调用要设超时，失败要可重试（指数退避）。

### 验收标准
- [ ] 上传一段音频，若干秒后会话状态变为 completed，transcript 有内容。
- [ ] StepFun 调用失败时状态为 failed，可手动触发重试。

---

## 模块 6：身份管理

### 职责
身份的增删改查、默认身份、图标/颜色，为时间线过滤和 Phase 2 的身份识别提供基础。

### 数据模型
```sql
CREATE TABLE identities (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#000000',
    icon VARCHAR(50) DEFAULT 'person',
    is_default BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    -- MySQL 不支持部分唯一索引，用生成列实现"每个用户仅一个默认身份"
    default_user_id CHAR(36) GENERATED ALWAYS AS (
        CASE WHEN is_default = TRUE AND deleted_at IS NULL THEN user_id ELSE NULL END
    ) STORED
);
CREATE INDEX idx_identities_user_id ON identities(user_id);
CREATE UNIQUE INDEX idx_identities_user_default ON identities(default_user_id);
```

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET    | /api/v1/identities | 当前用户所有身份 |
| POST   | /api/v1/identities | 创建 |
| PUT    | /api/v1/identities/:id | 更新 |
| DELETE | /api/v1/identities/:id | 软删除 |
| PUT    | /api/v1/identities/:id/default | 设为默认 |

### 验收标准
- [ ] 每个用户最多一个默认身份（生成列唯一索引兜底）。
- [ ] 删除默认身份后需指定新默认，或禁止删除唯一默认身份。

---

## 模块 7：时间线

### 职责
按时间倒序返回用户的语音会话列表，支持按身份、日期范围、状态过滤和分页。

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/timeline | 分页查询，query: identity_id, from, to, status, page, size |

### 响应结构
```go
type TimelineItem struct {
    SessionID    string    `json:"session_id"`
    IdentityID   string    `json:"identity_id,omitempty"`
    Transcript   string    `json:"transcript"`
    Duration     int       `json:"duration"`
    Status       string    `json:"status"`
    RecordedAt   time.Time `json:"recorded_at"`
}
```

### 验收标准
- [ ] 列表按 recorded_at 倒序，分页正确。
- [ ] 按身份/日期/状态过滤生效。
- [ ] 只返回当前用户自己的数据（越权访问他人返回空/403）。

---

## 模块 8：日报（简单版）

### 职责
按天聚合某用户的语音记录，返回当日概览：会话数、总时长、按身份分布、待办/笔记（Phase 1 先空，Phase 2 由 AI 填充）。

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/reports/daily?date=YYYY-MM-DD | 当日报告 |

### 响应结构
```go
type DailyReport struct {
    Date          string             `json:"date"`
    SessionCount  int                `json:"session_count"`
    TotalDuration int                `json:"total_duration"`
    ByIdentity    []IdentityStat     `json:"by_identity"`
    Todos         []string           `json:"todos"`         // Phase 2 填充
    Notes         []string           `json:"notes"`         // Phase 2 填充
}
```

### 验收标准
- [ ] 指定日期无数据时返回空报告而非报错。
- [ ] 会话数、总时长、身份分布统计正确。

---

## 验收结论（自动化测试逐项核对）

> 执行 `cd backend && go test ./...`，共 **99** 项测试断言全部通过（0 失败）。
> 测试分层：**单元测试**（model/pkg 纯逻辑）、**集成测试**（repository/service 用 go-sqlmock 模拟 MySQL，STT 用 httptest 假服务）、**E2E**（httptest 全路由；另有 `//go:build e2e` 的 `test/e2e_live_test.go` 面向真实 MySQL/Redis，需设置 `TEST_DATABASE_DSN`/`TEST_REDIS_URL`）。

| 模块 | 验收标准 | 结果 | 对应测试 |
|------|---------|------|---------|
| 1 骨架 | /health 200、.env 缺失明确报错、zap 请求日志 | ✅ | `TestE2EHealth` / `TestLoadMissingEnv` / `middleware.Logger` |
| 2 认证 | bcrypt 不存明文、401、refresh 续期、logout 失效 | ✅ | `TestAuthRegister` / `TestAuthMiddleware*` / `TestAuthRefresh` / `TestAuthLogout` |
| 3 上传+存储 | 格式(魔数)/大小校验、落库 MySQL、返回 session_id | ✅ | `TestAudioUpload*` / `TestDetectAudioExt` / `TestE2EUploadTimelineReport` |
| 4 状态机 | 合法流转、error_message、updated_at | ✅ | `TestCanTransition` / `TestAudioSessionRepoUpdateStatus*` |
| 5 STT | completed+transcript、失败置 failed 可重试 | ✅ | `worker.TestAudioProcessor*` / `TestSttTranscribe*` / `TestAudioRetry` |
| 6 身份 | 唯一默认、禁删默认 | ✅ | `TestIdentity*` / `TestE2EIdentityCRUD` |
| 7 时间线 | 倒序分页、过滤、只返回本人数据 | ✅ | `TestTimeline*` / `TestAudioSessionRepoListByUser` |
| 8 日报 | 空报告、统计正确 | ✅ | `TestReportDaily*` |

### 已修复的验收发现

以下 4 项已在本阶段修复并有测试覆盖：

1. **`JWT_EXPIRES_IN` 现在作用于 access token**：默认 `1h`；新增 `REFRESH_TOKEN_TTL`（默认 `168h`）用于 refresh token，`service/issueTokens` 直接读取配置（`.env.example` 已同步）。测试：`TestAuthAccessTokenUsesConfigTTL`。
2. **重试流转**：改为 `failed → processing`（与状态机一致），worker 通过 `ListProcessable`（pending + processing）认领重试任务。测试：`TestAudioRetry/failed_transitions_to_processing`、`TestAudioSessionRepoListProcessable`。
3. **音频格式魔数校验**：上传时按文件头魔数识别 WAV/MP3/M4A，内容与扩展名不符返回 400。测试：`TestDetectAudioExt`、`TestAudioUploadContentExtensionMismatch`。
4. **`recorded_at` 非法**：解析失败返回 400，不再静默回退。测试：`TestE2EUploadInvalidRecordedAt`。

---

## Phase 1 整体验收清单

- [x] `/health` 正常，能连 MySQL + Redis。
- [x] 用户可注册/登录，受保护接口需 JWT。
- [x] 硬件可上传音频，返回 session_id。
- [x] 音频自动转写为文本，会话状态为 completed。
- [x] 用户可管理身份，时间线按身份/时间查看转写文本。
- [x] 日报返回当日统计。
