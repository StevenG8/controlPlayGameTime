#!/usr/bin/env bash

# Windows 版本编译脚本
# 用途：为 Windows 平台编译 game-control 程序并输出可分发目录

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOOS_TARGET="windows"
GOARCH_TARGET="amd64"
OUTPUT_NAME="game-control.exe"
DIST_DIR="${ROOT_DIR}/dist/${GOOS_TARGET}-${GOARCH_TARGET}"
OUTPUT_PATH="${DIST_DIR}/${OUTPUT_NAME}"
SOURCE_PATH="./cmd/game-control"

echo "=========================================="
echo "Windows 版本编译脚本"
echo "=========================================="
echo ""
echo "编译配置："
echo "  目标平台: ${GOOS_TARGET}/${GOARCH_TARGET}"
echo "  输出目录: ${DIST_DIR}"
echo "  输出文件: ${OUTPUT_PATH}"
echo "  构建入口: ${SOURCE_PATH}"
echo ""

mkdir -p "${DIST_DIR}"

echo "开始编译..."
(
  cd "${ROOT_DIR}"
  GOOS="${GOOS_TARGET}" GOARCH="${GOARCH_TARGET}" go build -o "${OUTPUT_PATH}" "${SOURCE_PATH}"
)

# 打包运行所需附加文件
cp -f "${ROOT_DIR}/config.yaml.tmpl" "${DIST_DIR}/config.yaml"
cp -f "${ROOT_DIR}/README.md" "${DIST_DIR}/README.md"
cp -f "${ROOT_DIR}/scripts/windows/start-background.bat" "${DIST_DIR}/start-background.bat"
cp -f "${ROOT_DIR}/scripts/windows/add-autostart.bat" "${DIST_DIR}/add-autostart.bat"
cp -f "${ROOT_DIR}/scripts/windows/remove-autostart.bat" "${DIST_DIR}/remove-autostart.bat"

echo ""
echo "=========================================="
echo "编译成功！"
echo "=========================================="
echo "输出文件: ${OUTPUT_PATH}"

if command -v file >/dev/null 2>&1; then
  echo "文件类型: $(file "${OUTPUT_PATH}")"
fi

if command -v ls >/dev/null 2>&1; then
  echo "文件大小: $(ls -lh "${OUTPUT_PATH}" | awk '{print $5}')"
fi

echo ""
echo "分发目录内容："
ls -lh "${DIST_DIR}"

echo ""
echo "使用方法："
echo "  game-control.exe start [config]"
echo "  add-autostart.bat"
echo "  remove-autostart.bat"
echo "  game-control.exe status [config]"
echo "  game-control.exe validate [config]"
echo "  game-control.exe help"
echo ""
