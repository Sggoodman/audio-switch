package autostart

// Manager 开机自启管理接口
type Manager interface {
	// Enable 启用开机自启，exePath 为当前可执行文件路径
	Enable(exePath string) error
	// Disable 禁用开机自启
	Disable() error
	// IsEnabled 检查是否已启用开机自启
	IsEnabled() (bool, error)
}

// New 返回当前平台的 Manager 实现
func New() Manager {
	return newManager()
}
