//go:build linux

package notify

import (
	"audio-switch/internal/logger"
	"fmt"
	"os/exec"
)

// LinuxNotifier 使用 notify-send 发送通知
type LinuxNotifier struct{}

// New 创建 Linux 通知实例
func New() Notifier {
	return &LinuxNotifier{}
}

// Send 通过 notify-send 显示桌面通知
func (n *LinuxNotifier) Send(title, message string) error {
	out, err := exec.Command("notify-send", title, message).CombinedOutput()
	if err != nil {
		logger.Warn("Notify", "推送失败", "error", err, "output", string(out))
		return fmt.Errorf("notify-send failed: %w\n%s", err, string(out))
	}
	return nil
}
