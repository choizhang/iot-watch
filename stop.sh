#!/bin/bash

# 进入脚本所在的目录
cd "$(dirname "$0")"

echo "=========================================="
echo "🛑 正在一键停止 IoT 项目所有服务..."
echo "=========================================="

kill_port() {
    local port=$1
    local pids=$(lsof -t -i:$port)
    if [ -n "$pids" ]; then
        for pid in $pids; do
            kill -9 $pid 2>/dev/null
        done
    fi
}

# 1. 停止 C端移动端服务
if [ -f .logs/.frontend.pid ] || [ -f .frontend.pid ]; then
    FPID=$(cat .logs/.frontend.pid 2>/dev/null || cat .frontend.pid 2>/dev/null)
    echo "⬇️ 正在停止 C端移动端 (PID: $FPID)..."
    kill $FPID 2>/dev/null
    kill_port 5173
    rm -f .logs/.frontend.pid .frontend.pid
else
    echo "⚠️ 未找到移动端 PID 记录，清理端口 5173..."
    kill_port 5173
fi

# 2. 停止 B端指挥大屏服务
if [ -f .logs/.dashboard.pid ] || [ -f .dashboard.pid ]; then
    DPID=$(cat .logs/.dashboard.pid 2>/dev/null || cat .dashboard.pid 2>/dev/null)
    echo "⬇️ 正在停止 B端指挥大屏 (PID: $DPID)..."
    kill $DPID 2>/dev/null
    kill_port 5174
    rm -f .logs/.dashboard.pid .dashboard.pid
else
    echo "⚠️ 未找到大屏 PID 记录，清理端口 5174..."
    kill_port 5174
fi

# 3. 停止后端服务
if [ -f .logs/.backend.pid ] || [ -f .backend.pid ]; then
    BPID=$(cat .logs/.backend.pid 2>/dev/null || cat .backend.pid 2>/dev/null)
    echo "⬇️ 正在停止 Go 后端 (PID: $BPID)..."
    kill $BPID 2>/dev/null
    kill_port 8080
    kill_port 5007
    rm -f .logs/.backend.pid .backend.pid
else
    echo "⚠️ 未找到后端 PID 记录，清理端口 8080/5007..."
    kill_port 8080
    kill_port 5007
fi

# 4. 停止 Docker 容器
echo "⬇️ 正在关闭 Docker 容器..."
docker compose down

# 清理根目录残留的 loose .pid 和 .log 文件
rm -f *.log .*.pid

echo "=========================================="
echo "✅ 所有服务已成功停止，临时 PID 和日志已自动清理！"
echo "=========================================="
