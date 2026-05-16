package audio

// FormFactor 表示音频设备的物理类型
type FormFactor int

const (
	FormFactorUnknown    FormFactor = 0
	FormFactorSpeakers   FormFactor = 1
	FormFactorHeadphones FormFactor = 2
	FormFactorHeadset    FormFactor = 4
	FormFactorHDMI       FormFactor = 8
	FormFactorDisplay    FormFactor = 16
)

// Device 表示一个音频输出设备
type Device struct {
	ID         string
	Name       string
	IsDefault  bool
	FormFactor FormFactor
}

// FormFactorIcon 返回设备类型对应的显示文字
func (f FormFactor) String() string {
	switch f {
	case FormFactorSpeakers:
		return "Speakers"
	case FormFactorHeadphones:
		return "Headphones"
	case FormFactorHeadset:
		return "Headset"
	case FormFactorHDMI:
		return "HDMI"
	case FormFactorDisplay:
		return "Display"
	default:
		return "Unknown"
	}
}

// Audio 定义音频设备操作的跨平台接口
type Audio interface {
	// GetDevices 返回所有音频输出设备
	GetDevices() ([]Device, error)
	// GetDefaultDevice 返回当前默认音频输出设备
	GetDefaultDevice() (*Device, error)
	// SetDefaultDevice 设置指定设备为默认音频输出
	SetDefaultDevice(id string) error
	// SetDeviceVolume 设置指定设备的音量 (0-100)
	SetDeviceVolume(id string, volume int) error
}
