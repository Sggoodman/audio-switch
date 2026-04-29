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
 * Windows: 使用 PowerShell 获取音频输出设备
 */
async function getWindowsAudioDevices() {
  // 尝试使用 AudioDeviceCmdlets 模块
  const script = `
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
      Write-Output 'ERROR:AudioDeviceCmdlets not installed'
    }
  `;

  try {
    const encodedScript = encodePowerShellScript(script.trim());
    const output = await execPromise(`powershell -NoProfile -EncodedCommand ${encodedScript}`);
    if (output.includes('ERROR:AudioDeviceCmdlets not installed')) {
      return { error: 'AudioDeviceCmdlets', message: '请安装 AudioDeviceCmdlets 模块', hint: '在 PowerShell 中运行: Install-Module -Name AudioDeviceCmdlets -Force' };
    }
    const parsed = JSON.parse(output);
    // PowerShell ConvertTo-Json 对单元素数组返回对象，需转换为数组
    if (parsed && !Array.isArray(parsed) && parsed.name && parsed.id) {
      return [parsed];
    }
    return parsed;
  } catch (e) {
    // 备选方案：使用 systeminfo 或其他方法
    return { error: 'failed', message: e.message };
  }
}

/**
 * Windows: 切换到下一个音频输出设备
 */
async function switchWindowsAudioDevice() {
  const script = `
    try {
      Import-Module AudioDeviceCmdlets -ErrorAction Stop
      $current = Get-AudioDevice -Playback | Select-Object -ExpandProperty ID
      $devices = Get-AudioDevice -List | Where-Object { $_.Type -eq 'Playback' -and $_.State -eq 'Active' }
      $count = $devices.Count
      if ($count -lt 2) {
        Write-Output 'ERROR:No alternative device'
        return
      }
      $found = $false
      foreach ($dev in $devices) {
        if ($found) {
          $dev | Set-AudioDevice -ErrorAction Stop
          Write-Output "OK:$($dev.Name)"
          return
        }
        if ($dev.ID -eq $current) {
          $found = $true
        }
      }
      # 循环到第一个
      $devices[0] | Set-AudioDevice -ErrorAction Stop
      Write-Output "OK:$($devices[0].Name)"
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
    // 检查是否安装了 AudioDeviceCmdlets
    try {
      await execPromise('powershell -NoProfile -Command "Import-Module AudioDeviceCmdlets -ErrorAction Stop"');
      return { ready: true, platform };
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

// 暴露服务接口
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
  switchAudioDevice: async () => {
    const platform = getPlatform();
    if (platform === 'win32') {
      return switchWindowsAudioDevice();
    } else if (platform === 'darwin') {
      return switchMacAudioDevice();
    } else if (platform === 'linux') {
      return switchLinuxAudioDevice();
    }
    return { success: false, message: '不支持的平台' };
  },

  /**
   * 跳转到快捷键设置页面
   */
  redirectHotKeySetting: () => {
    if (window.utools && window.utools.redirectHotKeySetting) {
      window.utools.redirectHotKeySetting('切换音频');
    }
  }
};