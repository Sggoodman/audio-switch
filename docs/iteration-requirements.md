# Audio Switch 迭代需求说明书

> 基于当前代码库（main 分支 d602dbd）的全量审查，按优先级和类别整理。

---

## 一、稳定性与健壮性

### 1.1 Logger 未初始化将导致 panic

**文件**: `main.go:22-25`, `internal/logger/logger.go`

`logger.Init()` 仅在 `os.MkdirAll` 成功时调用。如果创建目录失败，`global` 为 nil，后续所有 `logger.Info/Warn` 调用均会 panic。

**建议**: `Init` 应保证即使文件创建失败也回退到 stdout，或将回退逻辑移到 `init()` 外层。

### 1.2 Logger.Sync() 未在退出时调用

**文件**: `main.go`

zap 的 `Sync()` 会刷写缓冲区到磁盘，但 `main()` 中无 `defer logger.Sync()` 调用。程序退出时最后的日志可能丢失。

### 1.3 saveConfig 逻辑错误：失败时仍记录"配置已保存"

**文件**: `ui/settings.go:467-473`

```go
if err := config.Save(s.cfg); err != nil {
    logger.Warn(...)   // 记录失败
    dialog.ShowError(...) // 弹窗报错，但没有 return
}
logger.Info("Settings", "配置已保存", ...) // ← 无论成功失败都会执行
```

`dialog.ShowError` 后缺少 `return`，导致失败后仍会打印"配置已保存"。

### 1.4 设备获取失败时静默吞没错误

**文件**: `ui/tray.go:108-109`, `ui/settings.go:115`, `ui/settings.go:396`

- `QuickSwitch()` 中 `GetDevices()` 失败直接 `return`，用户无任何反馈
- `buildQuickSwitchSection()` 用 `_` 忽略了 `GetDevices()` 的错误
- `buildDeviceListSection()` 同样忽略错误

**建议**: 至少在 `QuickSwitch` 中通过通知或日志告知用户。

### 1.5 通知发送错误始终被丢弃

**文件**: `ui/tray.go:154`, `internal/notify/notify_windows.go:34`

`notifier.Send()` 的返回值被 `_` 忽略。Windows 实现中 `Send` 始终返回 `nil`（因为推送在 goroutine 中异步执行），无法检测失败。虽然内部有日志，但调用方无法感知。

---

## 二、性能优化

### 2.1 GetDevices() 重复调用（同一场景调用 3 次）

**文件**: `ui/settings.go:83-89`, `ui/settings.go:115`, `ui/settings.go:396`

打开设置窗口时：
1. `Show()` → `RefreshDevices()` 调用一次
2. `buildQuickSwitchSection()` 内部又调用一次
3. `buildDeviceListSection()` 内部再调用一次

每次调用都涉及 COM API 初始化和枚举，开销较大。

**建议**: 在 `NewSettingsWindow` 或 `Show()` 中获取一次设备列表，通过参数传递给各 `build*` 方法。

### 2.2 音量滑块拖动时频繁写入磁盘

**文件**: `ui/settings.go:200-206`, `ui/settings.go:219-225`

滑块 `OnChanged` 每触发一次就调用 `saveConfig()`，连续拖动滑块会在短时间内产生数十次 JSON 序列化 + 磁盘写入。

**建议**: 引入防抖（debounce）机制，例如 300ms 内只保存最后一次。

---

## 三、代码质量

### 3.1 getExePath() 重复定义

**文件**: `main.go:124-130`, `ui/settings.go:479-485`

两个文件中有完全相同的 `getExePath()` 函数。

**建议**: 提取到 `internal/util/exe.go` 或类似位置。

### 3.2 设备 A/B 选择逻辑高度重复

**文件**: `ui/settings.go:130-187`

设备 A 和设备 B 的 `NewSelect` 回调代码几乎完全相同（约 55 行 × 2）。

**建议**: 抽取为通用函数，如 `buildDeviceSelect(deviceNames, deviceMap, cfgField, label string)`。

### 3.3 switchDevice 和 QuickSwitch 通知/刷新逻辑重复

**文件**: `ui/tray.go:106-167`, `ui/tray.go:170-187`

`switchDevice` 和 `QuickSwitch` 中的"发送通知 + RefreshMenu + settings.RefreshDevices"代码块完全相同。

**建议**: 提取为 `notifyAndRefresh(targetName, err)` 方法。

### 3.4 未使用的代码

