# 第一阶段：构建阶段
FROM golang:1.23-alpine AS builder
# 安装 git
RUN apk add --no-cache git

# 克隆仓库
RUN git clone https://github.com/wjlin0/deadpool.git /build

# 设置工作目录并构建
WORKDIR /build
RUN go mod tidy &&  go  build -o deadpool  # 输出小写名称

# 第二阶段：运行阶段
FROM alpine:latest

# 安装二进制文件到 /usr/local/bin（全局可执行）
COPY --from=builder /build/deadpool /usr/local/bin/


# 声明工作目录（可选，影响后续 COPY/ADD 的基准路径）
WORKDIR /app

# 暴露端口
EXPOSE 1080

# 直接使用命令名（因 /usr/local/bin 已在 PATH 中）