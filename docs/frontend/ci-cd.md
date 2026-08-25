# Pulse 前端 CI/CD

> 前端基于 Vite + TypeScript 构建，产物为纯静态文件，由 Nginx 托管。
> 本文档描述本地命令、GitHub Actions 流水线与 Docker 部署。

## 流水线概览

| 阶段 | 命令 | 说明 |
|------|------|------|
| 依赖安装 | `npm ci` | 按 lockfile 安装 |
| 代码检查 | `npm run lint` | ESLint |
| 类型检查 | `npm run type-check` | `tsc --noEmit` |
| 单元/组件/集成测试 | `npm test` | Vitest + React Testing Library |
| 覆盖率 | `npm run test:coverage` | Vitest + v8 覆盖率 |
| E2E | `npm run test:e2e` | Playwright（`e2e/` 目录） |
| 构建 | `npm run build` | `tsc && vite build` |
| Docker 构建 | `docker build -t pulse-frontend .` | 多阶段构建（node 构建 + nginx 托管），CI 含 SPA 回退与 `/api` 代理冒烟测试 |

> 说明：`frontend/package.json` 已含 `dev/build/preview/lint/type-check/test/test:coverage/test:e2e` 脚本；测试框架（Vitest + RTL + Playwright）已接入。

## 本地命令

```bash
cd frontend
npm ci              # 安装依赖
npm run lint        # 代码检查
npm run type-check  # 类型检查
npm run build       # 生产构建（tsc + vite build）
npm run preview     # 本地预览构建产物
```

## GitHub Actions 工作流

```yaml
# .github/workflows/frontend-ci.yml
name: Frontend CI

on:
  push:
    branches: [main, master]
    paths: ['frontend/**', '.github/workflows/frontend-ci.yml']
  pull_request:
    paths: ['frontend/**', '.github/workflows/frontend-ci.yml']

defaults:
  run:
    working-directory: frontend

jobs:
  lint-typecheck-unit-build:
    name: Lint, Typecheck, Unit & Build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: npm ci
      - run: npm run lint
      - run: npm run type-check
      - run: npm run test:coverage
      - run: npm run build

  e2e:
    name: E2E (Playwright)
    runs-on: ubuntu-latest
    needs: lint-typecheck-unit-build
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: npm ci
      - run: npx playwright install --with-deps chromium
      - run: npm run test:e2e
```

## Docker 部署

### Dockerfile（多阶段）

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### nginx.conf

```nginx
server {
    listen 80;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # 单页应用：所有前端路由回退到 index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 反向代理后端 API（变量 + resolver 延迟解析，后端未就绪时 nginx 仍可启动）
    location /api/ {
        resolver 127.0.0.11 valid=30s ipv6=off;
        set $backend http://backend:8080;
        proxy_pass $backend;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

### 构建与运行

```bash
docker build --build-arg VITE_API_BASE_URL=/api/v1 -t pulse-frontend:local frontend
docker run -p 5173:80 pulse-frontend:local
```

## 可覆盖的环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | 后端 API 基地址；生产构建时通过 `--build-arg` 注入 |

## 发布

前端为静态产物，可随主仓库的 Docker Compose 或独立静态托管（Nginx / CDN / 对象存储）发布。发布前需保证：

- `npm run build` 通过（含类型检查）。
- 环境变量 `VITE_API_BASE_URL` 指向正确的后端地址。
- 后端 CORS（`ALLOWED_ORIGINS`）已放行前端域名。
