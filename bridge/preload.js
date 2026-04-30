/*
preload.js 说明
- 运行于 Electron 预加载环境（preload），可使用：Node.js API、Electron 渲染进程 API、Web API、第三方 Node.js 库。
- 在此编写 Node.js / Electron 相关逻辑。
- 通过 window.services 向前端 UI 暴露封装后的服务接口。

约束：
- 禁止将 Node.js 原生模块（如 fs、child_process、require 等）直接暴露给前端。
- 仅允许暴露函数形式的受控能力。
*/

const { exec, spawn } = require('child_process');
const os = require('os');
const util = require('util');

/**
 * 获取平台类型
 */
function getPlatform() {
  return os.platform(); // 'win32', 'darwin', 'linux'
}

/**
 * 执行命令并返回结果
 */
function execPromise(command) {
  return new Promise((resolve, reject) => {
    exec(command, { timeout: 10000 }, (error, stdout, stderr) => {
      if (error) {
        reject(error);
      } else {
        resolve(stdout.trim());
      }
    });
  });
}

/**
 * 将 PowerShell 脚本编码为 Base64 用于 -EncodedCommand 参数
 * 避免编码和转义问题
 */
function encodePowerShellScript(script) {
  // PowerShell 使用 UTF-16 LE 编码
  const utf16le = Buffer.from(script, 'utf16le');
  return utf16le.toString('base64');
}

/**
 * 生成修复 PSModulePath 的 PowerShell 前缀
 * 解决模块安装在 PowerShell 7 路径但用 PowerShell 5.x 运行时找不到的问题
 * 同时设置 UTF-8 输出编码，解决中文乱码问题
 */
const PS_PREFIX = `
  [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
  $docsPath = [Environment]::GetFolderPath('MyDocuments')
  $ps7Modules = "$docsPath\\PowerShell\\Modules"
  if (Test-Path $ps7Modules) {
    $env:PSModulePath = "$ps7Modules;$env:PSModulePath"
  }
`.trim();

/**
 * 获取偏好切换设备（使用 utools.db 持久化存储）
 */
function getPreferredDevices() {
  try {
    const doc = window.utools.db.get('preferred_devices');
    return doc ? doc.data : null;
  } catch (e) {
    return null;
  }
}

/**
 * 保存偏好切换设备（包含音量设置）
 */
function savePreferredDevices(device1, device2) {
  try {
    const existing = window.utools.db.get('preferred_devices');
    const doc = {
      _id: 'preferred_devices',
      data: { device1, device2 }
    };
    if (existing && existing._rev) {
      doc._rev = existing._rev;
    }
    const result = window.utools.db.put(doc);
    return result.ok ? { success: true } : { success: false, message: '保存失败' };
  } catch (e) {
    return { success: false, message: e.message };
  }
}

/**
 * 获取指定音频设备的音量 (Windows)
 * 返回 0-100 的整数值
 */
async function getWindowsDeviceVolume(deviceId) {
  const script = `
    ${PS_PREFIX}
    try {
      Import-Module AudioDeviceCmdlets -ErrorAction Stop
      $device = Get-AudioDevice -List | Where-Object { $_.Type -eq 'Playback' -and $_.ID -eq '${deviceId}' }
      if ($device) {
        $volume = [int]($device.Volume * 100)
        Write-Output "OK:$volume"
      } else {
        Write-Output 'ERROR:Device not found'
      }
    } catch {
      Write-Output "ERROR:$($_.Exception.Message)"
    }
  `;
  try {
    const encodedScript = encodePowerShellScript(script.trim());
    const output = await execPromise(`powershell -NoProfile -EncodedCommand ${encodedScript}`);
    if (output.startsWith('ERROR:')) {
      return null;
    }
    if (output.startsWith('OK:')) {
      return parseInt(output.substring(3), 10);
    }
    return null;
  } catch (e) {
    return null;
  }
}

/**
 * 设置指定音频设备的音量 (Windows)
 * volume: 0-100
 */
