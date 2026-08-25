# Phase 3：生产化开发文档

> **目标**：让系统达到可上线标准——补全报告、加密存储、合规能力、可观测性与部署。
> **完成标志**：通过安全检查，可灰度上线。

## 模块概览

本阶段模块相互独立，可并行推进：

| 模块 | 依赖 | 说明 |
|------|------|------|
| 1 报告增强 | Phase1 时间线 + Phase2 AI | 周报、统计图表 |
| 2 加密存储 | Phase1 音频上传 | 音频 AES-256 加密 |
| 3 数据删除/导出 | 各模块 | 合规（GDPR/个保法） |
| 4 可观测性 | 骨架 | 日志、trace、指标 |
| 5 部署 | 全部 | docker-compose、CI/CD |

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
- 图表数据按天聚合（`DATE_FORMAT` 分组），避免实时扫全表。
- 数据量进一步增大时可再引入定时任务预计算日报/周报。

### 验收标准
- [x] 周报统计正确，含趋势与身份分布。
- [x] 大时间范围查询按天聚合（`DATE_FORMAT` 分组），避免实时扫全表。

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
func EncryptAudio(data []byte, key []byte) ([]byte, error)

// 读取时解密
func DecryptAudio(data []byte, key []byte) ([]byte, error)
```

### 加密方案
- 算法 AES-256-GCM，密钥 32 字节。
- 密钥管理：生产环境用 KMS/环境变量注入，**密钥不入库、不硬编码**。
- 每个用户或每个文件可用独立数据加密密钥（DEK），主密钥（KEK）管理 DEK（进阶，可先全局单密钥）。

### 存储侧
- 加密后的密文写入 `audio_sessions.audio_data`（LONGBLOB），不再以明文落库。
- 转写 worker 读取后先解密再提交 StepFun。

### 验收标准
- [x] audio_data 中存储的是密文，直接读取无法播放（AES-256-GCM，`nonce || 密文`）。
- [x] 后端解密后可正常 STT 转写（worker 转写前解密）。
- [x] 密钥不落库、不硬编码（`AUDIO_ENCRYPTION_KEY` 环境变量注入，base64 解码；留空关闭加密，生产必须配置）。

## 模块 3：数据删除/导出

### 职责
合规（GDPR/个保法）：用户可导出全部个人数据，也可注销账户。

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/account/export | 导出个人数据（用户/身份/设备/会话） |
| DELETE | /api/v1/account | 注销账户（软删除 + 吊销全部 token） |

### 实现
- 导出聚合 `AccountExport{user, identities, devices, sessions}`；敏感字段（password_hash / deleted_at / device_token_hash / audio_data）已由模型 JSON tag 排除。
- 注销采用软删除（置 `deleted_at`），已删除用户无法登录；同时吊销全部 refresh token。

### 验收标准
- [x] 支持数据导出，敏感字段不泄露。
- [x] 支持账户注销，删除后无法登录，refresh token 全部吊销。

---

## 模块 4：可观测性

### 职责
结构化日志、链路追踪、指标监控，便于线上排查与告警。

### 目录 / 文件
```
pkg/logger/logger.go          # zap：字段化日志、request_id
internal/middleware/logger.go # 请求日志中间件
internal/middleware/trace.go  # request_id 透传
internal/middleware/metrics.go # Prometheus 指标
```

### 要点
- 日志字段：`request_id`、`user_id`、`session_id`、`method`、`path`、`status`、`latency`。
- 处理管道每步埋点：上传、转写、AI 分析、回写各阶段耗时与结果。
- 指标：QPS、错误率、转写成功率、AI 置信度分布、队列积压长度。
- 告警：队列积压超阈值、转写失败率飙升、5xx 率异常。

### 验收标准
- [x] 每条请求日志含 request_id（`X-Request-ID` 生成/透传），可串联请求链路。
- [x] 核心指标可被 Prometheus 采集（`/metrics` 暴露 `pulse_http_requests_total` / `pulse_http_request_duration_seconds`）。
- [ ] 告警规则（队列积压、转写失败率、5xx）随部署在 Prometheus 侧配置。

---

## 模块 5：部署

### 职责
用 docker-compose 一键启动依赖与服务，配置 CI/CD。

### docker-compose 服务

已实现于仓库根目录 `docker-compose.yml`：

```yaml
services:
  mysql:     # mysql:8.0，健康检查 mysqladmin ping
  migrate:   # 一次性执行 pulse-migrate（等待 MySQL 就绪）
  backend:   # 多阶段构建 Go 镜像，暴露 8080，healthcheck 调 /health
```

### 要点
- 多阶段 Docker 构建（golang 编译 → 精简运行镜像，见 `backend/Dockerfile`）。
- 环境变量区分 dev/prod（`.env` 注入，密钥走 secret 管理）。
- 健康检查：容器 healthcheck 调 `/health`（200 健康 / 503 依赖不可用）。
- CI：`gofmt` + `go vet` → 单测（`-race`）→ 真实 MySQL e2e → 构建镜像（见 `.github/workflows/ci.yml`）。
- CD：推送 `v*` tag 构建镜像到 GHCR 并创建 Release（见 `.github/workflows/release.yml`）。

### 验收标准
- [x] `docker compose up` 可拉起完整环境。
- [x] CI 在 PR 上跑 lint + test，失败阻断合并。
- [x] 发布有回滚方案（镜像按语义版本打 tag，可回滚到上一版本）。

---

## Phase 3 整体验收清单

- [x] 周报/统计报告可用，大数据量查询按天聚合。
- [x] 音频加密存储，密钥安全。
- [x] 支持数据删除与导出，注销流程完整。
- [x] 日志可链路追踪，指标可采集（告警规则随 Prometheus 部署配置）。
- [x] docker-compose 部署 + CI/CD 跑通。

## 上线前安全检查（Checklist）

- [x] 密码 bcrypt、JWT 密钥强度、token 过期策略（Phase 1）。
- [x] 音频加密（本阶段）；传输 HTTPS 由部署侧 TLS 终结保证。
- [x] 越权访问防护（所有查询带 user_id 过滤）。
- [x] 敏感日志脱敏（不打印完整转写文本/密钥）。
- [ ] 数据库备份与恢复演练（运维任务，代码外）。
- [x] 第三方（StepFun）密钥不硬编码（环境变量注入）。
