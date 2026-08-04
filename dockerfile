# ============================================================
# 供应链溯源系统 - Dockerfile（多阶段构建 + 精简镜像 + 秒级部署优化版）
# ============================================================
#
# 【为什么要多阶段构建？】
#   构建 Go/Vue 需要巨大的工具镜像（golang 镜像约 300MB、node 镜像约 1GB），
#   但运行我们的程序根本不需要这些工具，只需要：
#     - Go 编译出的二进制（约 15MB）
#     - Vue 构建出的静态文件（几 MB）
#     - Nginx 运行时（约 23MB）
#   多阶段构建 = 在"工具镜像"里编译，只把"产物"拷到"运行镜像"，
#   最终镜像只包含运行必需的东西，体积从 1GB+ 降到 ~50MB。
#
# 【为什么能秒级部署？】
#   Docker 分层缓存：每条指令是一层，输入没变 → 缓存复用，不重跑。
#   关键编排原则：先 COPY 不常变的（依赖清单），再 COPY 常变的（源码）。
#     - 只改 Go 代码 → 前端两层 + Go 依赖层全部缓存命中，只重编译后端
#     - 只改 Vue 代码 → 只重构建前端，Go 依赖层缓存命中
#   再加上 BuildKit 的 cache mount（编译缓存持久化），增量构建 10 秒内完成。
#
# 【BuildKit 说明】
#   本文件使用了 --mount=type=cache（构建缓存持久化），需要 BuildKit。
#   Docker 23+ / docker compose v2 默认启用，无需额外配置。
#   如果你的 Docker 版本旧，去掉 # syntax 行和 --mount 参数也能构建（只是少了缓存加速）。
# ============================================================

# 声明使用新版 Dockerfile 语法（支持 cache mount 等 BuildKit 特性）
# syntax=docker/dockerfile:1

# ============================================================
# 第一阶段：构建 Vue 前端（工具镜像：node）
# ============================================================
FROM node:22-alpine AS frontend

WORKDIR /web

# 设置 npm 镜像源加速（国内用户）
RUN npm config set registry https://registry.npmmirror.com

# ★ 缓存优化关键点 1：先只复制 package*.json（依赖清单）
#   package.json / package-lock.json 不常变 → 这一层缓存长期有效
COPY supply-chain-frontend/package*.json ./

# 安装依赖（只有 package.json 变化时才会重新执行这步）
# --cache-dir 配合 BuildKit cache mount：npm 下载的包缓存到 /npm-cache
# 下次构建即使需要重装，也能秒级从本地缓存命中，不用重新上网下载
RUN --mount=type=cache,target=/npm-cache \
    npm ci --prefer-offline --cache /npm-cache

# ★ 缓存优化关键点 2：依赖装完，才复制源码
#   源码变化 → 只重跑"构建"，不重跑"装依赖"
#   --link：复用上一层文件系统，COPY 不产生新的数据复制（BuildKit 优化）
COPY --link supply-chain-frontend/ ./

# 构建生产产物到 dist/ 目录
# --emptyOutDir：清空旧输出，避免残留旧文件
RUN npm run build

# ============================================================
# 第二阶段：构建 Go 后端（工具镜像：golang）
# ============================================================
FROM golang:1.26-alpine AS backend

WORKDIR /app

# 设置 Go 代理加速（国内用户）
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

# ★ 缓存优化关键点 3：先只复制依赖清单，下载依赖
#   go.mod/go.sum 不常变 → 这层缓存长期有效
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# ★ 缓存优化关键点 4：源码（常变）放最后 COPY
#   --link：硬链接方式复制，加速构建（BuildKit 优化，不破坏下层缓存）
COPY --link . .

# 编译生产二进制
# 参数说明：
#   -ldflags="-w -s"  : -w 去掉 DWARF 调试信息, -s 去掉符号表 → 二进制体积减半
#   -trimpath        : 去掉编译时的绝对路径信息 → 更小更安全
#   ./cmd/api/       : Go 主入口在 cmd/api 目录（module 名为 main）
# --mount=type=cache：Go 编译缓存持久化到 /root/.cache/go-build
#                     改一行代码重编译时，只重编受影响的包，秒级完成
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-w -s" -o supply_app ./cmd/api/

# ============================================================
# 第三阶段：最终运行镜像（基础镜像：nginx，含 Go 应用 + Nginx）
# ============================================================
# nginx:alpine 约 23MB，自带 nginx 可执行文件，alpine 极简 Linux
FROM nginx:alpine

# 安装运行需要的工具
#   curl    - 健康检查（Docker HEALTHCHECK 和启动脚本都要用）
#   tini    - 轻量 init 进程，负责把 SIGTERM 信号转发给子进程，
#             保证 docker stop 时 Nginx 和 Go 应用能优雅退出
RUN apk add --no-cache curl tini

# 设置工作目录（Go 程序从这里加载 config/config.yaml）
WORKDIR /app

# ---- 从第一阶段拷贝前端静态文件 ----
# 只拷 dist 目录（Vue 构建产物），不含 node_modules 等垃圾
COPY --from=frontend /web/dist /usr/share/nginx/html

# ---- 从第二阶段拷贝 Go 二进制 ----
COPY --from=backend /app/supply_app /app/supply_app
# 赋予执行权限
RUN chmod +x /app/supply_app

# ---- 拷贝配置文件 ----
# ⚠️ 重要：Go 程序启动时读 config/config.yaml，容器里必须存在！
# 之前的教训：漏拷这行导致容器启动即崩溃
COPY --from=backend /app/config/config.yaml /app/config/config.yaml

# ---- 拷贝 Nginx 配置 ----
# nginx.conf 里配置了：静态文件服务 + /api 反代到 localhost:8080（Go 应用）
COPY nginx.conf /etc/nginx/conf.d/default.conf

# ---- 拷贝启动脚本 ----
# start.sh 负责：先启动 Go 后端 → 等待健康检查通过 → 再启动 Nginx
COPY start.sh /start.sh
RUN chmod +x /start.sh

# 声明容器对外端口（仅文档用途，真正暴露端口在 docker-compose 里）
EXPOSE 80

# ---- 容器健康检查 ----
# Docker 每 30 秒执行一次 curl，返回非 0 表示容器不健康
# 直接探测 Go 后端（8080），因为 nginx 反代了 /api 到它
# 配合 docker-compose 的 restart: always，故障自动重启
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# ---- 启动入口 ----
# tini 作为 PID 1（init 进程），正确处理信号
#  /start.sh 先起 Go 后端，再起 Nginx（阻塞式，保持前台运行）
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/start.sh"]
