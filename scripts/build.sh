#!/bin/bash
# 构建脚本
# 用法: ./scripts/build.sh [windows|darwin|linux|all|debug|app|windows-app]
# debug:     带终端窗口 + 调试符号，用于开发调试
# app:       macOS 应用包（.app + 带拖拽安装布局的 .dmg，universal）
# windows-app: Windows NSIS 安装包（需 makensis；CGO 交叉编译需 mingw）

set -e

# 统一切到仓库根目录，保证从任意位置执行都正确
cd "$(cd "$(dirname "$0")/.." && pwd)"

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
# 安装包版本：HEAD 正好命中 tag 时用 tag（正式发布），否则 dev
PKG_VERSION=${PKG_VERSION:-$(git describe --tags --exact-match 2>/dev/null || echo "dev")}
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

    # Windows 保持原名（Release 流程依赖 audio-switch.exe），
    # 其他平台带 os-arch 后缀，避免多架构构建互相覆盖
    local suffix=""
    if [ "$os" != "windows" ]; then
        suffix="-${os}-${arch}"
    fi

    local output="${BUILD_DIR}/${APP_NAME}${suffix}${ext}"

    echo "构建 $os/$arch..."

    # Windows 隐藏终端窗口
    local gui_flag=""
    if [ "$os" = "windows" ]; then
        gui_flag="-H windowsgui"
    fi

    CGO_ENABLED=1 GOOS="$os" GOARCH="$arch" go build \
        -ldflags="-s -w ${gui_flag} -X ui.Version=$VERSION" \
        -o "$output" \
        .

    # 复制图标文件
    if [ -f "assets/Icon.png" ]; then
        cp "assets/Icon.png" "${BUILD_DIR}/"
    fi
    # 生成 .ico 文件（用于通知图标）
    if [ -f "assets/Icon.png" ] && [ -f "scripts/convert-ico.go" ]; then
        echo "生成 Icon.ico..."
        go run scripts/convert-ico.go "${BUILD_DIR}/Icon.png" "${BUILD_DIR}/Icon.ico"
    fi

    echo "完成: $output"
}

build_debug() {
    local ext=""
    if [ "$OSTYPE" = "msys" ] || [ "$OSTYPE" = "win32" ]; then
        ext=".exe"
    fi

    local output="${BUILD_DIR}/${APP_NAME}${ext}"

    echo "构建调试版本（隐藏终端 + 调试符号）..."

    CGO_ENABLED=1 go build \
        -ldflags="-H windowsgui" \
        -o "$output" \
        .

    # 复制图标文件
    if [ -f "assets/Icon.png" ]; then
        cp "assets/Icon.png" "${BUILD_DIR}/"
    fi
    # 生成 .ico 文件（用于通知图标）
    if [ -f "assets/Icon.png" ] && [ -f "scripts/convert-ico.go" ]; then
        echo "生成 Icon.ico..."
        go run scripts/convert-ico.go "${BUILD_DIR}/Icon.png" "${BUILD_DIR}/Icon.ico"
    fi

    echo "完成: $output"
}

