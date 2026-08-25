# Phase 3 逐项验收报告

> 验收对象：`docs/backend/phase-3-production.md`
> 验收日期：本次会话
> 结论：**全部代码级验收项通过**；本次验收补齐了 CI 覆盖、部署配置与若干缺失测试，并记录了 2 项代码外/残余风险。
>
> **后续变更**：按需求移除 Redis 与「限流配额」模块——删除 `internal/middleware/ratelimit.go`、`internal/config/redis.go`、`internal/service/report_cache_test.go`；报告统计缓存（Redis）改为直查数据库；健康检查仅依赖 MySQL；`docker-compose.yml` 与 CI 亦去除 Redis。下方正文已同步为移除后的状态。

## 一、逐项验收结果

### 模块 1：报告增强

| 验收标准 | 结果 | 证据 |
|---------|------|------|
| 周报统计正确，含趋势与身份分布 | ✅ | `internal/service/report.go` `Weekly`/`aggregate`/`identityStats`/`DailyTrend`；测试 `report_weekly_test.go` |
| 大时间范围查询按天聚合（`DATE_FORMAT` 分组），避免实时扫全表 | ✅ | `report.go` `Stats`/`aggregate` 按天聚合；`report_test.go`/`report_weekly_test.go` 覆盖 |

### 模块 2：加密存储

| 验收标准 | 结果 | 证据 |
|---------|------|------|
| audio_data 中存储的是密文（AES-256-GCM，nonce‖密文） | ✅ | `service/audio.go` `EncryptAudio`；单测 `audio_crypto_test.go`；**新增** live e2e `TestLiveE2EPhase3Encryption` |
| 后端解密后可正常 STT 转写（worker 转写前解密） | ✅ | `worker/audio_processor.go` `processOne` 解密后提交 STT；`TestAudioProcessorDecryptsBeforeTranscribe` |
| 密钥不落库、不硬编码（`AUDIO_ENCRYPTION_KEY` 环境变量注入，base64 解码；留空关闭加密） | ✅ | `config/config.go` `decodeEncryptionKey`（`config_test.go` 覆盖）；`.env.example` 已注释说明 |

### 模块 3：数据删除/导出

| 验收标准 | 结果 | 证据 |
|---------|------|------|
| 支持数据导出，敏感字段不泄露 | ✅ | `service/account.go` `Export`；模型 `json:"-"` 排除 `password_hash`/`deleted_at`/`device_token_hash`/`audio_data`；`account_test.go` + **新增** `TestAccountExportJSONExcludesSensitiveFields` |
| 支持账户注销，删除后无法登录，refresh token 全部吊销 | ✅ | `account.go` `Delete`（软删除 + `RevokeAllForUser`）；`GetByEmail`/`GetByID` 带 `deleted_at IS NULL`；`account_test.go`、live e2e `TestLiveE2EPhase3Flow`（注销后登录 401） |

### 模块 4：可观测性

| 验收标准 | 结果 | 证据 |
|---------|------|------|
| 每条请求日志含 request_id（`X-Request-ID` 生成/透传） | ✅ | `middleware/trace.go` + `logger.go`；`trace_test.go` |
| 核心指标可被 Prometheus 采集（`/metrics` 暴露 `pulse_http_requests_total`/`pulse_http_request_duration_seconds`） | ✅ | `middleware/metrics.go` + `router.go` 注册 `/metrics`；`metrics_test.go` |
| 告警规则（队列积压、转写失败率、5xx） | ⬜（代码外） | 由 Prometheus 侧告警配置负责，本仓库未含 Prometheus 部署/规则文件 |

### 模块 5：部署

| 验收标准 | 结果 | 证据 |
|---------|------|------|
| `docker compose up` 可拉起完整环境 | ✅ | `docker-compose.yml`（mysql/migrate/backend + healthcheck） |
| CI 在 PR 上跑 lint + test，失败阻断合并 | ✅ | `.github/workflows/ci.yml`（gofmt+vet → test `-race` → e2e → docker build） |
| 发布有回滚方案（镜像按语义版本打 tag） | ✅ | `.github/workflows/release.yml`（`type=semver` 打 `v*` 标签推 GHCR） |

## 二、本次验收发现并修复的缺口

1. **CI e2e 矩阵缺少 Phase 3**：`.github/workflows/ci.yml` 的 e2e 矩阵原本只跑 `TestLiveE2EFullFlow`（Phase 1）与 `TestLiveE2EPhase2Devices`（Phase 2），Phase 3 的 live e2e 已存在但未被 CI 执行。已新增 `phase3` 套件（匹配 `TestLiveE2EPhase3*`）。
2. **docker-compose 未注入加密配置**：`docker-compose.yml` 的 backend 服务未透传 `AUDIO_ENCRYPTION_KEY`。已补上（默认空关闭加密，生产用 `.env` 注入）。
3. **敏感字段不泄露仅有 password_hash 断言**：补了覆盖 `device_token_hash`/`audio_data`/`deleted_at` 的单测。
4. **周报/统计/导出/注销缺 API 层集成测试**：补 `internal/api/e2e_phase3_test.go`。
5. **音频加密缺真实库落库断言（密文 at-rest）**：补 live e2e `TestLiveE2EPhase3Encryption`。

## 三、本次新增测试清单

**单元测试**
- `internal/service/account_test.go` → `TestAccountExportJSONExcludesSensitiveFields`

**集成测试（API 层，sqlmock 全路由）**
- `internal/api/e2e_phase3_test.go` → `TestE2EWeeklyReport` / `TestE2EStatsReport` / `TestE2EAccountExport` / `TestE2EAccountDelete` / `TestE2EAccountExportUnauthorized`

**E2E（真实 MySQL，`-tags e2e`）**
- `test/e2e_live_phase3_test.go` → `TestLiveE2EPhase3Encryption`（新增；`TestLiveE2EPhase3Flow` 已存在，本次纳入 CI）

## 四、残余风险 / 建议（不阻断上线判定）

1. **注销后 access token 在过期前仍有效**：`Auth` 中间件仅校验 JWT 签名（无状态），不校验用户是否仍存在/未删除；注销后登录与 refresh 已被正确阻断（`deleted_at IS NULL` 过滤 + refresh token 吊销），但已签发的 access token 在其 TTL 内仍可访问受保护接口。如需“立即失效”，建议在 `Auth` 中增加一次用户存在性校验，或缩短 access token TTL。
2. **告警规则与数据库备份演练**：均属代码外（Prometheus 侧配置 / 运维任务），仍保持文档中的未勾选状态。
3. 本地未能运行 live e2e（Docker daemon 未启动、无本地 MySQL）；该组测试已通过 `-tags e2e` 编译校验，并在 CI 中随 `phase3` 套件执行。
