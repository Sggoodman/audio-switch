/*
preload.js - Windows 音频设备切换插件
仅支持 Windows，使用 AudioDeviceCmdlets PowerShell 模块
*/

const { exec, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

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
 */
function encodePowerShellScript(script) {
  const utf16le = Buffer.from(script, 'utf16le');
  return utf16le.toString('base64');
}

/**
 * PowerShell 公共前缀：修复 PSModulePath + 设置 UTF-8 输出编码
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
 * Windows Toast 系统通知
 * @param {string} message - 通知消息
 * @param {boolean} isError - 是否为错误通知
 */
function notify(message, isError = false) {
  // 检查是否已禁用通知
  try {
    const doc = window.utools.db.get('settings');
    if (doc && doc.data && doc.data.notificationEnabled === false) {
      return;
    }
  } catch (e) {}

  const safeMsg = message.replace(/'/g, "''");

  const script = `
    ${PS_PREFIX}
    Import-Module BurntToast
    New-BurntToastNotification -Text '音频切换', '${safeMsg}' -Silent -UniqueIdentifier 'audio-switch'
    Start-Sleep -Seconds 2
    Remove-BTNotification -Group 'audio-switch'
  `;

  const encodedScript = encodePowerShellScript(script.trim());

  const child = spawn('powershell', [
    '-NoProfile',
    '-ExecutionPolicy', 'Bypass',
    '-EncodedCommand', encodedScript
  ], {
    stdio: 'ignore'
  });
  child.unref();
}

/**
 * 获取偏好切换设备（utools.db 持久化）
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
 * 保存偏好切换设备
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
 * 获取设置
 */
function getSettings() {
  try {
    const doc = window.utools.db.get('settings');
    return doc ? doc.data : { notificationEnabled: true };
  } catch (e) {
    return { notificationEnabled: true };
  }
}

/**
 * 保存设置
 */
function saveSettings(settings) {
  try {
    const existing = window.utools.db.get('settings');
    const doc = {
      _id: 'settings',
      data: settings
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
 * 获取音频输出设备列表
 */
async function getAudioDevices() {
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
    if (parsed && !Array.isArray(parsed) && parsed.name && parsed.id) {
      return [parsed];
    }
    return parsed || [];
  } catch (e) {
    return { error: 'failed', message: `执行失败: ${e.message}` };
  }
}

/**
 * 切换音频输出设备
 * - 如果设置了偏好设备，在两者之间互相切换
 * - 否则在所有 Playback 设备之间循环切换
 * - 切换成功后弹 Windows 系统弹窗
 */
async function switchAudioDevice() {
  const preferred = getPreferredDevices();
  let script;

  if (preferred && preferred.device1 && preferred.device2) {
    const id1 = preferred.device1.id;
    const id2 = preferred.device2.id;
    script = `
      ${PS_PREFIX}
      try {
        Import-Module AudioDeviceCmdlets -ErrorAction Stop
        $current = Get-AudioDevice -Playback | Select-Object -ExpandProperty ID
        $targetId = ''
        $targetVolume = ''
        if ($current -eq '${id1}') {
          $targetId = '${id2}'
          $vol2 = '${preferred.device2.volume ?? ''}'
          if ($vol2 -ne '') { $targetVolume = $vol2 }
        } else {
          $targetId = '${id1}'
          $vol1 = '${preferred.device1.volume ?? ''}'
          if ($vol1 -ne '') { $targetVolume = $vol1 }
        }
        $device = Get-AudioDevice -List | Where-Object { $_.Type -eq 'Playback' -and $_.ID -eq $targetId }
        if ($device) {
          $device | Set-AudioDevice -ErrorAction Stop | Out-Null
          if ($targetVolume -ne '') {
            Set-AudioDevice -PlaybackVolume ([int]$targetVolume) -ErrorAction Stop | Out-Null
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
 * 设置默认设备音量 (0-100)
 */
async function setDeviceVolume(deviceId, volume) {
  const script = `
    ${PS_PREFIX}
    try {
      Import-Module AudioDeviceCmdlets -ErrorAction Stop
      Set-AudioDevice -PlaybackVolume ${volume} -ErrorAction Stop | Out-Null
      Write-Output 'OK'
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
 * 检查 AudioDeviceCmdlets 模块是否可用
 */
async function checkRequirements() {
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
      return { ready: true };
    }
    return {
      ready: false,
      message: '需要安装 AudioDeviceCmdlets 模块',
      hint: '在 PowerShell 中运行: Install-Module -Name AudioDeviceCmdlets -Force'
    };
  } catch (e) {
    return {
      ready: false,
      message: '需要安装 AudioDeviceCmdlets 模块',
      hint: '在 PowerShell 中运行: Install-Module -Name AudioDeviceCmdlets -Force'
    };
  }
}

// 暴露服务接口
window.services = {
  checkRequirements: () => checkRequirements(),
  getAudioDevices: () => getAudioDevices(),
  switchAudioDevice: () => switchAudioDevice(),
  notify: (msg, isError = false) => notify(msg, isError),
  getPreferredDevices: () => getPreferredDevices(),
  savePreferredDevices: (d1, d2) => savePreferredDevices(d1, d2),
  setDeviceVolume: (id, vol) => setDeviceVolume(id, vol),
  getSettings: () => getSettings(),
  saveSettings: (settings) => saveSettings(settings),
  redirectHotKeySetting: () => {
    if (window.utools && window.utools.redirectHotKeySetting) {
      window.utools.redirectHotKeySetting('快速切换音频');
    }
  }
};
