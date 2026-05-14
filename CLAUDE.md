# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

仓库包含两个独立的音频切换工具：

1. **根目录** — uTools 插件（React 19 + MUI 7 + Electron preload），仅 Windows
2. **`audio-switch/`** — Go + Fyne 独立桌面应用，跨平台（Windows/macOS/Linux）

两者功能相同：快捷切换音频输出设备、音量预设、系统通知、全局热键。

---

## uTools 插件（根目录）

### 构建命令

| 命令 | 说明 |
|------|------|
| `npm install` | 安装依赖 |
| `npm run dev` | 开发模式（webpack watch） |
| `npm run build` | 生产构建到 `dist/` |

无测试框架。

### 架构

- **双层 Webpack 构建**：`bridge/preload.js` → `dist/preload.js`（Node.js/Electron 环境）+ `src/index.js` → `dist/index.js`（浏览器环境）
- **双入口**：`audio-settings`（React 设置页面）和 `audio-quick-switch`（preload 层直接切换，跳过 React）
- **设备枚举**：koffi FFI 调用 Windows Core Audio COM API（<10ms），失败回退 PowerShell
- **设备切换**：运行时编译 C# exe（通过 `csc.exe`），反射加载 `AudioDeviceCmdlets.dll` 的 `PolicyConfigClient`
- **数据持久化**：uTools 内置 PouchDB（`preferred_devices`、`settings`）

### PowerShell 规范

修改 `bridge/preload.js` 中的 PowerShell 命令时**必须**：
- 使用 `-EncodedCommand` + Base64 (UTF-16 LE)，通过 `encodePowerShellScript()` 函数
- 禁止 `-Command` + 字符串拼接（中文编码问题）
- 包含 `PS_PREFIX`（修复 PSModulePath + UTF-8 输出）

依赖模块：**AudioDeviceCmdlets**（设备操作）、**BurntToast**（Toast 通知）

---

## Go 独立应用（`audio-switch/`）

### 构建命令

```bash
cd audio-switch
go build -o audio-switch.exe .          # Windows 编译
GOOS=darwin GOARCH=amd64 go build .     # macOS 交叉编译
GOOS=linux GOARCH=amd64 go build .      # Linux 交叉编译
```

无测试文件。依赖：`fyne.io/fyne/v2`、`github.com/go-ole/go-ole`、`golang.design/x/hotkey`

### 架构

入口 `main.go` → 加载配置 → 创建 Fyne App → 初始化系统托盘 (`ui.TrayApp`) → 注册全局热键 → `a.Run()`

**跨平台接口分层**（通过 `//go:build` 标签切换实现）：

| 接口 | 文件 | Windows 实现 | macOS 实现 | Linux 实现 |
|------|------|-------------|-----------|-----------|
| `audio.Audio` | `internal/audio/audio.go` | 纯 Go COM API (go-wca) | SwitchAudioSource CLI | pactl CLI |
| `notify.Notifier` | `internal/notify/notify.go` | beeep (Win32 Toast) | osascript | notify-send |

**Windows 音频实现的关键设计决策**：

- **设备枚举**（`audio_windows.go`）：使用 `go-wca` 库封装的 `IMMDeviceEnumerator`、`IPropertyStore`、`PROPVARIANT`，通过 `wca.PKEY_Device_FriendlyName` 读取设备友好名称。无需手动 vtable 解引用
- **设备切换**（`audio_windows.go`）：通过 `CoCreateInstance` 直接激活 `IPolicyConfig` COM 接口（CLSID `{870AF99C-171D-4F9E-AF0D-E63DF40C2BC9}`），调用 `SetDefaultEndpoint`（vtable 索引 13）为所有角色（eConsole/eMultimedia/eCommunications）设置默认设备。如失败回退 `IPolicyConfigVista`（vtable 索引 6）。无外部依赖
- **音量控制**（`audio_windows.go`）：切换后通过 `go-wca` 的 `IAudioEndpointVolume::SetMasterVolumeLevelScalar` 设置音量。`SetDeviceVolume` 一步完成切换+音量
- **通知**（`notify_windows.go`）：使用 `beeep` 库发送 Windows Toast 通知，无需 PowerShell 或 BurntToast

**配置**：JSON 文件 `~/.config/audio-switch/config.json`，存储偏好设备（A/B）、热键、通知开关

### 外部依赖

- **Windows**：无外部依赖（纯 Go COM + beeep 通知）
- **macOS**：`SwitchAudioSource`（`brew install switchaudio-osx`）
- **Linux**：`pactl`（PulseAudio/PipeWire 自带）
