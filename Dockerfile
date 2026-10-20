FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o agenthub ./cmd/agenthub

FROM alpine:latest

# 创建非 root 用户，生产安全最佳实践
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# 复制编译好的二进制
COPY --from=builder /app/agenthub .

# 复制运行时需要的配置文件和迁移脚本
COPY configs/ ./configs/

# 切换到非 root 用户
ENV AGENTHUB_ENV=production

USER appuser

EXPOSE 8080

CMD ["./agenthub", "serve"]