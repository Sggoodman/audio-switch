//go:build windows

package audio

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// IPolicyConfig COM GUIDs
var (
	clsidPolicyConfigClient = ole.NewGUID("{870AF99C-171D-4F9E-AF0D-E63DF40C2BC9}")
	iidIPolicyConfig        = ole.NewGUID("{F8679F50-850A-41CF-9C72-430F290290C8}")
	iidIPolicyConfigVista   = ole.NewGUID("{568B9108-44BF-40B4-9006-86AFE5B5A620}")
)

var procCoCreateInstance = syscall.NewLazyDLL("ole32.dll").NewProc("CoCreateInstance")

// WindowsAudio 使用 Windows Core Audio COM API（纯 Go 实现，无外部依赖）
type WindowsAudio struct{}

func New() Audio { return &WindowsAudio{} }

// coInit 初始化 COM，返回是否需要调用 CoUninitialize
func coInit() bool {
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err == nil {
		return true
	}
	var oleErr *ole.OleError
	if errors.As(err, &oleErr) {
		hr := uint32(oleErr.Code())
		// S_FALSE: 已初始化，引用计数已递增，需要 CoUninitialize
		if hr == 0x00000001 {
			return true
		}
	}
	// RPC_E_CHANGED_MODE 或其他错误：不调用 CoUninitialize
	return false
}

// GetDevices 枚举所有活跃的音频输出设备
func (a *WindowsAudio) GetDevices() ([]Device, error) {
	if coInit() {
		defer ole.CoUninitialize()
	}

	var enumerator *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator, &enumerator,
	); err != nil {
		return nil, fmt.Errorf("create enumerator: %w", err)
	}
	defer enumerator.Release()

	// 获取默认设备 ID
	defaultID := ""
	var defaultDev *wca.IMMDevice
	if err := enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &defaultDev); err == nil {
		var id string
		if err := defaultDev.GetId(&id); err == nil {
			defaultID = id
		}
		defaultDev.Release()
	}

	// 枚举所有活跃输出设备
	var collection *wca.IMMDeviceCollection
	if err := enumerator.EnumAudioEndpoints(wca.ERender, wca.DEVICE_STATE_ACTIVE, &collection); err != nil {
		return nil, fmt.Errorf("enum endpoints: %w", err)
	}
	defer collection.Release()

	var count uint32
	if err := collection.GetCount(&count); err != nil {
		return nil, fmt.Errorf("get count: %w", err)
	}

	var devices []Device
	for i := uint32(0); i < count; i++ {
		var dev *wca.IMMDevice
		if err := collection.Item(i, &dev); err != nil {
			continue
		}

		var id string
		if err := dev.GetId(&id); err != nil {
			dev.Release()
			continue
		}

		name := getDeviceName(dev)
		devices = append(devices, Device{
			ID:        id,
			Name:      name,
			IsDefault: id == defaultID,
		})
		dev.Release()
	}
	return devices, nil
}

// GetDefaultDevice 返回当前默认音频输出设备
func (a *WindowsAudio) GetDefaultDevice() (*Device, error) {
	if coInit() {
		defer ole.CoUninitialize()
	}

	var enumerator *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator, &enumerator,
	); err != nil {
		return nil, fmt.Errorf("create enumerator: %w", err)
	}
	defer enumerator.Release()

	var dev *wca.IMMDevice
	if err := enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &dev); err != nil {
		return nil, fmt.Errorf("no default audio endpoint: %w", err)
	}
	defer dev.Release()

	var id string
	if err := dev.GetId(&id); err != nil {
		return nil, fmt.Errorf("get device id: %w", err)
	}

	name := getDeviceName(dev)
	return &Device{ID: id, Name: name, IsDefault: true}, nil
}

// SetDefaultDevice 通过 IPolicyConfig COM 接口切换默认音频设备
func (a *WindowsAudio) SetDefaultDevice(id string) error {
	if coInit() {
		defer ole.CoUninitialize()
	}
	return setDefaultEndpoint(id)
}

