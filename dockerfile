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

# 设置 Go 代理加速
ENV GOPROXY=https://goproxy.cn,direct

# 先复制依赖文件（利用 Docker 缓存）
COPY go.mod go.sum ./
RUN go mod download

# 再复制源码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o supply_app .

# 第三阶段：最终运行镜像
FROM nginx:alpine

# 安装 curl（可选）
RUN apk add --no-cache curl

# 复制 Go 应用
COPY --from=backend /app/supply_app /app/supply_app
RUN chmod +x /app/supply_app

# 复制前端静态文件
COPY --from=frontend /web/dist /usr/share/nginx/html

# 复制 Nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 复制启动脚本
COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 80
CMD ["/start.sh"]