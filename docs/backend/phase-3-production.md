# Phase 3：生产化开发文档

> **目标**：让系统达到可上线标准——补全报告、加密存储、合规能力、限流、可观测性与部署。
> **完成标志**：通过安全检查，可灰度上线。

## 模块概览

本阶段模块相互独立，可并行推进：

| 模块 | 依赖 | 说明 |
|------|------|------|
| 1 报告增强 | Phase1 时间线 + Phase2 AI | 周报、统计图表 |
| 2 加密存储 | Phase1 音频上传 | 音频 AES-256 加密 |
| 3 数据删除/导出 | 各模块 | 合规（GDPR/个保法） |
| 4 限流配额 | 认证 | 防刷、配额 |
| 5 可观测性 | 骨架 | 日志、trace、指标 |
| 6 部署 | 全部 | docker-compose、CI/CD |

---

## 模块 1：报告增强

### 职责
在 Phase 1 日报基础上，增加周报、时间统计、身份占比分析、趋势图表数据。

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/reports/daily?date= | 日报（已有） |
| GET | /api/v1/reports/weekly?week= | 周报 |
| GET | /api/v1/reports/stats?from=&to= | 统计汇总（图表数据） |

### 周报结构
```go
type WeeklyReport struct {
    Week           string         `json:"week"`
    SessionCount   int            `json:"session_count"`
    TotalDuration  int            `json:"total_duration"`
    ByIdentity     []IdentityStat `json:"by_identity"`
    TopTodos       []string       `json:"top_todos"`
    CommitmentsDone int           `json:"commitments_done"`
    DailyTrend     []DailyPoint   `json:"daily_trend"`   // 每天会话数/时长
}
```

### 性能策略
- 统计查询数据量大时，用**预计算 + 缓存**：定时任务（如每天凌晨）生成日报/周报存 Redis，请求先查缓存。
- 图表数据按天聚合，避免实时扫全表。

### 验收标准
- [ ] 周报统计正确，含趋势与身份分布。
- [ ] 大时间范围查询有缓存，响应在可接受范围（< 500ms）。

---

## 模块 2：加密存储

### 职责
音频二进制在 MySQL 中**加密存储**，写入前 AES-256-GCM 加密，读取时解密。

### 目录 / 文件
```
internal/service/audio.go      # 落库前加密 / 读取时解密
```

### 接口签名
```go
// 落库前加密（Phase 3 在 AudioService.Upload 中调用）
func encryptAudio(data []byte, key []byte) ([]byte, error)

// 读取时解密
func decryptAudio(data []byte, key []byte) ([]byte, error)
```

### 加密方案
- 算法 AES-256-GCM，密钥 32 字节。
- 密钥管理：生产环境用 KMS/环境变量注入，**密钥不入库、不硬编码**。
- 每个用户或每个文件可用独立数据加密密钥（DEK），主密钥（KEK）管理 DEK（进阶，可先全局单密钥）。

### 存储侧
- 加密后的密文写入 `audio_sessions.audio_data`（LONGBLOB），不再以明文落库。
- 转写 worker 读取后先解密再提交 StepFun。

### 验收标准
- [ ] audio_data 中存储的是密文，直接读取无法播放。
- [ ] 后端解密后可正常 STT 转写。
- [ ] 密钥不落库、不硬编码在代码中。

---

## 模块 3：数据删除 / 导出（合规）

### 职责
满足用户「删除我的数据」「导出我的数据」的权利，覆盖音频、转写、身份等。

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET    | /api/v1/export | 导出用户全部数据（打包返回） |
| DELETE | /api/v1/account | 注销账号（软删用户 + 异步清理数据） |
| DELETE | /api/v1/audio/:id | 删除单条语音（含音频二进制） |

### 删除策略
1. 用户注销 → 软删用户（deleted_at），立即阻断登录。
2. 异步任务清理：删除 audio_data 音频二进制、删除/匿名化转写文本、删除身份。
3. 保留审计日志（匿名化），满足「已删除」证明。

### 导出策略
- 导出为 JSON + 音频文件打包（zip）。
- 异步生成下载链接，避免大文件阻塞 HTTP。

