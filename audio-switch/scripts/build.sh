#!/bin/bash
# 跨平台编译脚本
# 用法: ./scripts/build.sh [platform]
# platform: windows / darwin / linux / all

set -e

VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="build"
APP_NAME="audio-switch"

# 清理旧构建
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

build() {
    local os=$1
    local arch=$2
    local ext=""

    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi

    local output="${BUILD_DIR}/${APP_NAME}-${os}-${arch}${ext}"

    echo "构建 $os/$arch..."
    CGO_ENABLED=1 GOOS="$os" GOARCH="$arch" go build \
        -ldflags="-s -w -X main.Version=$VERSION" \
        -o "$output" \
        .

    # 复制图标文件
    if [ -f "assets/Icon.png" ]; then
        cp "assets/Icon.png" "${BUILD_DIR}/"
    fi
    if [ -f "assets/Icon.ico" ] && [ "$os" = "windows" ]; then
        cp "assets/Icon.ico" "${BUILD_DIR}/"
    fi

    echo "完成: $output"
}

case "${1:-all}" in
    windows)
        build windows amd64
        ;;
    darwin)
        build darwin amd64
        build darwin arm64
        ;;
    linux)
        build linux amd64
        ;;
    all)
        echo "注意: CGO 交叉编译需要对应的 C 工具链"
        echo "Windows 构建..."
        build windows amd64
        # macOS 和 Linux 需要 CGO（Fyne 依赖），交叉编译需要额外工具链
        # 取消注释以启用：
        # build darwin amd64
        # build linux amd64
        ;;
    *)
        echo "未知平台: $1"
        echo "用法: $0 [windows|darwin|linux|all]"
        exit 1
        ;;
esac

echo ""
echo "构建完成！输出目录: $BUILD_DIR/"
ls -la "$BUILD_DIR/"
