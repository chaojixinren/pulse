SHELL := /bin/bash
GO ?= go
BACKEND_DIR := backend
FRONTEND_DIR := frontend

.PHONY: help build test test-race test-e2e vet fmt fmt-check lint lint-backend lint-frontend \
        type-check migrate clean docker-build docker-up docker-down \
        install-backend install-frontend install-all setup

help: ## 列出常用命令
	@echo "可用命令："
	@echo "  make install-all    安装所有依赖"
	@echo "  make setup          完整环境配置"
	@echo ""
	@echo "后端命令："
	@echo "  make build          编译 backend 二进制"
	@echo "  make test           运行单元/集成测试"
	@echo "  make test-race      带竞态检测运行测试"
	@echo "  make test-e2e       运行真实 MySQL/Redis e2e"
	@echo "  make vet            go vet"
	@echo "  make fmt            格式化 Go 代码"
	@echo "  make fmt-check      检查 gofmt"
	@echo "  make lint-backend   lint 后端代码"
	@echo "  make migrate        运行数据库迁移"
	@echo ""
	@echo "前端命令："
	@echo "  make lint-frontend  lint 前端代码"
	@echo "  make type-check     TypeScript 类型检查"
	@echo ""
	@echo "Docker 命令："
	@echo "  make docker-build   构建后端镜像"
	@echo "  make docker-up      启动 docker compose 全栈"
	@echo "  make docker-down    停止 docker compose"
	@echo ""
	@echo "通用："
	@echo "  make lint           检查所有代码"
	@echo "  make clean          清理构建产物"

install-all: install-backend install-frontend ## 安装所有依赖
	@echo "✅ 所有依赖安装完成"

install-backend: ## 安装 Go 依赖
	cd $(BACKEND_DIR) && $(GO) mod download
	@echo "✅ Go 依赖安装完成"

install-frontend: ## 安装前端依赖
	cd $(FRONTEND_DIR) && npm install
	@echo "✅ 前端依赖安装完成"

setup: install-all ## 完整环境配置
	@echo "🔧 配置 Git hooks..."
	npm run prepare
	@echo "✅ 环境配置完成！"

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

fmt: ## 格式化 Go 代码
	cd $(BACKEND_DIR) && $(GO) fmt ./...
	@echo "✅ Go 代码格式化完成"

fmt-check: ## 检查 gofmt
	cd $(BACKEND_DIR) && test -z "$$(gofmt -l .)" || (echo "以下文件未格式化:" && gofmt -l . && exit 1)

lint-backend: fmt-check vet ## lint 后端
	@if command -v golangci-lint >/dev/null 2>&1; then cd $(BACKEND_DIR) && golangci-lint run; else echo "golangci-lint 未安装，跳过（已执行 gofmt + go vet）"; fi

lint-frontend: ## lint 前端
	cd $(FRONTEND_DIR) && npm run lint

type-check: ## TypeScript 类型检查
	cd $(FRONTEND_DIR) && npm run type-check

lint: lint-backend lint-frontend type-check ## 检查所有代码
	@echo "✅ 所有代码检查通过"

migrate: ## 运行数据库迁移
	cd $(BACKEND_DIR) && $(GO) run ./cmd/migrate

docker-build: ## 构建镜像
	docker build -t pulse-backend:local $(BACKEND_DIR)

docker-up: ## 启动全栈
	docker compose up -d --build

docker-down: ## 停止
	docker compose down

clean: ## 清理
	cd $(BACKEND_DIR) && rm -f coverage.out
	cd $(FRONTEND_DIR) && rm -rf dist coverage
	@echo "✅ 清理完成"
