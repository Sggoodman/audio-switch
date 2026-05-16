# Audio Switch

跨平台音频输出设备快速切换工具。通过全局热键在两个预设音频设备之间一键切换，并支持音量预设和系统通知。

## 功能特性

- **快捷切换** — 全局热键在设备 A/B 之间一键切换，无需打开设置
- **音量预设** — 为每个设备设定目标音量，切换时自动应用
- **系统通知** — 切换后弹出 Toast 通知，显示当前活跃设备
- **开机自启** — 可选开机自动启动，最小化到系统托盘
- **跨平台** — 支持 Windows / macOS / Linux

## 支持平台

| 平台 | 设备管理 | 通知 | 备注 |
|------|---------|------|------|
| Windows | Windows Core Audio (COM) | Win32 Toast | 无外部依赖 |
| macOS | SwitchAudioSource CLI | osascript | 需 `brew install switchaudio-osx` |
| Linux | pactl CLI | notify-send | PulseAudio / PipeWire 自带 |

## 安装

### 预编译版本

前往 [Releases](https://github.com/Sggoodman/audio-switch/releases) 下载对应平台的可执行文件。

### 从源码构建

需要 Go 1.21+ 和 CGO 工具链（Fyne 依赖）。

```bash
# 克隆仓库
git clone https://github.com/Sggoodman/audio-switch.git
cd audio-switch

# 构建（Windows 示例）
go build -o audio-switch.exe .

# 或使用构建脚本（含图标资源）
./scripts/build.sh windows
```

交叉编译：

```bash
GOOS=darwin GOARCH=amd64 go build .
GOOS=linux GOARCH=amd64 go build .
```

## 使用

1. 启动后应用最小化到系统托盘
2. 右键托盘图标 → **Settings** 打开偏好设置
3. 选择设备 A 和设备 B，调整音量预设
4. 点击托盘图标或按热键即可切换

## 技术栈

- [Go](https://go.dev/) — 主语言
- [Fyne](https://fyne.io/) — 跨平台 GUI 框架（系统托盘 + 设置窗口）
- [go-wca](https://github.com/moutend/go-wca) — Windows Core Audio API 绑定
- [go-ole](https://github.com/go-ole/go-ole) — COM 接口支持
- [hotkey](https://golang.design/x/hotkey) — 全局热键注册

## 许可证

[MIT](LICENSE)
