# Audio Switch - Claude 开发指南

## 编译规则

**重要**: 请始终使用 `scripts/build.sh` 脚本进行编译，不要直接使用 `go build` 命令。

### 编译命令

```bash
# 调试版本（隐藏终端 + 调试符号，用于开发调试）
bash scripts/build.sh debug

# 发布版本（隐藏终端，优化体积）
bash scripts/build.sh windows

# 其他平台
bash scripts/build.sh darwin    # macOS
bash scripts/build.sh linux     # Linux
bash scripts/build.sh all       # 所有平台
```

### 输出目录

所有编译输出都在 `build/` 目录中，不会在项目根目录产生额外的二进制文件。

- 调试版本和发布版本都是: `build/audio-switch.exe`（因为 build 会先清空目录，不需要独特的文件名）

### 为什么使用编译脚本？

1. **统一的输出位置**: 所有编译产物都在 `build/` 目录
2. **正确的编译选项**: 脚本包含了正确的 ldflags 和 GUI 标志
3. **资源文件处理**: 自动复制图标文件到 build 目录
4. **调试符号**: debug 模式保留完整调试信息但仍然隐藏终端
5. **避免混乱**: 不会在根目录产生 .exe 文件

### 重新编译前

如果遇到编译错误或需要重新编译，确保先关闭正在运行的 `audio-switch.exe` 进程，否则可能无法清理 `build/` 目录。

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
| `Hotkey` | ui/tray.go 中热键相关逻辑 |
| `Autostart` | main.go 中开机自启相关逻辑 |
| `Notify` | internal/notify/ |
| `Audio` | internal/audio/ |

### 日志级别

| 级别 | 使用场景 |
|------|----------|
| `Info` | 程序启动/关闭、配置加载/保存成功、设备切换成功、热键注册 |
| `Warn` | 非致命错误（配置加载失败但使用默认值、通知推送失败） |
| `Error` | 致命错误 |
| `Debug` | UI 滑块值变化、初始化细节等调试信息 |

### 规则

1. **禁止使用标准库 `log`**，统一使用 `logger` 包
2. **每条日志必须有模块名**（第一个参数）
3. **使用结构化字段**传递数据，不要用 fmt.Sprintf 拼接到消息中
4. **中文消息描述**，简洁准确
5. **错误日志传递原始 error**：`logger.Warn("Module", "操作失败", "error", err)`

## 项目结构

- `ui/` - UI 相关代码（托盘、设置窗口）
- `internal/audio/` - 音频设备接口（平台相关）
- `internal/notify/` - 通知系统（平台相关）
- `internal/hotkey/` - 全局热键支持
- `internal/config/` - 配置管理
- `internal/logger/` - 日志系统（zap 封装）
- `scripts/` - 构建脚本
- `assets/` - 图标等资源文件

## Release 流程

### 发布新版本
   
1. 编译发布版本：
   ```bash
   bash scripts/build.sh windows

2. 获取远端最新标签并递增版本号：
git fetch --tags
# 查看当前最新标签
git tag --sort=-v:refname | head -5
3. 创建新标签并推送：
git tag vX.Y.Z
git push origin vX.Y.Z
4. 使用 gh CLI 创建 GitHub Release（Token 从环境变量 GITHUB_TOKEN 获取）：
gh release create vX.Y.Z build/audio-switch.exe \
  --title "vX.Y.Z" \
  --notes "release notes here"

注意事项
  
- GitHub Token 通过环境变量 GITHUB_TOKEN 自动获取，无需额外配置
- 版本号格式遵循语义化版本：v主版本.次版本.修订号
- 编译脚本会通过 git describe --tags 自动生成版本信息注入到二进制中