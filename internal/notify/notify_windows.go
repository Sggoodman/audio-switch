//go:build windows

package notify

import (
	"audio-switch/internal/logger"

	"github.com/go-toast/toast"
)

const (
	appID = "Audio Switch" // 显示给用户的应用名称
)

// WindowsNotifier 使用 Windows Toast 通知
type WindowsNotifier struct{}

// New 创建 Windows 通知实例
func New() Notifier {
	return &WindowsNotifier{}
}

// Send 发送 Toast 通知（异步，立即返回）
func (n *WindowsNotifier) Send(title, message string) error {
	go func() {
		notification := toast.Notification{
			AppID:    appID,
			Title:    title,
			Message:  message,
			Audio:    toast.Silent,
			Duration: toast.Short,
		}

		if err := notification.Push(); err != nil {
			logger.Warn("Notify", "推送失败", "error", err)
		}
	}()
	return nil
}
