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

## 项目结构

- `ui/` - UI 相关代码（托盘、设置窗口）
- `internal/audio/` - 音频设备接口（平台相关）
- `internal/notify/` - 通知系统（平台相关）
- `internal/hotkey/` - 全局热键支持
- `internal/config/` - 配置管理
- `scripts/` - 构建脚本
- `assets/` - 图标等资源文件
