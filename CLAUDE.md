# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个 uTools 插件，用于快捷键切换音频输出设备。采用 React + Electron 架构。

## 构建命令

| 命令 | 说明 |
|------|------|
| `npm install` | 安装依赖 |
| `npm run dev` | 开发模式（watch 模式） |
| `npm run build` | 生产构建 |

构建产物输出到 `dist/` 目录，包含：
- `index.js` - 前端 React 应用
- `preload.js` - Electron 预加载脚本
- `index.html`, `logo.png`, `plugin.json` - 静态资源

## 架构

**双层构建配置** (`webpack.config.js`)：
1. **electron-preload** 目标：编译 `bridge/preload.js`
2. **web** 目标：编译 `src/index.js` → `dist/index.js`

**核心架构**：
- `bridge/preload.js` - Node.js 层，通过 `window.services` 暴露 API 给渲染进程
- `src/App.js` - React UI，调用 `window.services` 与系统交互
- `public/plugin.json` - uTools 插件配置（入口、命令、预加载脚本）

**window.services API**：
| 方法 | 说明 |
|------|------|
| `getPlatform()` | 返回 'win32' / 'darwin' / 'linux' |
| `checkRequirements()` | 检查平台依赖是否满足 |
| `getAudioDevices()` | 获取音频设备列表 |
| `switchAudioDevice()` | 切换到下一个音频设备 |
| `redirectHotKeySetting()` | 跳转到 uTools 快捷键设置 |

## 平台依赖

| 平台 | 依赖 | 安装命令 |
|------|------|----------|
| Windows | AudioDeviceCmdlets | `Install-Module -Name AudioDeviceCmdlets -Force` |
| macOS | SwitchAudioSource | `brew install switchaudio-osx` |
| Linux | PulseAudio (pactl) | 通常已预装 |

## Windows PowerShell 注意事项

修改 `bridge/preload.js` 中的 PowerShell 命令时：
- **必须使用 `-EncodedCommand` + Base64 (UTF-16 LE) 编码**
- 避免使用 `-Command` + 字符串拼接，会导致编码/转义问题
- 使用 `encodePowerShellScript()` 函数进行编码

示例：
```javascript
function encodePowerShellScript(script) {
  const utf16le = Buffer.from(script, 'utf16le');
  return utf16le.toString('base64');
}

const script = `try { ... } catch { ... }`;
const encoded = encodePowerShellScript(script.trim());
await execPromise(`powershell -NoProfile -EncodedCommand ${encoded}`);
```

## 开发流程

1. 修改源码（`src/` 或 `bridge/`）
2. 运行 `npm run build` 构建
3. 将 `dist/` 目录内容复制到 uTools 插件目录进行测试