# package_app 组装 .app bundle（universal）并打 DMG 安装包
package_app() {
    local app_dir="${BUILD_DIR}/Audio Switch.app"
    local exe="${app_dir}/Contents/MacOS/audio-switch"
    local version="${PKG_VERSION#v}"
    # CFBundleVersion 要求纯数字版本号，git describe 的附加信息剥掉
    if ! echo "$version" | grep -qE '^[0-9]+(\.[0-9]+)*$'; then
        version="1.0.0"
    fi

    echo "组装 Audio Switch.app (universal)..."
    rm -rf "$app_dir"
    mkdir -p "${app_dir}/Contents/MacOS" "${app_dir}/Contents/Resources"

    # 合并双架构为 universal 二进制
    lipo -create \
        "${BUILD_DIR}/${APP_NAME}-darwin-amd64" \
        "${BUILD_DIR}/${APP_NAME}-darwin-arm64" \
        -output "$exe"

    # 生成 .icns 图标（sips 缩放 + iconutil 打包，无需外部依赖）
    local iconset
    iconset="$(mktemp -d)/Icon.iconset"
    mkdir -p "$iconset"
    local size
    for size in 16 32 128 256 512; do
        sips -z "$size" "$size" assets/Icon.png --out "${iconset}/icon_${size}x${size}.png" >/dev/null
        local double=$((size * 2))
        sips -z "$double" "$double" assets/Icon.png --out "${iconset}/icon_${size}x${size}@2x.png" >/dev/null
    done
    iconutil -c icns "$iconset" -o "${app_dir}/Contents/Resources/Icon.icns"
    cp assets/Icon.png "${app_dir}/Contents/Resources/Icon.png"

    # Info.plist：LSUIElement=true 为纯托盘应用（无 Dock 图标、无终端窗口）
    cat > "${app_dir}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>Audio Switch</string>
    <key>CFBundleDisplayName</key>
    <string>Audio Switch</string>
    <key>CFBundleIdentifier</key>
    <string>com.audioswitch.app</string>
    <key>CFBundleExecutable</key>
    <string>audio-switch</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleIconFile</key>
    <string>Icon.icns</string>
    <key>CFBundleShortVersionString</key>
    <string>${version}</string>
    <key>CFBundleVersion</key>
    <string>${version}</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
PLIST

    # ad-hoc 签名：未签名二进制拷贝到其他 Mac 会报「已损坏」，签名后右键打开即可运行
    codesign --force --sign - "$app_dir" >/dev/null 2>&1 && echo "ad-hoc 签名完成"

    # 打 DMG：背景提示图 + 图标布局（app 在左、Applications 在右）
    echo "生成 DMG 安装包（含拖拽安装布局）..."
    local tmp bg staging rw_dmg final_dmg
    tmp="$(mktemp -d)"
    bg="${tmp}/background.png"
    rw_dmg="${tmp}/rw.dmg"

    clang -w -framework Cocoa scripts/dmg_background.m -o "${tmp}/dmg_background"
    "${tmp}/dmg_background" "$bg"

    staging="${tmp}/staging"
    mkdir -p "${staging}/.background"
    cp -R "$app_dir" "$staging/"
    cp "$bg" "${staging}/.background/background.png"
    ln -s /Applications "${staging}/Applications"

    # 先打可读写镜像，挂载后由 Finder 写入窗口布局（.DS_Store），再压缩为最终 DMG
    hdiutil create -volname "Audio Switch" -format UDRW -ov \
        -srcfolder "$staging" "$rw_dmg" >/dev/null
    # 清理可能残留的同名挂载点（上次失败中断时可能未卸载）
    local v
    for v in "/Volumes/Audio Switch" "/Volumes/Audio Switch 1" "/Volumes/Audio Switch 2" "/Volumes/Audio Switch 3"; do
        [ -d "$v" ] && hdiutil detach -force -quiet "$v" 2>/dev/null || true
    done

    hdiutil attach -readwrite -noverify -noautoopen -quiet "$rw_dmg"
    sleep 3
    if ! osascript <<'ASEOF' >/dev/null
tell application "Finder"
    tell disk "Audio Switch"
        open
        delay 2
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set bounds of container window to {200, 120, 860, 520}
        set the arrangement of the icon view options of container window to not arranged
        set icon size of the icon view options of container window to 128
        set position of item "Audio Switch.app" of container window to {150, 180}
        set position of item "Applications" of container window to {460, 180}
        set background picture of the icon view options of container window to POSIX file "/Volumes/Audio Switch/.background/background.png"
        close
        open
        update without registering applications
        delay 3
    end tell
end tell
ASEOF
    then
        hdiutil detach -force -quiet "/Volumes/Audio Switch" 2>/dev/null || true
        echo "错误: 设置 DMG 窗口布局失败" >&2
        exit 1
    fi
    local i
    for i in 1 2 3; do
        hdiutil detach -quiet "/Volumes/Audio Switch" 2>/dev/null && break
        sleep 2
        hdiutil detach -force -quiet "/Volumes/Audio Switch" 2>/dev/null && break
    done

    final_dmg="${BUILD_DIR}/Audio-Switch-${PKG_VERSION}-darwin-universal.dmg"
    rm -f "$final_dmg"
    hdiutil convert "$rw_dmg" -format UDZO -o "$final_dmg" >/dev/null
    rm -rf "$tmp"

    # 清理中间产物（双架构二进制已用 lipo 并入 .app，图标已在 bundle Resources 内）
    rm -f "${BUILD_DIR}/${APP_NAME}-darwin-amd64" \
          "${BUILD_DIR}/${APP_NAME}-darwin-arm64" \
          "${BUILD_DIR}/Icon.ico" "${BUILD_DIR}/Icon.png"

    echo "完成: ${final_dmg}"
    echo "完成: ${app_dir}"
}

# package_windows 用 NSIS 生成 Windows 安装包（可选安装路径 + 开始菜单/桌面快捷方式 + 卸载程序）
package_windows() {
    if ! command -v makensis >/dev/null 2>&1; then
        echo "错误: 生成 Windows 安装包需要 NSIS (makensis)"
        echo "安装: brew install makensis   (macOS)"
        echo "      Windows 从 https://nsis.sourceforge.io 下载安装"
        exit 1
    fi
    echo "生成 Windows 安装包..."
    makensis -V2 \
        -DVERSION="${PKG_VERSION#v}" \
        -DVERNAME="${PKG_VERSION}" \
        scripts/installer.nsi
    # 清理中间产物（已打入安装包）
    rm -f "${BUILD_DIR}/${APP_NAME}.exe" "${BUILD_DIR}/Icon.ico" "${BUILD_DIR}/Icon.png"
    echo "完成: ${BUILD_DIR}/Audio-Switch-${PKG_VERSION}-setup.exe"
}

case "${1:-windows}" in
    debug)
        build_debug
        ;;
    app)
        build darwin amd64
        build darwin arm64
        package_app
        ;;
    windows-app)
        build windows amd64
        package_windows
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
