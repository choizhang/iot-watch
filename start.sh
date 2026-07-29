#!/bin/bash

# 进入脚本所在的目录
cd "$(dirname "$0")"

# 创建隐藏的运行管理与日志目录
mkdir -p .logs

echo "=========================================="
echo "🚀 正在一键启动 IoT 项目 (Docker + 后端 + H5移动端 + 安防大屏)"
echo "=========================================="

# 1. 启动 Docker Compose 中间件
echo "[1/4] 正在启动 Docker 中间件 (MySQL, Redis, InfluxDB)..."
docker compose up -d
if [ $? -ne 0 ]; then
    echo "❌ Docker Compose 启动失败，请检查 Docker 是否在运行。"
    exit 1
fi

# 等待中间件初始化就绪
echo "⌛ 等待数据库初始化就绪 (8秒)..."
sleep 8

# 2. 启动 Go 后端服务
echo "[2/4] 正在后台启动 Go 后端服务..."
cd backend
go mod tidy
go run . > ../.logs/backend.log 2>&1 &
BACKEND_PID=$!
echo $BACKEND_PID > ../.logs/.backend.pid
cd ..

# 3. 启动 Vue C端家属移动端服务
echo "[3/4] 正在后台启动 C端家属移动端服务 (端口 5173)..."
cd frontend
pnpm dev > ../.logs/frontend.log 2>&1 &
FRONTEND_PID=$!
echo $FRONTEND_PID > ../.logs/.frontend.pid
cd ..

# 4. 启动 Vue B端社区指挥大屏服务
echo "[4/4] 正在后台启动 B端社区安防大屏服务 (端口 5174)..."
if [ -d "dashboard" ]; then
    cd dashboard
    pnpm dev > ../.logs/dashboard.log 2>&1 &
    DASHBOARD_PID=$!
    echo $DASHBOARD_PID > ../.logs/.dashboard.pid
    cd ..
fi

echo "=========================================="
echo "✅ 项目启动指令已成功下发！"
echo "=========================================="
echo "📱 C端家属移动端访问地址: http://localhost:5173"
echo "🖥️ B端社区安防大屏访问地址: http://localhost:5174"
echo "🏥 后端健康检查地址: http://localhost:8080/health"
echo "------------------------------------------"
echo "📄 查看后端日志: tail -f .logs/backend.log"
echo "📄 查看移动端日志: tail -f .logs/frontend.log"
echo "📄 查看大屏日志: tail -f .logs/dashboard.log"
echo "🛑 停止项目请运行: ./stop.sh"
echo "=========================================="
