/*
preload.js - Windows 音频设备切换插件
仅支持 Windows
koffi 用于快速设备枚举（<10ms），C# exe 用于设备切换和音量设置，PowerShell 用于通知和降级回退
koffi 延迟加载：首次调用时才加载，避免原生模块崩溃导致整个 preload 失效
*/

const { exec, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

// ---- 调试日志（写入文件，窗口关闭后仍可查看）----
const LOG_FILE = path.join(process.env.TEMP || 'C:\\Temp', 'audio-switch-debug.log');
function _log(msg) {
  try {
    const ts = new Date().toISOString().substr(11, 12);
    fs.appendFileSync(LOG_FILE, `[${ts}] ${msg}\n`);
  } catch (e) {}
}
_log('=== preload.js 加载开始 ===');
process.on('uncaughtException', (e) => _log('UNCAUGHT: ' + e.stack));
process.on('unhandledRejection', (r) => _log('UNHANDLED: ' + r));

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

// ============================================================
// koffi Core Audio 模块（延迟加载）
// undefined = 未加载, null = 加载失败, object = 可用
// ============================================================
let _koffiApi = undefined;

function loadKoffiApi() {
  if (_koffiApi !== undefined) return _koffiApi;

  try {
    const koffi = require('koffi');
    const ole32 = koffi.load('ole32.dll');
    const msvcrt = koffi.load('msvcrt.dll');

    // ---- 类型定义 ----
    const GUID = koffi.struct('GUID', {
      Data1: 'unsigned long', Data2: 'unsigned short',
      Data3: 'unsigned short', Data4: 'unsigned char[8]',
    });
    const VoidPtrHolder = koffi.struct('VoidPtrHolder', { ptr: 'void *' });
    const UIntHolder = koffi.struct('UIntHolder', { val: 'unsigned int' });
    const PROPERTYKEY = koffi.struct('PROPERTYKEY', {
      fmtid: GUID,
      pid: 'unsigned long',
    });
    const PROPVARIANT = koffi.struct('PROPVARIANT', {
      vt: 'unsigned short',
      wReserved1: 'unsigned short',
      wReserved2: 'unsigned short',
      wReserved3: 'unsigned short',
      pwszVal: 'void *',
      _pad: 'unsigned char[8]',
    });

    // ---- Proto ----
    const pad_t = koffi.proto('long __stdcall PAD(void*, void*, void**)');
    const R_t = koffi.proto('unsigned long __stdcall R(void*)');
    const EAE_t = koffi.proto('long __stdcall EAE(void*, int, unsigned int, _Out_ void*)');
    const GDAE_t = koffi.proto('long __stdcall GDAE(void*, int, int, _Out_ void*)');
    const GI_t = koffi.proto('long __stdcall GI(void*, _Out_ void*)');
    const GC_t = koffi.proto('long __stdcall GC(void*, _Out_ void*)');
    const IT_t = koffi.proto('long __stdcall IT(void*, unsigned int, _Out_ void*)');
    const OPS_t = koffi.proto('long __stdcall OPS(void*, unsigned long, _Out_ void*)');
    const GV_t = koffi.proto('long __stdcall GV(void*, PROPERTYKEY*, _Out_ PROPVARIANT*)');
    const GAT_t = koffi.proto('long __stdcall GAT(void*, unsigned int, _Out_ PROPERTYKEY*)');

    // ---- Vtable 结构体 ----
    // IMMDeviceEnumerator::GetDevice 签名: (this, LPCWSTR id, IMMDevice** out)
    const GD2_t = koffi.proto('long __stdcall GD2(void*, str16, _Out_ void*)');
    // IMMDeviceEnumerator::SetDefaultAudioEndpoint (Win8.1+) 签名: (this, EDataFlow, ERole, IMMDevice*)
    const SDAE_t = koffi.proto('long __stdcall SDAE(void*, int, int, void*)');
    const EnumVtbl = koffi.struct('EnumVtbl', {
      QI: koffi.pointer(pad_t), AR: koffi.pointer(R_t), Release: koffi.pointer(R_t),
      EnumAudioEndpoints: koffi.pointer(EAE_t), GetDefaultAudioEndpoint: koffi.pointer(GDAE_t),
      GetDevice: koffi.pointer(GD2_t), RENC: koffi.pointer(pad_t), URENC: koffi.pointer(pad_t),
      SetDefaultAudioEndpoint: koffi.pointer(SDAE_t),
    });
    const DeviceVtbl = koffi.struct('DeviceVtbl', {
      QI: koffi.pointer(pad_t), AR: koffi.pointer(R_t), Release: koffi.pointer(R_t),
      Activate: koffi.pointer(pad_t), OpenPropertyStore: koffi.pointer(OPS_t),
      GetId: koffi.pointer(GI_t), GetState: koffi.pointer(pad_t),
    });
    const CollectionVtbl = koffi.struct('CollectionVtbl', {
      QI: koffi.pointer(pad_t), AR: koffi.pointer(R_t), Release: koffi.pointer(R_t),
      GetCount: koffi.pointer(GC_t), Item: koffi.pointer(IT_t),
    });
    const GCPS_t = koffi.proto('long __stdcall GCPS(void*, _Out_ void*)');
    const SV_t = koffi.proto('long __stdcall SV(void*, void*, void*)');
    const CM_t = koffi.proto('long __stdcall CM(void*)');
    const PropStoreVtbl = koffi.struct('PropStoreVtbl', {
      QI: koffi.pointer(pad_t), AR: koffi.pointer(R_t), Release: koffi.pointer(R_t),
      GetCount: koffi.pointer(GCPS_t),
      GetAt: koffi.pointer(GAT_t),
      GetValue: koffi.pointer(GV_t),
      SetValue: koffi.pointer(SV_t),
      Commit: koffi.pointer(CM_t),
    });

    // ---- API 函数 ----
    const CoInit = ole32.func('long __stdcall CoInitializeEx(void*, unsigned long)');
    const CLSIDFromStr = ole32.func('long __stdcall CLSIDFromString(str16, _Out_ GUID*)');
    const CoCreate = ole32.func('long __stdcall CoCreateInstance(GUID*, void*, unsigned long, GUID*, _Out_ void**)');
    const CoFree = ole32.func('void __stdcall CoTaskMemFree(void*)');
    const memcpy = msvcrt.func('void * memcpy(_Out_ void*, const void*, unsigned long long)');
    const StrBuf = koffi.struct('StrBuf', { data: 'unsigned char[512]' });

    // ---- 辅助函数 ----
    function readVtbl(objPtr, VtblType) {
      let ph = {};
      memcpy(koffi.as(ph, 'VoidPtrHolder *'), objPtr, 8);
      let vtbl = {};
      memcpy(koffi.as(vtbl, koffi.pointer(VtblType)), ph.ptr, koffi.sizeof(VtblType));
      return vtbl;
    }

    function readWStr(ptr) {
      if (!ptr) return '';
      let buf = {};
      memcpy(koffi.as(buf, koffi.pointer(StrBuf)), ptr, 512);
      const bytes = buf.data;
      let s = '';
      for (let i = 0; i < bytes.length; i += 2) {
        const c = bytes[i] | (bytes[i + 1] << 8);
        if (c === 0) break;
        s += String.fromCharCode(c);
      }
      return s;
    }

    function callOut(methodPtr, proto, ...args) {
      const fn = koffi.decode(methodPtr, proto);
      let out = {};
      const hr = fn(...args, koffi.as(out, 'VoidPtrHolder *'));
      return [hr, out.ptr];
    }

    function callOutUint(methodPtr, proto, ...args) {
      const fn = koffi.decode(methodPtr, proto);
      let out = {};
      const hr = fn(...args, koffi.as(out, 'UIntHolder *'));
      return [hr, out.val];
    }

    // ---- 状态 ----
    let enumPtr = null;
    let eVtbl = null;

    // ---- 公共方法 ----
    function init() {
      if (enumPtr) return;
      const hr = CoInit(null, 0x02);
      // 0=OK, 1=S_FALSE(已初始化), 0x80010106=RPC_E_CHANGED_MODE(COM已用其他模式初始化，可直接用)
      if (hr !== 0 && hr !== 1 && hr !== 0x80010106) {
        throw new Error('CoInitializeEx failed: 0x' + (hr >>> 0).toString(16));
      }

      let clsid = {}, iid = {};
      CLSIDFromStr('{BCDE0395-E52F-467C-8E3D-C4579291692E}', clsid);
      CLSIDFromStr('{A95664D2-9614-4F35-A746-DE8DB63617E6}', iid);
      let out = [null];
      const hr2 = CoCreate(clsid, null, 0x17, iid, out);
      if (hr2 !== 0) throw new Error('CoCreateInstance failed');

      enumPtr = out[0];
      eVtbl = readVtbl(enumPtr, EnumVtbl);
    }

    function getDefaultDeviceId() {
      if (!enumPtr) throw new Error('Not initialized');
      _log('getDefaultId: call GetDefaultAudioEndpoint');
      const [hr, defPtr] = callOut(eVtbl.GetDefaultAudioEndpoint, GDAE_t, enumPtr, 0, 0);
      _log('getDefaultId: hr=0x' + (hr >>> 0).toString(16) + ' ptr=' + !!defPtr);
      if (hr !== 0 || !defPtr) return '';

      _log('getDefaultId: readVtbl');
      const dVtbl = readVtbl(defPtr, DeviceVtbl);
      _log('getDefaultId: call GetId');
      const [hr2, idPtr] = callOut(dVtbl.GetId, GI_t, defPtr);
      _log('getDefaultId: call readWStr');
      const id = hr2 === 0 && idPtr ? readWStr(idPtr) : '';
      if (idPtr) CoFree(idPtr);
      _log('getDefaultId: Release');
      koffi.call(dVtbl.Release, R_t, defPtr);
      _log('getDefaultId: done len=' + id.length);
      return id;
    }

    function getDevices() {
      if (!enumPtr) throw new Error('Not initialized');

      const [hr, collPtr] = callOut(eVtbl.EnumAudioEndpoints, EAE_t, enumPtr, 0, 1);
      if (hr !== 0 || !collPtr) return [];

      const cVtbl = readVtbl(collPtr, CollectionVtbl);
      const [hr2, count] = callOutUint(cVtbl.GetCount, GC_t, collPtr);
      _log('koffi getDevices: count=' + count);
      const defaultId = getDefaultDeviceId();
      _log('koffi getDevices: defaultId 长度=' + defaultId.length);

      const devices = [];
      for (let i = 0; i < count; i++) {
        const [hr3, devPtr] = callOut(cVtbl.Item, IT_t, collPtr, i);
        if (hr3 !== 0 || !devPtr) continue;

        const dVtbl = readVtbl(devPtr, DeviceVtbl);

        // 获取设备 ID
        const [hr4, idPtr] = callOut(dVtbl.GetId, GI_t, devPtr);
        const id = hr4 === 0 && idPtr ? readWStr(idPtr) : '';
        if (idPtr) CoFree(idPtr);

        // 通过 PropertyStore 扫描获取设备名称
        // PKEY_Device_FriendlyName = {A45C254E...}, pid=14
        let name = '(unknown)';
        const [hr5, storePtr] = callOut(dVtbl.OpenPropertyStore, OPS_t, devPtr, 0);
        if (hr5 === 0 && storePtr) {
          const sVtbl = readVtbl(storePtr, PropStoreVtbl);
          const getValFn = koffi.decode(sVtbl.GetValue, GV_t);

          // 扫描所有属性，找 fmtid.Data1 === 0xA45C254E && pid === 14
          let countHolder = {};
          koffi.call(sVtbl.GetCount, GCPS_t, storePtr, koffi.as(countHolder, 'UIntHolder *'));
          const propCount = countHolder.val;

          for (let j = 0; j < propCount; j++) {
            let pkOut = {};
            const pkHr = koffi.call(sVtbl.GetAt, GAT_t, storePtr, j, pkOut);
            if (pkHr === 0 && pkOut.fmtid && pkOut.fmtid.Data1 === 0xA45C254E && pkOut.pid === 14) {
              let pvOut = {};
              const valHr = getValFn(storePtr, koffi.as(pkOut, 'PROPERTYKEY *'), pvOut);
              if (valHr === 0 && pvOut.vt === 31 && pvOut.pwszVal) {
                name = readWStr(pvOut.pwszVal);
                CoFree(pvOut.pwszVal);
              }
              break;
            }
          }

          koffi.call(sVtbl.Release, R_t, storePtr);
        }

        devices.push({ id: id, name: name, isDefault: id === defaultId });
        koffi.call(dVtbl.Release, R_t, devPtr);
      }

      koffi.call(cVtbl.Release, R_t, collPtr);
      return devices;
    }

    _koffiApi = { init, getDevices, getDefaultDeviceId };
    _log('koffi: 全部初始化成功');
  } catch (e) {
    _log('koffi: 加载失败: ' + e.message);
    _koffiApi = null;
  }

  return _koffiApi;
}

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
      _log('notify: 已禁用，跳过');
      return;
    }
  } catch (e) {}
  _log('notify: ' + message);

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
 * 优先使用 koffi（<10ms），失败回退 PowerShell
 */
