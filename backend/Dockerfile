# syntax=docker/dockerfile:1

# ---------- 构建阶段 ----------
FROM golang:1.22-alpine AS build
WORKDIR /src

# 先复制依赖清单，利用 Docker 层缓存加速重复构建
COPY go.mod go.sum ./
RUN go mod download

# 复制源码，构建 server 与 migrate 两个二进制
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pulse-server ./cmd/server && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pulse-migrate ./cmd/migrate

# ---------- 运行阶段 ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl && adduser -D -u 10001 pulse

WORKDIR /app
COPY --from=build /out/pulse-server /usr/local/bin/pulse-server
COPY --from=build /out/pulse-migrate /usr/local/bin/pulse-migrate
COPY migrations ./migrations

USER pulse
EXPOSE 8080

# /health 返回 200（健康）或 503（MySQL/Redis 不可用），curl -f 据此判定就绪
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=5 CMD curl -fsS http://127.0.0.1:8080/health > /dev/null || exit 1

ENTRYPOINT ["pulse-server"]
