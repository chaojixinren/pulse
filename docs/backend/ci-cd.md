# 后端 CI/CD

Pulse 后端持续集成（CI）与持续交付（CD）说明。

## 概览

| 阶段 | 触发条件 | 内容 |
|------|----------|------|
| CI lint | push / PR（`backend/**`） | `gofmt` 检查 + `go vet ./...` + `go vet -tags e2e ./...` |
| CI test | push / PR | `go test ./... -race`，上传覆盖率产物（含 Phase 2 单测/集成测） |
| CI e2e | push / PR | 启动 MySQL，执行迁移后按矩阵跑 Phase 1 / Phase 2 真实基础设施 e2e |
| CI docker | push / PR | 构建多阶段 Docker 镜像（不推送） |
| CD release | 推送 `v*` tag | 构建镜像推送到 GHCR，并创建 GitHub Release |

工作流文件：

- `.github/workflows/ci.yml` — 持续集成
- `.github/workflows/release.yml` — 持续交付

## 本地常用命令（Makefile）

在仓库根目录执行：

```bash
make build        # 编译 backend
make test         # 单元 + 集成测试（sqlmock，无需外部依赖）
make test-race    # 带竞态检测
make test-e2e     # 真实 MySQL e2e（需 TEST_DATABASE_DSN）
make vet          # go vet（含 e2e build tag）
make fmt-check    # 检查 gofmt
make lint         # gofmt + go vet（如安装 golangci-lint 则一并执行）
make migrate      # 运行数据库迁移
make docker-build # 构建后端镜像
make docker-up    # 启动 docker compose 全栈
make docker-down  # 停止
```

## Docker Compose 一键部署

```bash
docker compose up -d --build
```

会依次拉起：

1. `mysql`（mysql:8.0，库名 `pulse`）
2. `migrate`（一次性执行 `pulse-migrate` 迁移，等待 MySQL 就绪）
3. `backend`（等待迁移完成后启动，暴露 `8080`）
4. `frontend`（等待 backend 健康后启动，Nginx 托管静态产物，暴露 `FRONTEND_PORT`，默认 `5173`）

启动后健康检查：`curl -fsS http://localhost:8080/health`，返回 200 表示 MySQL 可用。前端通过 `http://localhost:${FRONTEND_PORT:-5173}` 访问，`/api/**` 由前端 Nginx 反向代理到 `backend:8080`，因此浏览器请求与后端同源，无需额外 CORS 配置。

## 可覆盖的环境变量（docker compose）

compose 会读取同目录的 `.env`（不存在则使用默认值）：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MYSQL_ROOT_PASSWORD` | `rootpass` | MySQL root 密码 |
| `MYSQL_DATABASE` | `pulse` | 数据库名 |
| `MYSQL_USER` / `MYSQL_PASSWORD` | `pulse` / `pulsepass` | 应用数据库账号 |
| `DATABASE_DSN` | `pulse:pulsepass@tcp(mysql:3306)/pulse?...` | 后端 MySQL DSN |
| `JWT_SECRET` | `dev-secret-change-me` | JWT 密钥（生产务必覆盖） |
| `STEPFUN_API_KEY` | 空 | StepFun 语音识别密钥 |
| `GIN_MODE` | `release` | Gin 运行模式 |
| `FRONTEND_PORT` | `5173` | 前端 Nginx 映射到宿主机端口 |

> 生产环境请务必覆盖 `JWT_SECRET`、`MYSQL_*` 密码与 `STEPFUN_API_KEY`，密钥建议走 secret 管理。

## CD：发布镜像到 GHCR

推送形如 `v1.0.0` 的 tag 即触发：

```bash
git tag v1.0.0
git push origin v1.0.0
```

镜像推送到 `ghcr.io/<owner>/pulse`，同时生成 `v1.0.0`、`1.0.0`、`1.0`、`1` 等 tag，并在 GitHub 创建 Release。
