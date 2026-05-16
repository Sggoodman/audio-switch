package notify

// Notifier 定义跨平台通知接口
type Notifier interface {
	// Send 发送系统通知
	Send(title, message string) error
}
