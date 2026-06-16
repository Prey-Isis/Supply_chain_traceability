#!/bin/sh
# 后台启动 Go 应用
/app/supply_app &

# 前台启动 Nginx（必须前台，否则容器会退出）
nginx -g "daemon off;"