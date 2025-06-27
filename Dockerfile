# 第一阶段：构建阶段
FROM golang:1.23-alpine AS builder
# 安装 git
RUN apk add --no-cache git

COPY ./ /build
# 设置工作目录并构建
WORKDIR /build
RUN go mod tidy &&  go build  -ldflags="-s -w" -gcflags="-l" -trimpath -o deadpool cmd/deadpool/deadpool.go  # 输出小写名称

# 第二阶段：运行阶段
FROM alpine:latest
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add dnsmasq && \
    echo "listen-address=127.0.0.11" > /etc/dnsmasq.conf && \
    echo "no-resolv" >> /etc/dnsmasq.conf && \
    echo "server=8.8.8.8" >> /etc/dnsmasq.conf && \
    rm -rf /var/cache/apk/*
# 安装二进制文件到 /usr/local/bin（全局可执行）
COPY --from=builder /build/deadpool /usr/local/bin/


# 声明工作目录（可选，影响后续 COPY/ADD 的基准路径）
WORKDIR /app

# 暴露端口
EXPOSE 1080

# 直接使用命令名（因 /usr/local/bin 已在 PATH 中）
ENTRYPOINT ["sh", "-c", "dnsmasq --no-daemon & deadpool"]