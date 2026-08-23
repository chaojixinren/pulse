SHELL := /bin/bash
GO ?= go
BACKEND_DIR := backend

.PHONY: help build test test-race test-e2e vet fmt-check lint migrate clean docker-build docker-up docker-down

help: ## 列出常用命令
	@echo "可用命令："
	@echo "  make build         编译 backend 二进制"
	@echo "  make test          运行单元/集成测试（sqlmock，无需外部依赖）"
	@echo "  make test-race     带竞态检测运行测试"
	@echo "  make test-e2e      运行真实 MySQL/Redis e2e（需 TEST_DATABASE_DSN / TEST_REDIS_URL）"
	@echo "  make vet           go vet（含 e2e build tag）"
	@echo "  make fmt-check     检查 gofmt"
	@echo "  make lint          gofmt + go vet（如安装 golangci-lint 则一并执行）"
	@echo "  make migrate       运行数据库迁移"
	@echo "  make docker-build  构建后端镜像"
	@echo "  make docker-up     启动 docker compose 全栈"
	@echo "  make docker-down   停止 docker compose"

build: ## 编译 backend
	cd $(BACKEND_DIR) && $(GO) build ./cmd/...

test: ## 单元 + 集成测试
	cd $(BACKEND_DIR) && $(GO) test ./... -count=1

test-race: ## 竞态检测测试
	cd $(BACKEND_DIR) && $(GO) test ./... -race -count=1

test-e2e: ## 真实基础设施 e2e
	cd $(BACKEND_DIR) && $(GO) test -tags e2e -run TestLiveE2EFullFlow ./test/ -v -count=1

vet: ## go vet
	cd $(BACKEND_DIR) && $(GO) vet ./... && $(GO) vet -tags e2e ./...

fmt-check: ## 检查 gofmt
	cd $(BACKEND_DIR) && test -z "$$(gofmt -l .)" || (echo "以下文件未格式化:" && gofmt -l . && exit 1)

lint: fmt-check vet ## lint
	@if command -v golangci-lint >/dev/null 2>&1; then cd $(BACKEND_DIR) && golangci-lint run; else echo "golangci-lint 未安装，跳过（已执行 gofmt + go vet）"; fi

migrate: ## 迁移
	cd $(BACKEND_DIR) && $(GO) run ./cmd/migrate

clean: ## 清理
	cd $(BACKEND_DIR) && rm -f coverage.out

docker-build: ## 构建镜像
	docker build -t pulse-backend:local $(BACKEND_DIR)

docker-up: ## 启动全栈
	docker compose up -d --build

docker-down: ## 停止
	docker compose down