async function setWindowsDeviceVolume(deviceId, volume) {
  const script = `
    ${PS_PREFIX}
    try {
      Import-Module AudioDeviceCmdlets -ErrorAction Stop
      $device = Get-AudioDevice -List | Where-Object { $_.Type -eq 'Playback' -and $_.ID -eq '${deviceId}' }
      if ($device) {
        $volumeValue = [double]${volume} / 100
        $device | Set-AudioDevice -Volume $volumeValue -ErrorAction Stop | Out-Null
        Write-Output 'OK'
      } else {
        Write-Output 'ERROR:Device not found'
      }
    } catch {
      Write-Output "ERROR:$($_.Exception.Message)"
    }
  `;
  try {
    const encodedScript = encodePowerShellScript(script.trim());
    const output = await execPromise(`powershell -NoProfile -EncodedCommand ${encodedScript}`);
    if (output.startsWith('ERROR:')) {
      return { success: false, message: output.substring(6) };
    }
    return output === 'OK' ? { success: true } : { success: false, message: output };
  } catch (e) {
    return { success: false, message: e.message };
  }
}

/**
 * Windows: 使用 PowerShell 获取音频输出设备
 */
async function getWindowsAudioDevices() {
  const script = `
    ${PS_PREFIX}
    $ErrorActionPreference = 'Stop'
    try {
      Import-Module AudioDeviceCmdlets -ErrorAction Stop
      $devices = Get-AudioDevice -List | Where-Object { $_.Type -eq 'Playback' }
      $result = @()
      foreach ($dev in $devices) {
        $result += [PSCustomObject]@{
          name = $dev.Name
          id = $dev.ID
          isDefault = $dev.Default
        }
      }
      $result | ConvertTo-Json -Compress
    } catch {
      Write-Output "ERROR:$($_.Exception.Message)"
    }
  `;

  try {
    const encodedScript = encodePowerShellScript(script.trim());
    const output = await execPromise(`powershell -NoProfile -EncodedCommand ${encodedScript}`);
    if (output.startsWith('ERROR:')) {
      const errorMsg = output.substring(6);
      return {
        error: 'AudioDeviceCmdlets',
        message: 'AudioDeviceCmdlets 加载失败',
        hint: `错误: ${errorMsg}`
      };
    }
    const parsed = JSON.parse(output);
    // PowerShell ConvertTo-Json 对单元素数组返回对象，需转换为数组
    if (parsed && !Array.isArray(parsed) && parsed.name && parsed.id) {
      return [parsed];
    }
    return parsed || [];
  } catch (e) {
    return { error: 'failed', message: `执行失败: ${e.message}` };
  }
}

/**
 * Windows: 切换音频输出设备
 * 如果设置了偏好设备，在两个偏好设备之间互相切换
 * 否则在所有 Playback 设备之间循环切换
 */
async function switchWindowsAudioDevice() {
  const preferred = getPreferredDevices();
  let script;

  if (preferred && preferred.device1 && preferred.device2) {
    // 双设备互相切换模式
    const id1 = preferred.device1.id;
    const id2 = preferred.device2.id;
    script = `
      ${PS_PREFIX}
      try {
        Import-Module AudioDeviceCmdlets -ErrorAction Stop
        $current = Get-AudioDevice -Playback | Select-Object -ExpandProperty ID
        $targetId = ''
        if ($current -eq '${id1}') {
          $targetId = '${id2}'
        } else {
          $targetId = '${id1}'
        }
        $device = Get-AudioDevice -List | Where-Object { $_.Type -eq 'Playback' -and $_.ID -eq $targetId }
        if ($device) {
          $device | Set-AudioDevice -ErrorAction Stop | Out-Null
          # 设置音量（如果配置中存在）
          $volume = '${preferred.device1.volume ?? ''}'
          if ($targetId -eq '${id2}') {
            $volume = '${preferred.device2.volume ?? ''}'
          }
          if ($volume -ne '') {
            $volumeValue = [double]$volume / 100
            $device | Set-AudioDevice -Volume $volumeValue -ErrorAction Stop | Out-Null
          }
          Write-Output "OK:$($device.Name)"
        } else {
          Write-Output 'ERROR:Device not found'
        }
      } catch {
        Write-Output "ERROR:$($_.Exception.Message)"
      }
    `;
  } else {
    // 循环切换模式
    script = `
      ${PS_PREFIX}
      try {
        Import-Module AudioDeviceCmdlets -ErrorAction Stop
        $current = Get-AudioDevice -Playback | Select-Object -ExpandProperty ID
        $devices = Get-AudioDevice -List | Where-Object { $_.Type -eq 'Playback' }
        $count = $devices.Count
        if ($count -lt 2) {
          Write-Output 'ERROR:No alternative device'
          return
        }
        $found = $false
        foreach ($dev in $devices) {
          if ($found) {
            $dev | Set-AudioDevice -ErrorAction Stop | Out-Null
            Write-Output "OK:$($dev.Name)"
            return
          }
          if ($dev.ID -eq $current) {
            $found = $true
          }
        }
        $devices[0] | Set-AudioDevice -ErrorAction Stop | Out-Null
        Write-Output "OK:$($devices[0].Name)"
      } catch {
        Write-Output "ERROR:$($_.Exception.Message)"
      }
    `;
  }

  try {
    const encodedScript = encodePowerShellScript(script.trim());
    const output = await execPromise(`powershell -NoProfile -EncodedCommand ${encodedScript}`);
    if (output.startsWith('ERROR:')) {
      return { success: false, message: output.substring(6) };
    }
    if (output.startsWith('OK:')) {
      return { success: true, deviceName: output.substring(3) };
    }
    return { success: false, message: output };
  } catch (e) {
    return { success: false, message: e.message };
  }
}