### 验收标准
- [ ] 注销后无法登录，数据异步清理完成。
- [ ] 导出包包含用户全部数据，格式可读。
- [ ] 删除单条语音同时清理音频二进制，不产生孤儿数据。

---

## 模块 4：限流与配额

### 职责
保护上传/登录等敏感接口，限制每用户音频配额，防刷与成本控制。

### 目录 / 文件
```
internal/middleware/ratelimit.go
internal/service/quota.go     # 配额计算
```

### 限流维度
- 全局：接口级 QPS（如登录 10 次/分钟/IP）。
- 用户级：上传次数、音频总时长/总字节数（日/月配额）。
- 用 Redis 计数器 + 滑动窗口实现。

### 接口签名
```go
// middleware/ratelimit.go
func RateLimit(limit int, window time.Duration) gin.HandlerFunc

// service/quota.go
func (s *QuotaService) CheckUploadQuota(ctx, userID string) error
func (s *QuotaService) ConsumeUpload(ctx, userID string, bytes int64) error
```

### 验收标准
- [ ] 超限请求返回 429，带 Retry-After。
- [ ] 上传配额用尽返回明确错误，不影响其他接口。

---

## 模块 5：可观测性

### 职责
结构化日志、链路追踪、指标监控，便于线上排查与告警。

### 目录 / 文件
```
pkg/logger/logger.go          # zap：字段化日志、request_id
internal/middleware/logger.go # 请求日志中间件
internal/middleware/trace.go  # request_id 透传
```

### 要点
- 日志字段：`request_id`、`user_id`、`session_id`、`method`、`path`、`status`、`latency`。
- 处理管道每步埋点：上传、转写、AI 分析、回写各阶段耗时与结果。
- 指标：QPS、错误率、转写成功率、AI 置信度分布、队列积压长度。
- 告警：队列积压超阈值、转写失败率飙升、5xx 率异常。

### 验收标准
- [ ] 每条日志含 request_id，可串联一次请求/处理的全链路。
- [ ] 核心指标可被 Prometheus 采集，配置基础告警。

---

## 模块 6：部署

### 职责
用 docker-compose 一键启动依赖与服务，配置 CI/CD。

### docker-compose 服务

已实现于仓库根目录 `docker-compose.yml`：

```yaml
services:
  mysql:     # mysql:8.0，健康检查 mysqladmin ping
  redis:     # redis:7-alpine，健康检查 redis-cli ping
  migrate:   # 一次性执行 pulse-migrate（等待 MySQL 就绪）
  backend:   # 多阶段构建 Go 镜像，暴露 8080，healthcheck 调 /health
```

### 要点
- 多阶段 Docker 构建（golang 编译 → 精简运行镜像，见 `backend/Dockerfile`）。
- 环境变量区分 dev/prod（`.env` 注入，密钥走 secret 管理）。
- 健康检查：容器 healthcheck 调 `/health`（200 健康 / 503 依赖不可用）。
- CI：`gofmt` + `go vet` → 单测（`-race`）→ 真实 MySQL/Redis e2e → 构建镜像（见 `.github/workflows/ci.yml`）。
- CD：推送 `v*` tag 构建镜像到 GHCR 并创建 Release（见 `.github/workflows/release.yml`）。

### 验收标准
- [x] `docker compose up` 可拉起完整环境。
- [x] CI 在 PR 上跑 lint + test，失败阻断合并。
- [x] 发布有回滚方案（镜像按语义版本打 tag，可回滚到上一版本）。

---

## Phase 3 整体验收清单

- [ ] 周报/统计报告可用，大数据量查询有缓存。
- [ ] 音频加密存储，密钥安全。
- [ ] 支持数据删除与导出，注销流程完整。
- [ ] 登录/上传有限流与配额。
- [ ] 日志可链路追踪，指标可采集，告警就位。
- [x] docker-compose 部署 + CI/CD 跑通。

## 上线前安全检查（Checklist）

- [ ] 密码 bcrypt、JWT 密钥强度、token 过期策略。
- [ ] 音频加密、传输 HTTPS、私有空间。
- [ ] 越权访问防护（所有查询带 user_id 过滤）。
- [ ] 敏感日志脱敏（不打印完整转写文本/密钥）。
- [ ] 数据库备份与恢复演练。
- [ ] 第三方（StepFun）密钥不硬编码。
