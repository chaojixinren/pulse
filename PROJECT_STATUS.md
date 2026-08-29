# Pulse 项目状态

> 本文件为项目整体状态看板，持续更新。**最后更新：2025-08（Phase 3 完成）**

## 🎯 项目概述

Pulse（时笺）是一个智能身份空间系统：通过可穿戴硬件采集日常对话，云端 AI（adk-go + StepFun ASR）自动转写、识别身份并提取待办/承诺，前端按身份生成日报、周报与统计报告。

## 📊 总体进度

| 模块 | 状态 | 说明 |
|------|------|------|
| 后端（Go） | ✅ 三阶段完成 | MVP 闭环 + AI 身份识别 + 生产化，测试/CI/CD/Docker 齐备 |
| 前端（React） | ✅ 三阶段完成 | 身份/设备/时间线/报告/账户，单测/E2E/CI/CD/Docker 齐备 |
| 部署 | ✅ 全栈 compose | MySQL + 迁移 + 后端 + 前端 一键启动 |
| 硬件（ESP32-S3） | ⬜ 未开始 | 技术栈已定，固件/PCB/外壳仅有文档 |

## ✅ 已完成

### 后端（`backend/`，Go 1.26.5 + Gin）

- **Phase 1 MVP**：项目骨架、JWT 认证（bcrypt + refresh token）、音频上传/存储、会话状态机、StepFun STT 转写、身份管理、时间线、日报。
- **Phase 2 AI**：adk-go 身份识别、待办/承诺/笔记提取、设备管理（创建/绑定/解绑/指令，一次性手抄 token）、AI 结果展示。
- **Phase 3 生产化**：音频 AES-256-GCM 加密、账号导出/注销、周报/统计、可观测性（metrics/trace/日志）、限流与优雅关闭。
- 测试：`go test ./...` 全绿（单元 + sqlmock 集成 + httptest API e2e）；真实 MySQL 的 live e2e 在 CI 中跑（`-tags e2e`）。
- CI/CD：`.github/workflows/ci.yml`（lint/test/e2e/docker）+ `release.yml`（推送 GHCR）。

### 前端（`frontend/`，React 18 + TypeScript + Vite）

- **Phase 1 MVP**：认证、身份管理、时间线、日报。
- **Phase 2 设备与 AI**：设备绑定/管理、AI 识别结果与「未识别」徽标。
- **Phase 3 生产化**：周报/统计（自建 SVG 图表）、账户设置（导出/注销）、错误边界、路由懒加载 + 代码分割、深色模式持久化、access token 自动续期。
- 测试：**209** 单元/集成（Vitest）+ **33** E2E（Playwright）全绿。
- CI/CD：`.github/workflows/frontend-ci.yml`（lint/typecheck/单测/E2E/Docker 冒烟）。
- 部署：多阶段 Dockerfile + Nginx（SPA 回退 + `/api` 反向代理）。

### 部署（`docker-compose.yml`）

```bash
docker compose up -d --build
```

拉起 `mysql` → `migrate` → `backend` → `frontend` 四个服务；前端经 Nginx 把 `/api/**` 代理到后端，与浏览器同源，无需额外 CORS。访问 `http://localhost:${FRONTEND_PORT:-5173}`。

## ⬜ 剩余待办

1. **前后端真实联调**：前端 E2E 目前用内存 mock；尚无「浏览器 → 真实前端 → 真实后端 → MySQL」的端到端联调测试。
2. **硬件固件**：ESP32-S3 音频采集/网络通信/云端上传均未实现（仅技术选型）。
3. **运维项**（代码外）：Prometheus 告警规则、数据库备份与恢复演练、生产安全扫描。
4. **移动端**：待定（未来可能 React Native）。

## 🚀 快速开始

```bash
# 全栈（推荐）
docker compose up -d --build

# 或本地开发
cd backend && cp .env.example .env && go run cmd/server/main.go
cd frontend && npm ci && npm run dev
```

## 📁 项目结构

```
Pulse/
├── backend/        # Go 后端（api/service/repository/model/middleware/worker/migrations）
├── frontend/       # React 前端（pages/components/services/contexts/e2e）
├── hardware/       # ESP32-S3（文档）
├── docs/           # 设计文档 + backend/ + frontend/ 分阶段文档
└── docker-compose.yml
```

## 📖 文档索引

- 后端设计：[docs/backend-design.md](docs/backend-design.md)；分阶段：[docs/backend/](docs/backend/)
- 前端设计：[docs/frontend-design.md](docs/frontend-design.md)；分阶段：[docs/frontend/](docs/frontend/)
- 后端 CI/CD：[docs/backend/ci-cd.md](docs/backend/ci-cd.md)
- 前端 CI/CD：[docs/frontend/ci-cd.md](docs/frontend/ci-cd.md)
- 硬件：[hardware/README.md](hardware/README.md)
