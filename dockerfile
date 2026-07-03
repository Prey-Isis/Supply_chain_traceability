# ============================================================
# 供应链溯源系统 - Dockerfile
# 多阶段构建：前端构建 → Go 后端构建 → Nginx + App 运行
# ============================================================

# 第一阶段：构建 Vue 前端
FROM node:18-alpine AS frontend

WORKDIR /web

# 设置 npm 镜像源加速（国内用户）
RUN npm config set registry https://registry.npmmirror.com

# 复制依赖文件（利用 Docker 缓存）
COPY supply-chain-frontend/package*.json ./

# 安装依赖（只有 package.json 变化时才重新安装）
RUN npm ci

# 复制源码并构建
COPY supply-chain-frontend/ ./
RUN npm run build

# 第二阶段：构建 Go 后端
FROM golang:1.26-alpine AS backend

WORKDIR /app

# 安装必要的构建工具
RUN apk add --no-cache gcc musl-dev

# 设置 Go 代理加速
ENV GOPROXY=https://goproxy.cn,direct

# 先复制依赖文件（利用 Docker 缓存）
COPY go.mod go.sum ./
RUN go mod download

# 再复制源码
COPY . .

# 构建（注意入口在 cmd/api/ 目录）
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o supply_app ./cmd/api/

# 第三阶段：最终运行镜像
FROM nginx:alpine

# 安装必要工具（用于健康检查）
RUN apk add --no-cache curl

# 复制 Go 应用
COPY --from=backend /app/supply_app /app/supply_app
RUN chmod +x /app/supply_app

# 复制配置文件 ⚠️ 之前漏掉了，导致启动失败！
COPY --from=backend /app/config/config.yaml /app/config/config.yaml

# 复制前端静态文件
COPY --from=frontend /web/dist /usr/share/nginx/html

# 复制 Nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 复制启动脚本
COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD curl -f http://localhost:80/ || exit 1

CMD ["/start.sh"]
