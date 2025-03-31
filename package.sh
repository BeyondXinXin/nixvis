#!/bin/bash

echo "开始构建 NixVis..."
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

echo "清理旧文件..."
rm -f nixvis

# 获取版本信息
BUILD_TIME=$(date "+%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse --short=7 HEAD 2>/dev/null || echo "unknown")

echo "版本信息:"
echo " - 构建时间: ${BUILD_TIME}"
echo " - Git提交: ${GIT_COMMIT}"

echo "编译主程序..."
go build -ldflags="-s -w -X 'github.com/beyondxinxin/nixvis/internal/util.BuildTime=${BUILD_TIME}' -X 'github.com/beyondxinxin/nixvis/internal/util.GitCommit=${GIT_COMMIT}'" -o nixvis ./cmd/nixvis/main.go

if [ $? -eq 0 ]; then
    echo "构建成功! 可执行文件: nixvis"

    # 显示文件大小
    FILE_SIZE=$(du -h nixvis | cut -f1)
    echo "文件大小: ${FILE_SIZE}"

    # 检查是否正确嵌入了资源
    echo "验证资源嵌入..."
    strings nixvis | grep -q "<!DOCTYPE html>" && echo "✓ HTML资源已嵌入" || echo "✗ HTML资源可能未正确嵌入"
    strings nixvis | grep -q ".css" && echo "✓ CSS资源已嵌入" || echo "✗ CSS资源可能未正确嵌入"
    strings nixvis | grep -q ".js" && echo "✓ JS资源已嵌入" || echo "✗ JS资源可能未正确嵌入"

    echo "构建完成，可执行文件已准备就绪"
else
    echo "构建失败!"
    exit 1
fi