// SetDeviceVolume 切换到指定设备并设置音量（纯 Go 实现）
func (a *WindowsAudio) SetDeviceVolume(id string, volume int) error {
	if coInit() {
		defer ole.CoUninitialize()
	}

	// 先切换设备
	if err := setDefaultEndpoint(id); err != nil {
		return fmt.Errorf("switch device: %w", err)
	}

	// 再设置音量
	if volume < 0 || volume > 100 {
		return nil
	}
	return setEndpointVolume(float32(volume) / 100.0)
}

// ---- 辅助函数 ----

// getDeviceName 从设备属性存储中读取友好名称
func getDeviceName(dev *wca.IMMDevice) string {
	var ps *wca.IPropertyStore
	if err := dev.OpenPropertyStore(wca.STGM_READ, &ps); err != nil {
		return "Unknown"
	}
	defer ps.Release()

	var pv wca.PROPVARIANT
	if err := ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv); err != nil {
		return "Unknown"
	}

	name := pv.String()
	if name == "" {
		return "Unknown"
	}
	return name
}

// setDefaultEndpoint 通过 IPolicyConfig 切换默认音频设备
// 优先使用 Windows 10+ 的 IPolicyConfig，失败回退到 IPolicyConfigVista
func setDefaultEndpoint(deviceID string) error {
	idPtr, err := syscall.UTF16PtrFromString(deviceID)
	if err != nil {
		return fmt.Errorf("invalid device ID: %w", err)
	}

	// IPolicyConfig (Win10+): SetDefaultEndpoint 在 vtable 索引 13
	if err := callSetDefaultEndpoint(iidIPolicyConfig, 13, idPtr); err == nil {
		return nil
	}

	// IPolicyConfigVista: SetDefaultEndpoint 在 vtable 索引 6
	if err := callSetDefaultEndpoint(iidIPolicyConfigVista, 6, idPtr); err != nil {
		return fmt.Errorf("IPolicyConfig 和 IPolicyConfigVista 均失败: %w", err)
	}
	return nil
}

// callSetDefaultEndpoint 创建 IPolicyConfig 实例并调用 SetDefaultEndpoint
func callSetDefaultEndpoint(iid *ole.GUID, vtableIdx int, deviceIDPtr *uint16) error {
	var pc uintptr
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(clsidPolicyConfigClient)),
		0,
		uintptr(wca.CLSCTX_ALL),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&pc)),
	)
	if hr != 0 || pc == 0 {
		return fmt.Errorf("CoCreateInstance: 0x%08X", hr)
	}
	defer comRelease(pc)

	fn := vtableMethod(pc, vtableIdx)
	// 为所有角色设置默认设备: eConsole=0, eMultimedia=1, eCommunications=2
	for role := 0; role <= 2; role++ {
		hr, _, _ = syscall.SyscallN(fn, pc,
			uintptr(unsafe.Pointer(deviceIDPtr)),
			uintptr(role))
		if hr != 0 {
			return fmt.Errorf("SetDefaultEndpoint(role=%d): 0x%08X", role, hr)
		}
	}
	return nil
}

// setEndpointVolume 通过 IAudioEndpointVolume 设置当前默认设备音量 (0.0-1.0)
func setEndpointVolume(level float32) error {
	// 切换后默认设备即为目标设备
	var enumerator *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator, &enumerator,
	); err != nil {
		return fmt.Errorf("create enumerator: %w", err)
	}
	defer enumerator.Release()

	var dev *wca.IMMDevice
	if err := enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &dev); err != nil {
		return fmt.Errorf("get default endpoint: %w", err)
	}
	defer dev.Release()

	var aev *wca.IAudioEndpointVolume
	if err := dev.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, 0, &aev); err != nil {
		return fmt.Errorf("activate IAudioEndpointVolume: %w", err)
	}
	defer aev.Release()

	return aev.SetMasterVolumeLevelScalar(level, nil)
}

// vtableMethod 获取 COM 对象 vtable 中指定索引的方法地址
func vtableMethod(comPtr uintptr, idx int) uintptr {
	vt := *(*uintptr)(unsafe.Pointer(comPtr))
	return *(*uintptr)(unsafe.Pointer(vt + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
}

// comRelease 调用 IUnknown::Release
func comRelease(ptr uintptr) {
	syscall.SyscallN(vtableMethod(ptr, 2), ptr)
}