/**
 * macOS: 获取音频输出设备
 */
async function getMacAudioDevices() {
  try {
    // 使用 system_profiler 获取音频设备
    const output = await execPromise('system_profiler SPAudioDataType -json');
    const data = JSON.parse(output);
    const devices = [];

    if (data.SPAudioDataType) {
      const audioData = data.SPAudioDataType;
      for (const device of Object.values(audioData)) {
        if (device['deviceCanBeDefaultOutputDevice'] && device['coreaudio:deviceNameCF']) {
          devices.push({
            name: device['coreaudio:deviceNameCF'],
            id: device['coreaudio:deviceUID'],
            isDefault: device['coreaudio:deviceCanBeDefaultOutputDevice'] === 1
          });
        }
      }
    }
    return devices;
  } catch (e) {
    return { error: 'failed', message: e.message };
  }
}

/**
 * macOS: 切换到下一个音频输出设备
 */
async function switchMacAudioDevice() {
  try {
    // 获取当前默认设备
    const current = await execPromise('SwitchAudioSource -c');
    // 获取所有可用设备
    const allDevices = await execPromise('SwitchAudioSource -a');

    if (!allDevices) {
      return { success: false, message: 'No audio devices found' };
    }

    const devices = allDevices.split('\n').filter(d => d.trim());
    const currentIndex = devices.indexOf(current);

    if (currentIndex === -1 || devices.length < 2) {
      return { success: false, message: 'No alternative device' };
    }

    // 切换到下一个设备
    const nextIndex = (currentIndex + 1) % devices.length;
    const nextDevice = devices[nextIndex];

    await execPromise(`SwitchAudioSource -s "${nextDevice}"`);
    return { success: true, deviceName: nextDevice };
  } catch (e) {
    return { success: false, message: e.message };
  }
}

/**
 * Linux: 获取音频输出设备 (PulseAudio)
 */
async function getLinuxAudioDevices() {
  try {
    const output = await execPromise('pactl list sinks short');
    const lines = output.split('\n').filter(l => l.trim());
    const devices = [];

    for (const line of lines) {
      const parts = line.split(/\s+/);
      if (parts.length >= 2) {
        devices.push({
          name: parts[1],
          id: parts[0],
          isDefault: false
        });
      }
    }

    // 获取默认设备
    try {
      const defaultSink = await execPromise('pactl get-default-sink');
      for (const dev of devices) {
        if (dev.name === defaultSink || dev.id === defaultSink) {
          dev.isDefault = true;
        }
      }
    } catch (e) {
      // ignore
    }

    return devices;
  } catch (e) {
    return { error: 'failed', message: e.message };
  }
}

/**
 * Linux: 切换到下一个音频输出设备 (PulseAudio)
 */
async function switchLinuxAudioDevice() {
  try {
    // 获取当前默认设备
    const current = await execPromise('pactl get-default-sink');
    const output = await execPromise('pactl list sinks short');
    const lines = output.split('\n').filter(l => l.trim());

    const devices = [];
    for (const line of lines) {
      const parts = line.split(/\s+/);
      if (parts.length >= 2) {
        devices.push(parts[1]);
      }
    }

    if (devices.length < 2) {
      return { success: false, message: 'No alternative device' };
    }

    const currentIndex = devices.indexOf(current);
    const nextIndex = (currentIndex + 1) % devices.length;
    const nextDevice = devices[nextIndex];

    await execPromise(`pactl set-default-sink ${nextDevice}`);
    return { success: true, deviceName: nextDevice };
  } catch (e) {
    return { success: false, message: e.message };
  }
}

