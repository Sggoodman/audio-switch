//go:build darwin

package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation

#include <stdlib.h>
#include <string.h>
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <CoreServices/CoreServices.h>

#pragma clang diagnostic ignored "-Wdeprecated-declarations"

// cfstrToC 把 CFString 拷贝为 C 字符串（调用方负责 free）
static char *cfstrToC(CFStringRef s) {
	if (s == NULL) return NULL;
	CFIndex len = CFStringGetMaximumSizeForEncoding(CFStringGetLength(s), kCFStringEncodingUTF8) + 1;
	char *buf = (char *)malloc(len);
	if (buf == NULL) return NULL;
	if (!CFStringGetCString(s, buf, len, kCFStringEncodingUTF8)) {
		free(buf);
		return NULL;
	}
	return buf;
}

// listOutputDevices 枚举所有具有输出流的音频设备（调用方负责 free 返回的数组）
static OSStatus listOutputDevices(AudioDeviceID **out, UInt32 *n) {
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDevices,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMaster
	};
	UInt32 size = 0;
	OSStatus st = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &size);
	if (st != noErr) return st;
	AudioDeviceID *ids = (AudioDeviceID *)malloc(size);
	if (ids == NULL) return kAudioHardwareUnspecifiedError;
	st = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, ids);
	if (st != noErr) { free(ids); return st; }

	AudioDeviceID *outs = (AudioDeviceID *)malloc(size);
	if (outs == NULL) { free(ids); return kAudioHardwareUnspecifiedError; }
	UInt32 m = 0;
	for (UInt32 i = 0; i < size / sizeof(AudioDeviceID); i++) {
		AudioObjectPropertyAddress sa = {
			kAudioDevicePropertyStreamConfiguration,
			kAudioDevicePropertyScopeOutput,
			kAudioObjectPropertyElementMaster
		};
		UInt32 ssize = 0;
		if (AudioObjectGetPropertyDataSize(ids[i], &sa, 0, NULL, &ssize) != noErr || ssize == 0)
			continue;
		AudioBufferList *bufs = (AudioBufferList *)malloc(ssize);
		if (bufs == NULL) continue;
		if (AudioObjectGetPropertyData(ids[i], &sa, 0, NULL, &ssize, bufs) == noErr) {
			UInt32 ch = 0;
			for (UInt32 b = 0; b < bufs->mNumberBuffers; b++)
				ch += bufs->mBuffers[b].mNumberChannels;
			if (ch > 0) outs[m++] = ids[i];
		}
		free(bufs);
	}
	free(ids);
	*out = outs;
	*n = m;
	return noErr;
}

// deviceName 返回设备名（调用方负责 free），失败返回 NULL
static char *deviceName(AudioDeviceID id) {
	AudioObjectPropertyAddress addr = {
		kAudioObjectPropertyName,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMaster
	};
	CFStringRef name = NULL;
	UInt32 size = sizeof(name);
	if (AudioObjectGetPropertyData(id, &addr, 0, NULL, &size, &name) != noErr)
		return NULL;
	char *s = cfstrToC(name);
	CFRelease(name);
	return s;
}

// findDeviceByName 按名称查找输出设备 ID，找不到返回 kAudioObjectUnknown
static AudioDeviceID findDeviceByName(const char *name) {
	AudioDeviceID *ids; UInt32 n;
	if (listOutputDevices(&ids, &n) != noErr) return kAudioObjectUnknown;
	AudioDeviceID found = kAudioObjectUnknown;
	for (UInt32 i = 0; i < n; i++) {
		char *dn = deviceName(ids[i]);
		if (dn != NULL && strcmp(dn, name) == 0) found = ids[i];
		free(dn);
		if (found != kAudioObjectUnknown) break;
	}
	free(ids);
	return found;
}

// defaultOutputDevice 返回当前默认输出设备
static AudioDeviceID defaultOutputDevice(void) {
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMaster
	};
	AudioDeviceID id = kAudioObjectUnknown;
	UInt32 size = sizeof(id);
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &id);
	return id;
}

// setDefaultOutputDevice 设置默认输出设备
static OSStatus setDefaultOutputDevice(AudioDeviceID id) {
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMaster
	};
	return AudioObjectSetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, sizeof(id), &id);
}

