#!/bin/bash

# Windows 版本编译脚本
# 用途：为 Windows 平台编译 game-control 程序

set -e  # 遇到错误立即退出

echo "=========================================="
echo "Windows 版本编译脚本"
echo "=========================================="

# 设置编译参数
GOOS=windows
GOARCH=amd64
OUTPUT_NAME="game-control.exe"
SOURCE_PATH="./cmd/game-control/main.go"

echo ""
echo "编译配置："
echo "  目标平台: $GOOS/$GOARCH"
echo "  输出文件: $OUTPUT_NAME"
echo "  源文件: $SOURCE_PATH"
echo ""

# 编译
echo "开始编译..."
GOOS=$GOOS GOARCH=$GOARCH go build -o "$OUTPUT_NAME" "$SOURCE_PATH"

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "编译成功！"
    echo "=========================================="
    echo "输出文件: $OUTPUT_NAME"
    
    # 显示文件信息
    if command -v file &> /dev/null; then
        echo "文件类型: $(file $OUTPUT_NAME)"
    fi
    
    if command -v ls &> /dev/null; then
        echo "文件大小: $(ls -lh $OUTPUT_NAME | awk '{print $5}')"
    fi
    
    echo ""
    echo "使用方法："
    echo "  game-control.exe start [config]  启动游戏时间控制守护进程"
    echo "  game-control.exe status [config] 查询当前游戏时间状态"
    echo "  game-control.exe reset [config]  手动重置每日游戏时间配额"
    echo "  game-control.exe validate [config] 验证配置文件"
    echo "  game-control.exe help           显示帮助信息"
    echo ""
else
    echo ""
    echo "=========================================="
    echo "编译失败！"
    echo "=========================================="
    exit 1
fi