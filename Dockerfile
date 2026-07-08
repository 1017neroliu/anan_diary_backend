# ============================================
# Anan Diary Backend — Multi-stage Docker build
# 依赖以 vendor/ 形式提交，构建可离线复现
# ============================================

# ---- Build stage ----
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 复制全部源码（含 vendor/、go.mod、go.sum）
COPY . .

# 使用 vendor 目录构建，禁用网络下载
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /app/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从 builder 复制编译好的二进制
COPY --from=builder /app/server .
# 复制迁移脚本
COPY --from=builder /app/migrations ./migrations
# 上传目录（运行时挂载 volume 覆盖）
RUN mkdir -p /app/uploads

EXPOSE 8080

# 使用非 root 用户运行
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app
USER appuser

CMD ["./server"]