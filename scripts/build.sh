#!/bin/bash
# 构建脚本（mygo 打包链）
# 用法: ./scripts/build.sh [debug|app|windows-app|linux|all]
#   debug:       go build 直出二进制（开发调试，隐藏终端 + 调试符号）
#   app:         mygo build 打 macOS 应用包（.app + .dmg，universal 双架构）
#   windows-app: mygo build 打 Windows 安装包（纯 Go 交叉编译，无需 mingw）
#   linux:       mygo build 打 Linux 安装包（.deb + tar.gz）
#   all:         mygo build 打全平台安装包（darwin universal + windows + linux）
#
# 说明：
# - macOS 的 CoreAudio 实现是 cgo，mygo build 需透传 CGO_ENABLED=1（仅本机目标）
# - Windows/Linux 代码全为纯 Go，交叉编译 CGO_ENABLED=0 即可
# - 安装包版本号取 mygo.json 的 version 字段，由 git tag 自动同步（见 sync_version）

set -e

# 统一切到仓库根目录，保证从任意位置执行都正确
cd "$(cd "$(dirname "$0")/.." && pwd)"

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
# 安装包版本：HEAD 正好命中 tag 时用 tag（正式发布），否则 dev
PKG_VERSION=${PKG_VERSION:-$(git describe --tags --exact-match 2>/dev/null || echo "dev")}
OUT_DIR="dist"
APP_NAME="audio-switch"
MYGO="go tool mygo"

# sync_version 把安装包版本写入 mygo.json（mygo build 从中取版本号）
sync_version() {
    local ver="$1"
    # dev 构建（未命中 tag）退回 mygo.json 中的基础版本
    if [ "$ver" = "dev" ] || [ -z "$ver" ]; then
        return
    fi
    ver="${ver#v}" # 去掉 v 前缀
    sed -i '' "s/\"version\": *\"[^\"]*\"/\"version\": \"$ver\"/" mygo.json
    echo "mygo.json 版本已同步为 $ver"
}

build_debug() {
    local ext=""
    if [ "$OSTYPE" = "msys" ] || [ "$OSTYPE" = "win32" ]; then
        ext=".exe"
    fi

    local output="build/${APP_NAME}${ext}"
    mkdir -p build

    echo "构建调试版本..."
    CGO_ENABLED=1 go build \
        -ldflags="-X ui.Version=$VERSION" \
        -o "$output" \
        .
    echo "完成: $output"
}

build_app() {
    sync_version "$PKG_VERSION"
    echo "打包 macOS 应用（universal）..."
    CGO_ENABLED=1 $MYGO build -platform darwin/universal
    echo "完成，产物在 $OUT_DIR/darwin-universal/"
}

build_windows_app() {
    sync_version "$PKG_VERSION"
    echo "打包 Windows 安装包（amd64）..."
    CGO_ENABLED=0 $MYGO build -platform windows/amd64
    echo "完成，产物在 $OUT_DIR/windows-amd64/"
}

build_linux() {
    sync_version "$PKG_VERSION"
    echo "打包 Linux 安装包（amd64：.deb + tar.gz）..."
    CGO_ENABLED=0 $MYGO build -platform linux/amd64
    echo "完成，产物在 $OUT_DIR/linux-amd64/"
}

build_all() {
    sync_version "$PKG_VERSION"
    echo "打包全平台..."
    # macOS（cgo，本机编译）与其余平台（纯 Go）分开调用
    CGO_ENABLED=1 $MYGO build -platform darwin/universal
    CGO_ENABLED=0 $MYGO build -platform windows/amd64,linux/amd64
    echo "完成，产物在 $OUT_DIR/ 下各平台目录"
}

case "${1:-}" in
    debug)
        build_debug
        ;;
    app)
        build_app
        ;;
    windows-app)
        build_windows_app
        ;;
    linux)
        build_linux
        ;;
    all)
        build_all
        ;;
    *)
        echo "用法: $0 [debug|app|windows-app|linux|all]"
        echo "  debug:       开发调试二进制（build/ 目录）"
        echo "  app:         macOS .app + .dmg（universal）"
        echo "  windows-app: Windows 安装包（纯 Go，无需 mingw）"
        echo "  linux:       Linux 安装包（.deb + tar.gz）"
        echo "  all:         全平台安装包"
        exit 1
        ;;
esac
