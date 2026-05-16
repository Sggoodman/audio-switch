#!/bin/bash
# 构建脚本
# 用法: ./scripts/build.sh [windows|darwin|linux|all|debug]
# debug: 带终端窗口 + 调试符号，用于开发调试

set -e

VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="build"
APP_NAME="audio-switch"

# 清理旧构建
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# 生成 Windows 资源文件（嵌入图标和版本信息）
if command -v go-winres &>/dev/null && [ -f "winres/winres.json" ]; then
    echo "生成 Windows 资源文件..."
    go-winres make
fi

build() {
    local os=$1
    local arch=$2
    local ext=""

    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi

    local output="${BUILD_DIR}/${APP_NAME}${ext}"

    echo "构建 $os/$arch..."

    # Windows 隐藏终端窗口
    local gui_flag=""
    if [ "$os" = "windows" ]; then
        gui_flag="-H windowsgui"
    fi

    CGO_ENABLED=1 GOOS="$os" GOARCH="$arch" go build \
        -ldflags="-s -w ${gui_flag} -X main.Version=$VERSION" \
        -o "$output" \
        .

    # 复制图标文件
    if [ -f "assets/Icon.png" ]; then
        cp "assets/Icon.png" "${BUILD_DIR}/"
    fi

    echo "完成: $output"
}

build_debug() {
    local ext=""
    if [ "$OSTYPE" = "msys" ] || [ "$OSTYPE" = "win32" ]; then
        ext=".exe"
    fi

    local output="${BUILD_DIR}/${APP_NAME}-debug${ext}"

    echo "构建调试版本（带终端 + 调试符号）..."

    go build \
        -o "$output" \
        .

    echo "完成: $output"
}

case "${1:-windows}" in
    debug)
        build_debug
        ;;
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
        echo "未知参数: $1"
        echo "用法: $0 [windows|darwin|linux|all|debug]"
        exit 1
        ;;
esac

echo ""
echo "构建完成！输出目录: $BUILD_DIR/"
ls -la "$BUILD_DIR/"
