#!/bin/sh
set -e

echo "Starting Go backend..."
/app/supply_app &

echo "Waiting for Go backend to start..."
sleep 3

echo "Starting Nginx..."
nginx -g "daemon off;"