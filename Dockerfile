# 构建阶段
FROM golang:1.23-alpine AS builder

# 安装必要的构建工具
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.Version=$(git describe --tags 2>/dev/null || echo 'dev') \
    -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
    -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
    -o mcp-mysql ./cmd

# 运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/mcp-mysql .
COPY --from=builder /app/env.example .

# 复制时区数据
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 设置权限
RUN chown -R appuser:appuser /app && \
    chmod +x /app/mcp-mysql

# 切换到非root用户
USER appuser

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 暴露端口（如果需要）
# EXPOSE 8080

# 设置环境变量默认值
ENV MYSQL_HOST=localhost
ENV MYSQL_PORT=3306
ENV MYSQL_USER=root
ENV MYSQL_PASSWORD=
ENV MYSQL_DATABASE=test
ENV MYSQL_MAX_CONNS=10
ENV MYSQL_IDLE_CONNS=5
ENV MYSQL_CONN_LIFETIME_MINUTES=30
ENV TZ=UTC

# 启动应用
ENTRYPOINT ["./mcp-mysql"]