// setVolumeOn 对指定 element 写入 VolumeScalar
static OSStatus setVolumeOn(AudioDeviceID id, UInt32 el, Float32 v) {
	AudioObjectPropertyAddress va = {
		kAudioDevicePropertyVolumeScalar,
		kAudioDevicePropertyScopeOutput,
		el
	};
	return AudioObjectSetPropertyData(id, &va, 0, NULL, sizeof(v), &v);
}

// setDeviceVolume 设置设备输出音量 (0.0-1.0)：
// 先按 PreferredChannelsForStereo 逐通道写入（部分设备如 AirPods 仅支持通道级），
// 全部失败则回退到 element 0（内建扬声器等仅 master 通道可控），
// 仍失败则返回错误（HDMI 设备的硬件音量无法软设置）
static OSStatus setDeviceVolume(AudioDeviceID id, Float32 v) {
	AudioObjectPropertyAddress pa = {
		kAudioDevicePropertyPreferredChannelsForStereo,
		kAudioDevicePropertyScopeOutput,
		kAudioObjectPropertyElementMaster
	};
	UInt32 channels[2] = {1, 2};
	UInt32 size = sizeof(channels);
	AudioObjectGetPropertyData(id, &pa, 0, NULL, &size, channels);

	OSStatus st = noErr;
	int ok = 0;
	for (int i = 0; i < 2; i++) {
		OSStatus s = setVolumeOn(id, channels[i], v);
		if (s == noErr) ok++;
		else st = s;
	}
	if (ok > 0) return noErr;
	return setVolumeOn(id, 0, v) != noErr ? st : noErr;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// MacOSAudio 通过 CoreAudio API 控制音频设备
type MacOSAudio struct{}

// New 创建音频操作实例（macOS 平台）
func New() Audio {
	return &MacOSAudio{}
}

// GetDevices 枚举所有音频输出设备
func (a *MacOSAudio) GetDevices() ([]Device, error) {
	var ids *C.AudioDeviceID
	var n C.UInt32
	if st := C.listOutputDevices(&ids, &n); st != C.noErr {
		return nil, fmt.Errorf("list output devices: OSStatus %d", int(st))
	}
	defer C.free(unsafe.Pointer(ids))
	if n == 0 {
		return nil, fmt.Errorf("no output devices found")
	}

	cur := C.defaultOutputDevice()

	devices := make([]Device, 0, int(n))
	for _, id := range unsafe.Slice(ids, int(n)) {
		nameC := C.deviceName(id)
		if nameC == nil {
			continue
		}
		name := C.GoString(nameC)
		C.free(unsafe.Pointer(nameC))
		if name == "" {
			continue
		}
		devices = append(devices, Device{
			ID:        name, // macOS 用设备名作为 ID（与配置存储保持一致）
			Name:      name,
			IsDefault: id == cur,
		})
	}
	return devices, nil
}

// GetDefaultDevice 获取当前默认音频输出设备
func (a *MacOSAudio) GetDefaultDevice() (*Device, error) {
	id := C.defaultOutputDevice()
	if id == C.kAudioObjectUnknown {
		return nil, fmt.Errorf("failed to get default output device")
	}
	nameC := C.deviceName(id)
	if nameC == nil {
		return nil, fmt.Errorf("failed to get default device name")
	}
	name := C.GoString(nameC)
	C.free(unsafe.Pointer(nameC))
	return &Device{ID: name, Name: name, IsDefault: true}, nil
}

// SetDefaultDevice 切换到指定音频输出设备（按设备名）
func (a *MacOSAudio) SetDefaultDevice(id string) error {
	cid := C.CString(id)
	defer C.free(unsafe.Pointer(cid))
	devID := C.findDeviceByName(cid)
	if devID == C.kAudioObjectUnknown {
		return fmt.Errorf("audio device not found: %s", id)
	}
	if st := C.setDefaultOutputDevice(devID); st != C.noErr {
		return fmt.Errorf("set default device: OSStatus %d", int(st))
	}
	return nil
}

// SetDeviceVolume 设置音量 (0-100)
func (a *MacOSAudio) SetDeviceVolume(id string, volume int) error {
	cid := C.CString(id)
	defer C.free(unsafe.Pointer(cid))
	devID := C.findDeviceByName(cid)
	if devID == C.kAudioObjectUnknown {
		return fmt.Errorf("audio device not found: %s", id)
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	if st := C.setDeviceVolume(devID, C.float(float32(volume)/100)); st != C.noErr {
		return fmt.Errorf("set volume: OSStatus %d", int(st))
	}
	return nil
}