async function getAudioDevices() {
  // ---- koffi 快速路径 ----
  const api = loadKoffiApi();
  if (api) {
    try {
      api.init();
      const devices = api.getDevices();
      if (devices && devices.length > 0) return devices;
    } catch (e) {
      _log('getAudioDevices koffi 失败: ' + e.message);
    }
  }

  // ---- PowerShell 回退 ----
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

// ============================================================
// C# 编译切换工具（csc.exe 预编译，独立进程运行）
// 避免 Electron STA 限制：exe 在自己的进程中初始化 COM
// ============================================================
// C# 源码版本号，变更时强制重新编译
const CS_VERSION = 'v5';
const SWITCH_EXE_DIR = path.join(process.env.TEMP || 'C:\\Temp', 'audio-switch');
const SWITCH_EXE_PATH = path.join(SWITCH_EXE_DIR, `SwitchAudio_${CS_VERSION}.exe`);

// 通过反射加载 AudioDeviceCmdlets.dll，切换设备 + 设置音量
const CS_SOURCE = `
using System;
using System.IO;
using System.Reflection;
class P {
  [STAThread]
  static int Main(string[] a) {
    var log = Path.Combine(Path.GetTempPath(), "audio-switch", "switch-log.txt");
    try {
      if (a.Length < 1) { Console.Write("ERROR:No device ID"); return 1; }
      string deviceId = a[0];
      int volume = a.Length >= 2 ? int.Parse(a[1]) : -1;
      File.WriteAllText(log, "DevId: " + deviceId + " Vol: " + volume + "\\n");

      // 查找 AudioDeviceCmdlets.dll
      string docs = Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments);
      string[] moduleDirs = {
        Path.Combine(docs, "PowerShell", "Modules", "AudioDeviceCmdlets"),
        Path.Combine(docs, "WindowsPowerShell", "Modules", "AudioDeviceCmdlets"),
      };
      string dllPath = null;
      foreach (var md in moduleDirs) {
        if (!Directory.Exists(md)) continue;
        foreach (var ver in Directory.GetDirectories(md)) {
          var p = Path.Combine(ver, "AudioDeviceCmdlets.dll");
          if (File.Exists(p)) { dllPath = p; break; }
        }
        if (dllPath != null) break;
      }
      if (dllPath == null) {
        File.WriteAllText(log, "ERROR:AudioDeviceCmdlets.dll not found\\n");
        Console.Error.WriteLine("ERROR:AudioDeviceCmdlets.dll not found");
        return 1;
      }
      File.AppendAllText(log, "DLL: " + dllPath + "\\n");

      var asm = Assembly.LoadFrom(dllPath);
      var pcType = asm.GetType("CoreAudioApi.PolicyConfigClient");
      var eRoleType = asm.GetType("CoreAudioApi.ERole");
      if (pcType == null || eRoleType == null) {
        File.AppendAllText(log, "ERROR:Type not found\\n");
        Console.Error.WriteLine("ERROR:Type not found");
        return 1;
      }

      // 切换默认设备
      var pc = Activator.CreateInstance(pcType);
      var sdeMethod = pcType.GetMethod("SetDefaultEndpoint");
      for (int role = 0; role <= 2; role++) {
        var eRole = Enum.ToObject(eRoleType, role);
        sdeMethod.Invoke(pc, new object[] { deviceId, eRole });
        File.AppendAllText(log, "role=" + role + " OK\\n");
      }

      // 设置音量（通过 AudioEndpointVolume）
      if (volume >= 0 && volume <= 100) {
        var enumType = asm.GetType("CoreAudioApi.MMDeviceEnumerator");
        var edfType = asm.GetType("CoreAudioApi.EDataFlow");
        var enumObj = Activator.CreateInstance(enumType);
        var getDefault = enumType.GetMethod("GetDefaultAudioEndpoint");
        var eRender = Enum.ToObject(edfType, 0);
        var eConsole = Enum.ToObject(eRoleType, 0);
        var device = getDefault.Invoke(enumObj, new object[] { eRender, eConsole });
        if (device != null) {
          var aev = device.GetType().GetProperty("AudioEndpointVolume").GetValue(device);
          if (aev != null) {
            aev.GetType().GetProperty("MasterVolumeLevelScalar")
              .SetValue(aev, (float)(volume / 100.0), null);
            File.AppendAllText(log, "Volume: " + volume + "% OK\\n");
          }
        }
      }

      File.AppendAllText(log, "DONE\\n");
      Console.Write("OK");
      return 0;
    } catch (Exception e) {
      var inner = e.InnerException != null ? e.InnerException.Message : e.Message;
      File.AppendAllText(log, "FATAL:" + inner + "\\n" + e.StackTrace + "\\n");
      Console.Error.WriteLine("ERROR:" + inner);
      return 1;
    }
  }
}
`.trim();

/**
 * 确保切换 exe 已编译，返回 exe 路径（失败返回 null）
 */
function ensureSwitchExe() {
  try {
    // 已编译且文件存在则直接返回
    if (fs.existsSync(SWITCH_EXE_PATH)) return SWITCH_EXE_PATH;

    // 查找 csc.exe
    const cscCandidates = [
      'C:\\Windows\\Microsoft.NET\\Framework64\\v4.0.30319\\csc.exe',
      'C:\\Windows\\Microsoft.NET\\Framework\\v4.0.30319\\csc.exe',
    ];
    let cscPath = null;
    for (const p of cscCandidates) {
      if (fs.existsSync(p)) { cscPath = p; break; }
    }
    if (!cscPath) {
      _log('ensureSwitchExe: csc.exe 未找到');
      return null;
    }

    // 写入源码并编译
    if (!fs.existsSync(SWITCH_EXE_DIR)) fs.mkdirSync(SWITCH_EXE_DIR, { recursive: true });
    const srcPath = path.join(SWITCH_EXE_DIR, 'SwitchAudio.cs');
    fs.writeFileSync(srcPath, CS_SOURCE, 'utf8');

    const { execSync } = require('child_process');
    execSync(`"${cscPath}" /nologo /optimize /out:"${SWITCH_EXE_PATH}" "${srcPath}"`, {
      timeout: 10000,
      windowsHide: true,
    });
    _log('ensureSwitchExe: 编译成功');
    return SWITCH_EXE_PATH;
  } catch (e) {
    _log('ensureSwitchExe: 编译失败: ' + e.message);
    return null;
  }
}

/**
 * 通过编译的 C# exe 切换音频设备
 * exe 在独立进程中运行，不受 Electron COM 限制
 */
async function switchViaExe(targetId, volume) {
  const exePath = ensureSwitchExe();
  if (!exePath) return false;

  try {
    const args = volume !== undefined && volume !== '' && volume >= 0
      ? [targetId, String(volume)]
      : [targetId];
    const result = await new Promise((resolve) => {
      const child = spawn(exePath, args, { windowsHide: true });
      const timer = setTimeout(() => { child.kill(); resolve({ code: -1, stdout: '', stderr: 'timeout' }); }, 10000);
      let out = '', err = '';
      child.stdout.on('data', (d) => { out += d; });
      child.stderr.on('data', (d) => { err += d; });
      child.on('close', (code) => { clearTimeout(timer); resolve({ code, stdout: out.trim(), stderr: err.trim() }); });
      child.on('error', (e) => { clearTimeout(timer); resolve({ code: -1, stdout: '', stderr: e.message }); });
    });
    _log('switchViaExe: code=' + result.code + ' stdout=' + result.stdout + ' stderr=' + result.stderr);
    return result.code === 0 && result.stdout === 'OK';
  } catch (e) {
    _log('switchViaExe exception: ' + e.message);
    return false;
  }
}

/**
 * 切换音频输出设备
 * koffi 快速枚举 → C# exe 切换（独立进程，~50ms）→ PowerShell 回退
 * - 如果设置了偏好设备，在两者之间互相切换
 * - 否则在所有 Playback 设备之间循环切换
 * - 切换成功后弹 Windows 系统弹窗
 */
async function switchAudioDevice() {
  const preferred = getPreferredDevices();

  // ---- koffi 快速枚举 ----
  const api = loadKoffiApi();
  if (api) {
    try {
      api.init();
      const currentId = api.getDefaultDeviceId();
      let targetId, targetVolume, targetName;

      if (preferred && preferred.device1 && preferred.device2) {
        if (currentId === preferred.device1.id) {
          targetId = preferred.device2.id;
          targetVolume = preferred.device2.volume;
        } else {
          targetId = preferred.device1.id;
          targetVolume = preferred.device1.volume;
        }
      } else {
        const devices = api.getDevices();
        if (devices.length < 2) return { success: false, message: 'No alternative device' };
        let targetIndex = 0;
        for (let i = 0; i < devices.length; i++) {
          if (devices[i].id === currentId) {
            targetIndex = (i + 1) % devices.length;
            break;
          }
        }
        targetId = devices[targetIndex].id;
        targetName = devices[targetIndex].name;
      }

      // 1. 通过编译的 C# exe 切换 + 设置音量（独立进程）
      const volume = (targetVolume !== undefined && targetVolume !== '') ? parseInt(targetVolume) : undefined;
      const switched = await switchViaExe(targetId, volume);
      _log('switchAudioDevice: exe切换=' + switched);
      if (switched) {
        if (!targetName) {
          const devices = api.getDevices();
          const found = devices.find(d => d.id === targetId);
          targetName = found ? found.name : 'Unknown';
        }

        return { success: true, deviceName: targetName };
      }

      // 2. exe 切换失败，回退 PowerShell
      _log('switchAudioDevice: exe切换失败，回退 PowerShell');
    } catch (e) {
      _log('switchAudioDevice koffi 失败: ' + e.message);
    }
  }

  // ---- PowerShell 完整回退 ----
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
 * 设置默认设备音量 (0-100)（通过 PowerShell）
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
 * 检查环境是否就绪
 * 优先尝试 koffi 初始化，失败回退 PowerShell 检查
 */
async function checkRequirements() {
  // ---- koffi 快速检查 ----
  const api = loadKoffiApi();
  if (api) {
    try {
      api.init();
      return { ready: true };
    } catch (e) {
      // koffi 初始化失败，继续用 PS 检查
    }
  }

  // ---- PowerShell 回退检查 ----
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
_log('preload: 注册 window.services');
window.services = {
  checkRequirements: () => { _log('API: checkRequirements'); return checkRequirements(); },
  getAudioDevices: () => { _log('API: getAudioDevices'); return getAudioDevices(); },
  switchAudioDevice: () => { _log('API: switchAudioDevice'); return switchAudioDevice(); },
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

/**
 * 快捷切换优化：在 preload 层直接处理 onPluginEnter
 * - audio-quick-switch: 跳过 React 加载，直接执行切换 + 通知 + 退出
 * - 其他入口: 通过回调转发给 React 处理
 */
let _pluginEnterCallback = null;

window.services.registerPluginEnterCallback = (cb) => {
  _log('registerPluginEnterCallback 被调用');
  _pluginEnterCallback = cb;
};

try {
  window.utools.onPluginEnter(({ code }) => {
    _log('onPluginEnter: code=' + code);
    if (code === 'audio-quick-switch') {
      switchAudioDevice().then(result => {
        _log('switchAudioDevice 结果: ' + JSON.stringify(result));
        const msg = result.success
          ? `已切换到: ${result.deviceName}`
          : `切换失败: ${result.message}`;
        notify(msg, !result.success);
        setTimeout(() => window.utools.outPlugin(), 500);
      });
    } else if (_pluginEnterCallback) {
      _pluginEnterCallback({ code });
    }
  });
  _log('preload: 注册 onPluginEnter 完成');
} catch (e) {
  _log('preload: onPluginEnter 注册失败: ' + e.message);
}
