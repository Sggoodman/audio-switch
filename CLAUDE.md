# Audio Switch - Claude 开发指南

## 技术栈

GUI/托盘/热键/通知/自启基于 [mygo](https://github.com/egoist/mygo)（纯 Go 桌面框架，native UI 无 webview），
设置窗口为 mygo native UI（`ui` 包，Elm 架构：view 是状态的纯函数，每帧重建）。
音频设备控制为自研 `internal/audio`（macOS 走 cgo CoreAudio，Windows 走 go-wca，Linux 走 pactl）。

## 编译规则

**重要**: 请始终使用 `scripts/build.sh` 脚本进行编译，不要直接使用 `go build` 命令。

### 编译命令

```bash
# 调试版本（go build 直出，带调试符号，用于开发调试）
bash scripts/build.sh debug

# macOS 应用包（mygo build：.app + .dmg，universal 双架构）
bash scripts/build.sh app

# Windows 安装包（mygo build + NSIS：可选安装路径的 setup.exe）
bash scripts/build.sh windows-app

# 全平台安装包（darwin universal + windows amd64 + linux amd64）
bash scripts/build.sh all
```

### 安装包说明

- `build.sh app` 产出 `dist/darwin-universal/`：
  - `Audio Switch.app` — universal（amd64+arm64）应用包，Accessory 策略（纯托盘应用：无 Dock 图标）
  - `audio-switch-<version>.dmg` — 磁盘映像安装包
- `build.sh windows-app` 产出 `dist/windows-amd64/`：
  - `audio-switch-setup-<version>-amd64.exe` — NSIS 安装器（需 `brew install makensis`）
  - `Audio Switch.exe` — 裸可执行文件
- Windows/Linux 代码全为纯 Go，交叉编译**不再需要 mingw**（CGO_ENABLED=0）
- macOS 的 CoreAudio 是 cgo，`build.sh` 自动透传 `CGO_ENABLED=1`（仅限 darwin 目标，勿在交叉编译 windows 时开 cgo）
- 安装包版本号从 git tag 自动同步到 `mygo.json` 的 `version` 字段，**必须先打 tag 再打包**
- 开发热重载：`go tool mygo dev`（改 .go 文件自动重编译重启）
- native UI 代码检查：`CGO_ENABLED=1 go tool mygo vet`（检查 view 构建生命周期问题）

### 输出目录

- 调试版本: `build/audio-switch`（或 Windows 下 `.exe`）
- 打包产物: `dist/<os>-<arch>/`

### 重新编译前

如果遇到编译错误或需要重新编译，确保先关闭正在运行的 Audio Switch 进程。

## 日志规范

项目使用 `go.uber.org/zap` 日志库，通过 `internal/logger` 包封装提供统一的日志接口。

### 基本用法

```go
import "audio-switch/internal/logger"

logger.Info("Module", "消息描述", "key1", value1, "key2", value2)
logger.Warn("Module", "非致命错误描述", "error", err)
logger.Error("Module", "致命错误描述", "error", err)
logger.Debug("Module", "调试信息", "detail", something)
```

### 日志格式

```
2026-05-20T10:00:00.000+0800  INFO  main.go:34  消息描述  {"module": "Module", "key1": "value1"}
```

### 模块命名

| 模块名 | 使用位置 |
|--------|----------|
| `Main` | main.go |
| `Settings` | ui/settings.go |
| `Tray` | ui/tray.go |
| `Hotkey` | ui/hotkey.go、ui/tray.go 中热键相关逻辑 |
| `Autostart` | main.go 中开机自启相关逻辑 |
| `Notify` | ui/tray.go 中通知相关逻辑 |
| `Audio` | internal/audio/ |

### 日志级别

| 级别 | 使用场景 |
|------|----------|
| `Info` | 程序启动/关闭、配置加载/保存成功、设备切换成功、热键注册 |
| `Warn` | 非致命错误（配置加载失败但使用默认值、通知推送失败） |
| `Error` | 致命错误 |
| `Debug` | UI 滑块值变化、初始化细节等调试信息 |

### 规则

1. **禁止使用标准库 `log`**（main.go 中 log.Fatal 除外），统一使用 `logger` 包
2. **每条日志必须有模块名**（第一个参数）
3. **使用结构化字段**传递数据，不要用 fmt.Sprintf 拼接到消息中
4. **中文消息描述**，简洁准确
5. **错误日志传递原始 error**：`logger.Warn("Module", "操作失败", "error", err)`

## 项目结构

- `ui/` - mygo 托盘 + native UI 设置窗口（tray.go 托盘与业务逻辑、settings.go 设置窗口、hotkey.go 热键转换）
- `internal/audio/` - 音频设备接口（平台相关，macOS 为 cgo）
- `internal/config/` - 配置管理
- `internal/logger/` - 日志系统（zap 封装）
- `scripts/` - 构建脚本（build.sh、make-tray-icon.go 托盘图标生成）
- `assets/` - 图标资源（Icon.png 应用图标、tray-template.png 托盘 template 图标）
- `resources/` - mygo 打包资源（icon.png 1024×1024 打包图标）
- `mygo.json` - mygo 应用配置（名称/标识符/版本）

### 托盘/热键/通知要点

- 热键格式 `Ctrl+Alt+S` 与用户配置兼容；mygo GlobalShortcut 直接接受该格式
- mygo 的 UI/托盘 API 须在主线程调用，后台 goroutine 用 `mygo.RunOnMain(fn)` 切回
- 设置窗口 view 中用 `c.KeysDown()` + `c.Modifiers()` 实现热键录制（窗口需有焦点）
- macOS 通知、开机自启（SetOpenAtLogin）仅对打包后的应用生效，裸二进制运行会记 Warn（属预期）
- 纯托盘应用保活：`mygo.App.OnWindowAllClosed(func(){})` 空监听即可
- 托盘图标须为黑色透明 PNG（template 模式），由 `scripts/make-tray-icon.go` 从 assets/Icon.png 生成

## Release 流程

### 发布新版本

1. 提交所有改动后，获取远端最新标签并递增版本号：
   ```bash
   git fetch --tags
   git tag --sort=-v:refname | head -5   # 查看当前最新标签
   ```
2. 创建新标签（安装包版本号取自 tag，**必须先打 tag 再打包**）：
   ```bash
   git tag vX.Y.Z
   ```
3. 打包各平台安装包：
   ```bash
   bash scripts/build.sh app          # macOS: dist/darwin-universal/*.dmg
   bash scripts/build.sh windows-app  # Windows: dist/windows-amd64/audio-switch-setup-*.exe（需 makensis）
   ```
4. 推送代码与标签：
   ```bash
   git push origin main vX.Y.Z
   ```
5. 使用 gh CLI 创建 GitHub Release，上传安装包（而非裸二进制）：
   ```bash
   gh release create vX.Y.Z \
     "dist/darwin-universal/audio-switch-X.Y.Z.dmg" \
     "dist/windows-amd64/audio-switch-setup-X.Y.Z-amd64.exe" \
     --title "vX.Y.Z" \
     --notes "release notes here"
   ```

注意事项

- GitHub Token 通过环境变量 GITHUB_TOKEN 自动获取，无需额外配置
- 版本号格式遵循语义化版本：v主版本.次版本.修订号
- build.sh 会把 tag 版本自动写入 mygo.json；HEAD 不在 tag 上时用 mygo.json 现有版本号（文件名不带 dev 后缀，注意不要误发）
- 交叉编译 Windows 时保持 CGO_ENABLED=0（build.sh 已处理）
