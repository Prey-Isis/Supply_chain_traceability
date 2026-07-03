#!/bin/sh
set -e

echo "============================================"
echo "  供应链溯源系统 - 容器启动中..."
echo "============================================"

# 根据环境变量动态更新配置文件（Docker 环境）
# 如果设置了 DB_HOST 环境变量，覆盖 config.yaml 中的数据库配置
if [ -n "${DB_HOST}" ]; then
    echo "📝 检测到环境变量，更新数据库配置..."
    
    # 创建临时配置目录（确保存在）
    mkdir -p /app/config
    
    # 使用 sed 替换数据库配置
    sed -i "s/host:.*/host: ${DB_HOST}/" /app/config/config.yaml
    sed -i "s/port:.*/port: ${DB_PORT:-3306}/" /app/config/config.yaml
    sed -i "s/username:.*/username: ${DB_USER:-root}/" /app/config/config.yaml
    sed -i "s/password:.*/password: \"${DB_PASSWORD}\"/" /app/config/config.yaml
    sed -i "s/database:.*/database: ${DB_NAME:-supply_chain}/" /app/config/config.yaml
    
    echo "✅ 配置更新完成: DB_HOST=${DB_HOST}, DB_PORT=${DB_PORT:-3306}"
fi

# 设置 Gin 运行模式
if [ -n "${GIN_MODE}" ]; then
    export GIN_MODE="${GIN_MODE}"
    echo "🏭 运行模式: ${GIN_MODE}"
fi

echo ""
echo "🚀 正在启动 Go 后端服务..."
/app/supply_app &

# 等待 Go 后端就绪（健康检查方式，最多等待30秒）
echo "⏳ 等待 Go 后端就绪..."
MAX_WAIT=30
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo "✅ Go 后端已就绪 (耗时 ${WAITED}s)"
        break
    fi
    sleep 1
    WAITED=$((WAITED + 1))
done

if [ $WAITED -ge $MAX_WAIT ]; then
    echo "⚠️  警告: Go 后端未在 ${MAX_WAIT}s 内就绪，继续启动 Nginx..."
fi

echo ""
echo "🌐 正在启动 Nginx..."
exec nginx -g "daemon off;"
