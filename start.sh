#!/bin/sh
set -e

# ============================================================
# 供应链溯源系统 - 容器启动脚本
# 配置由 Go 程序从环境变量读取，不再使用 sed 修改 config.yaml
# ============================================================

APP_PID=""

# 优雅关闭：Docker 停止容器时，先关闭 Go 后端再关闭 Nginx
cleanup() {
    echo ""
    echo "============================================"
    echo "  收到停止信号，正在优雅关闭服务..."
    echo "============================================"
    if [ -n "${APP_PID}" ] && kill -0 "${APP_PID}" 2>/dev/null; then
        echo "🛑 正在关闭 Go 后端 (PID: ${APP_PID})..."
        kill -TERM "${APP_PID}" 2>/dev/null
        # 等待最多 10 秒
        for i in $(seq 1 10); do
            if ! kill -0 "${APP_PID}" 2>/dev/null; then
                echo "✅ Go 后端已安全关闭"
                break
            fi
            sleep 1
        done
        # 如果还没关，强制 kill
        if kill -0 "${APP_PID}" 2>/dev/null; then
            echo "⚠️  强制关闭 Go 后端"
            kill -KILL "${APP_PID}" 2>/dev/null || true
        fi
    fi
    echo "✅ 关闭完成"
    exit 0
}
trap cleanup SIGTERM SIGINT SIGQUIT

echo "============================================"
echo "  供应链溯源系统 - 容器启动中..."
echo "============================================"

# 打印关键环境变量（隐藏密码）
echo "📋 配置信息:"
echo "   DB_HOST=${DB_HOST:-localhost}"
echo "   DB_PORT=${DB_PORT:-3306}"
echo "   DB_USER=${DB_USER:-root}"
echo "   DB_NAME=${DB_NAME:-supply_chain}"
echo "   GIN_MODE=${GIN_MODE:-release}"
echo "   APP_PORT=${APP_PORT:-80}"

echo ""
echo "🚀 正在启动 Go 后端服务..."
/app/supply_app &
APP_PID=$!

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
echo "   监听端口: ${APP_PORT:-80}"
echo "   外部访问: http://<服务器IP>:${APP_PORT:-80}"
echo "============================================"
exec nginx -g "daemon off;"