| 位置 | 说明 |
|------|------|
| `internal/audio/audio.go:4-12` | `FormFactor` 类型和相关常量已定义，但 Windows 实现从未填充该字段，UI 也未使用 |
| `internal/config/config.go:104-106` | `GetPreferredDevices()` 函数未被任何代码调用 |
| `internal/audio/audio.go:24-39` | `FormFactor.String()` 方法未被使用 |

**建议**: 要么在 Windows 实现中读取并使用 `FormFactor`（在托盘菜单区分耳机/扬声器），要么移除死代码。

### 3.5 buildQuickSwitchSection 函数过长

**文件**: `ui/settings.go:112-392`

单个函数约 280 行，包含设备选择、音量滑块、通知开关、开机自启、热键设置等所有 UI 构建逻辑。

**建议**: 已有 `buildQuickSwitchSection` 的命名，但内部仍应拆分：音量滑块、开机自启、热键设置各为独立方法。

---

## 四、用户体验增强

### 4.1 设置窗口每次重建

**文件**: `ui/tray.go:214-216`

每次打开偏好设置都 `NewSettingsWindow` 创建全新窗口。旧窗口未显式关闭/销毁，可能造成资源泄漏。

**建议**: 复用窗口实例，打开时调用 `RefreshDevices()` + 数据绑定更新即可。

### 4.2 缺少设备热插拔检测

当用户插入/拔出 USB 耳机时，托盘菜单和设置窗口不会自动刷新设备列表。

**建议**: 可通过 Windows `WM_DEVICECHANGE` 消息或定时轮询实现。轮询间隔建议 2-3 秒。

### 4.3 无日志文件轮转

**文件**: `internal/logger/logger.go:21`

日志以 `O_APPEND` 模式写入，无大小限制和轮转机制。长期运行后 `app.log` 可能无限增长。

**建议**: 引入 `lumberjack` 或类似库，设置大小上限和文件数限制。

### 4.4 缺少"关于"对话框

当前没有版本号展示、项目链接等信息入口。

### 4.5 托盘菜单无设备类型图标

`Device` 结构体已定义 `FormFactor`（扬声器/耳机/HDMI 等），但托盘菜单中所有设备显示相同，无法直观区分。

### 4.6 刷新按钮体验粗糙

**文件**: `ui/settings.go:439-445`

刷新按钮直接 `win.Hide()` 再 `ShowSettings()` 重建整个窗口，用户会看到窗口闪烁。

**建议**: 仅刷新数据并更新 UI 组件，而非销毁重建窗口。

---

## 五、跨平台一致性

### 5.1 macOS/Linux 通知未使用 logger

**文件**: `internal/notify/notify_darwin.go`, `internal/notify/notify_linux.go`

Windows 实现已迁移到 zap logger，但 macOS 和 Linux 的通知实现中没有错误日志记录。

### 5.2 go-toast 库年久失修

**文件**: `go.mod:8`

`github.com/go-toast/toast` 最后更新于 2019 年。考虑替换为更活跃的库如 `github.com/soniakeys/notify` 或 Windows 原生 API 调用。

---

## 六、构建与工程化

### 6.1 build.sh 中 VERSION 硬编码

**文件**: `scripts/build.sh:8`

```bash
VERSION=${VERSION:-"1.0.0"}
```

版本号默认为 1.0.0 且与 git tag 无关联。构建脚本应自动从 `git describe --tags` 获取版本号。

### 6.2 缺少单元测试

项目中没有任何 `_test.go` 文件。以下模块适合优先补充测试：
- `internal/config` — 配置序列化/反序列化
- `internal/hotkey` — 热键字符串解析
- `internal/autostart` — 各平台启禁用逻辑
- `internal/logger` — 日志初始化和输出

---

## 优先级建议

| 优先级 | 编号 | 说明 |
|--------|------|------|
| P0 | 1.1 | Logger 未初始化可能 panic |
| P0 | 1.3 | saveConfig 失败后仍记录成功 |
| P1 | 1.2 | 日志丢失风险 |
| P1 | 2.1 | GetDevices 重复调用性能问题 |
| P1 | 2.2 | 滑块频繁写入磁盘 |
| P1 | 3.1 | getExePath 重复代码 |
| P2 | 3.2-3.5 | 代码重复和过长函数 |
| P2 | 4.1 | 设置窗口资源管理 |
| P2 | 4.3 | 日志轮转 |
| P2 | 6.2 | 补充单元测试 |
| P3 | 4.2 | 设备热插拔检测 |
| P3 | 4.5 | 设备类型图标 |
| P3 | 5.1-5.2 | 跨平台一致性 |
| P3 | 6.1 | 版本号自动获取 |
