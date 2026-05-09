# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

uTools 插件，用于快捷切换 Windows 音频输出设备。React 19 + MUI 7 + Electron preload 架构，通过 PowerShell 的 AudioDeviceCmdlets 模块控制系统音频设备。

## 构建命令

| 命令 | 说明 |
|------|------|
| `npm install` | 安装依赖 |
| `npm run dev` | 开发模式（webpack watch） |
| `npm run build` | 生产构建 |

构建产物输出到 `dist/`：`index.js`（React 应用）、`preload.js`（Electron preload）、`index.html`、`logo.png`、`plugin.json`。

无测试框架配置。

## 架构

### 双层 Webpack 构建 (`webpack.config.js`)

1. **electron-preload** 目标：`bridge/preload.js` → `dist/preload.js`（Node.js 环境，Babel target electron 22）
2. **web** 目标：`src/index.js` → `dist/index.js`（浏览器环境，Babel target chrome 108，支持 JSX/LESS）

`public/` 目录通过 CopyWebpackPlugin 原样复制到 `dist/`。

### 双入口特性 (`public/plugin.json`)

- **audio-settings**（"音频输出设置"）：React UI 页面，展示设备列表、配置偏好设备、音量预设、通知开关
- **audio-quick-switch**（"快速切换音频"）：在 preload 层直接执行切换并弹出系统通知后退出，**跳过 React 加载**以优化速度。用户可为此入口绑定全局快捷键实现一键切换

### 通信层

`bridge/preload.js` 通过 `window.services` 对象暴露 API 给渲染进程：

| 方法 | 说明 |
|------|------|
| `checkRequirements()` | 检查 AudioDeviceCmdlets 模块是否可用 |
| `getAudioDevices()` | 获取音频输出设备列表 |
| `switchAudioDevice()` | 在偏好设备间切换，或循环切换所有设备 |
| `setDeviceVolume(id, vol)` | 设置当前设备音量 (0-100) |
| `notify(msg, isError)` | 弹出 BurntToast 系统通知 |
| `getPreferredDevices()` / `savePreferredDevices(d1, d2)` | 读写偏好设备配置 |
| `getSettings()` / `saveSettings(settings)` | 读写通知开关等设置 |
| `redirectHotKeySetting()` | 跳转 uTools 快捷键设置页 |
| `registerPluginEnterCallback(cb)` | 注册 onPluginEnter 回调（供 React 使用） |

### 数据持久化

使用 uTools 内置的 PouchDB 风格数据库 (`window.utools.db`)，通过 `_id` / `_rev` 机制管理文档版本。存储两类数据：
- `preferred_devices`：两个偏好设备及其音量预设
- `settings`：通知开关等用户设置

### UI 技术栈

- React 19 + MUI 7（@emotion 样式引擎）
- 样式以 MUI `sx` prop 为主，`index.less` 为辅
- 跟随系统明暗主题（`prefers-color-scheme`）
- `ErrorBoundary.js` 捕获渲染错误和未处理的 Promise 异常

## Windows PowerShell 规范

修改 `bridge/preload.js` 中的 PowerShell 命令时**必须遵守**：

- **使用 `-EncodedCommand` + Base64 (UTF-16 LE) 编码**，通过 `encodePowerShellScript()` 函数
- **禁止使用** `-Command` + 字符串拼接，会导致中文编码/转义问题
- 每个脚本都应包含 `PS_PREFIX`（修复 PSModulePath + 设置 UTF-8 输出编码）

```javascript
const script = `${PS_PREFIX} ...your script...`;
const encoded = encodePowerShellScript(script.trim());
await execPromise(`powershell -NoProfile -EncodedCommand ${encoded}`);
```

### PowerShell 模块依赖

- **AudioDeviceCmdlets**：音频设备枚举和切换
- **BurntToast**：系统通知弹窗

## 开发流程

1. 修改源码（`src/` 或 `bridge/`）
2. `npm run build` 构建
3. 将 `dist/` 目录内容复制到 uTools 插件目录测试