/**
 * 检查系统需求是否满足
 */
async function checkRequirements() {
  const platform = getPlatform();

  if (platform === 'win32') {
    const checkScript = `
      ${PS_PREFIX}
      $ErrorActionPreference = 'Stop'
      try {
        Import-Module AudioDeviceCmdlets -ErrorAction Stop
        Get-AudioDevice -List -ErrorAction Stop | Out-Null
        Write-Output 'OK'
      } catch {
        Write-Output "ERROR:$($_.Exception.Message)"
      }
    `;
    try {
      const encodedScript = encodePowerShellScript(checkScript.trim());
      const output = await execPromise(`powershell -NoProfile -EncodedCommand ${encodedScript}`);
      if (output === 'OK') {
        return { ready: true, platform };
      }
      return {
        ready: false,
        platform,
        message: '需要安装 AudioDeviceCmdlets 模块',
        hint: '在 PowerShell 中运行: Install-Module -Name AudioDeviceCmdlets -Force'
      };
    } catch (e) {
      return {
        ready: false,
        platform,
        message: '需要安装 AudioDeviceCmdlets 模块',
        hint: '在 PowerShell 中运行: Install-Module -Name AudioDeviceCmdlets -Force'
      };
    }
  } else if (platform === 'darwin') {
    // macOS 通常自带音频切换功能
    return { ready: true, platform };
  } else if (platform === 'linux') {
    // 检查 pactl 是否可用
    try {
      await execPromise('pactl get-default-sink');
      return { ready: true, platform };
    } catch (e) {
      return {
        ready: false,
        platform,
        message: '需要 PulseAudio (pactl)',
        hint: '请确保系统已安装 PulseAudio'
      };
    }
  }

  return { ready: false, message: '不支持的平台' };
}

/**
 * 通用切换逻辑（根据平台分发）
 */
async function switchAudioDevice() {
  const platform = getPlatform();
  if (platform === 'win32') {
    return switchWindowsAudioDevice();
  } else if (platform === 'darwin') {
    return switchMacAudioDevice();
  } else if (platform === 'linux') {
    return switchLinuxAudioDevice();
  }
  return { success: false, message: '不支持的平台' };
}

// 暴露服务接口（供 UI 界面使用）
window.services = {
  /**
   * 获取平台信息
   */
  getPlatform: () => getPlatform(),

  /**
   * 检查系统需求
   */
  checkRequirements: () => checkRequirements(),

  /**
   * 获取音频输出设备列表
   */
  getAudioDevices: async () => {
    const platform = getPlatform();
    if (platform === 'win32') {
      return getWindowsAudioDevices();
    } else if (platform === 'darwin') {
      return getMacAudioDevices();
    } else if (platform === 'linux') {
      return getLinuxAudioDevices();
    }
    return { error: 'unsupported', message: '不支持的平台' };
  },

  /**
   * 切换到下一个音频输出设备
   */
  switchAudioDevice: () => switchAudioDevice(),

  /**
   * 获取偏好切换设备
   */
  getPreferredDevices: () => getPreferredDevices(),

  /**
   * 保存偏好切换设备
   */
  savePreferredDevices: (device1, device2) => savePreferredDevices(device1, device2),

  /**
   * 跳转到快捷键设置页面
   */
  redirectHotKeySetting: () => {
    if (window.utools && window.utools.redirectHotKeySetting) {
      window.utools.redirectHotKeySetting('切换音频');
    }
  },

  /**
   * 获取设备音量 (Windows)
   */
  getDeviceVolume: async (deviceId) => {
    const platform = getPlatform();
    if (platform === 'win32') {
      return getWindowsDeviceVolume(deviceId);
    }
    // macOS 和 Linux 暂不支持
    return null;
  },

  /**
   * 设置设备音量 (Windows)
   */
  setDeviceVolume: async (deviceId, volume) => {
    const platform = getPlatform();
    if (platform === 'win32') {
      return setWindowsDeviceVolume(deviceId, volume);
    }
    return { success: false, message: '当前平台不支持设置音量' };
  }
};
