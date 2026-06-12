# 第一阶段：构建 Vue 前端
FROM node:18-alpine AS frontend

WORKDIR /web
# 注意：目录名改成 supply-chain-frontend
COPY supply-chain-frontend/package*.json ./
RUN npm ci --only=production
COPY supply-chain-frontend/ ./
RUN npm run build

# 第二阶段：构建 Go 后端
FROM golang:1.21-alpine AS backend

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o supply_app .

# 第三阶段：最终运行镜像
FROM nginx:alpine

COPY --from=backend /app/supply_app /app/supply_app
# 确认你的 Vue build 输出目录（通常是 dist）
COPY --from=frontend /web/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 80
CMD ["/start.sh"]