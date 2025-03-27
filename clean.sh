#!/bin/bash
# filepath: clean_nixvis.sh

echo "开始清理nixvis服务..."

# 检查nixvis是否在运行
if pgrep -f "nixvis" >/dev/null; then
    echo "发现nixvis正在运行，准备停止..."

    # 获取nixvis进程ID并终止
    pkill -f "nixvis"

    # 等待进程完全终止
    sleep 2

    # 检查是否成功终止
    if pgrep -f "nixvis" >/dev/null; then
        echo "nixvis未能正常终止，强制停止..."
        pkill -9 -f "nixvis"
    fi

    echo "nixvis已停止运行"
else
    echo "nixvis当前未运行"
fi

# 检查是否有进程占用8088端口
if lsof -i :8088 >/dev/null 2>&1; then
    echo "发现端口8088被占用，准备停止相关进程..."

    # 获取占用8088端口的进程ID
    PID=$(lsof -t -i :8088)
    echo "占用端口的进程ID: $PID"

    # 终止进程
    kill $PID

    # 等待进程完全终止
    sleep 2

    # 检查是否成功终止
    if lsof -i :8088 >/dev/null 2>&1; then
        echo "进程未能正常终止，强制停止..."
        kill -9 $PID
    fi

    echo "端口8088已释放"
else
    echo "端口8088当前未被占用"
fi

echo "开始清理 nixvis_data 目录..."
rm -rf ./nixvis_data
echo "清理工作完成"
