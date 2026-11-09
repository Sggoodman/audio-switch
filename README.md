# Audio Switch

跨平台音频输出设备快速切换工具。通过全局热键在两个预设音频设备之间一键切换，并支持音量预设和系统通知。

## 功能特性

- **快捷切换** — 全局热键在设备 A/B 之间一键切换，无需打开设置
- **音量预设** — 为每个设备设定目标音量，切换时自动应用
- **系统通知** — 切换后弹出系统通知，显示当前活跃设备
- **开机自启** — 可选开机自动启动，最小化到系统托盘
- **轻量常驻** — 基于 [mygo](https://github.com/egoist/mygo) 原生 UI，无 webview；常驻物理内存实测约 19MB（同类 Fyne 实现约 94MB）
- **跨平台** — 支持 Windows / macOS / Linux

## 支持平台

| 平台 | 设备管理 | 通知 | 备注 |
|------|---------|------|------|
| Windows | Windows Core Audio (COM) | 系统通知 | 无外部依赖 |
| macOS | CoreAudio（cgo 原生绑定） | 用户通知中心 | 无外部依赖 |
| Linux | pactl CLI | 桌面通知服务 | PulseAudio / PipeWire 自带；托盘需 `libayatana-appindicator3` |

## 安装

前往 [Releases](https://github.com/Sggoodman/audio-switch/releases) 下载对应平台安装包：

| 平台 | 文件 |
|------|------|
| macOS（Apple Silicon / Intel 通用） | `audio-switch-<版本>.dmg` |
| Windows | `audio-switch-setup-<版本>-amd64.exe` |
| Linux (Debian 系) | `audio-switch_<版本>_amd64.deb` |
| Linux（其他发行版） | `audio-switch-<版本>-linux-amd64.tar.gz`（含 install.sh） |

## 使用

1. 启动后应用常驻系统托盘（无 Dock 图标 / 任务栏窗口）
2. 点击托盘图标：菜单中列出所有输出设备，点击即切换
3. **偏好设置** 中选择设备 A / B 与音量预设，可录制自定义全局热键
4. 任何时候按热键（默认 `Ctrl+Alt+S`）在 A/B 设备间一键切换

## 从源码构建

需要 Go 1.27+（mygo 要求）；打包 macOS 版另需 Xcode 命令行工具（cgo 编译 CoreAudio 绑定），打包 Windows 安装器另需 NSIS（`brew install makensis`）。

```bash
git clone https://github.com/Sggoodman/audio-switch.git
cd audio-switch

# 开发调试（直出二进制）
./scripts/build.sh debug

# 打安装包
./scripts/build.sh app          # macOS: .app + .dmg (universal)
./scripts/build.sh windows-app  # Windows: setup.exe（纯 Go 交叉编译，无需 mingw）
./scripts/build.sh linux        # Linux: .deb + tar.gz
./scripts/build.sh all          # 全平台

# 开发热重载（改代码自动重编译重启）
go tool mygo dev
```

## 技术栈

- [Go](https://go.dev/) — 主语言
- [mygo](https://github.com/egoist/mygo) — 桌面框架：托盘 / 全局热键 / 通知 / 开机自启 / native UI 设置窗口
- [go-wca](https://github.com/moutend/go-wca) — Windows Core Audio API 绑定
- [zap](https://github.com/uber-go/zap) — 结构化日志

音频设备控制在 macOS 上通过 cgo 直接调用 CoreAudio，Windows 上基于 COM，Linux 上通过 pactl（兼容 PulseAudio / PipeWire）。

## 许可证

[MIT](LICENSE)
