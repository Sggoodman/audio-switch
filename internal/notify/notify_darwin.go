//go:build darwin

package notify

import (
	"audio-switch/internal/logger"
	"fmt"
	"os/exec"
)

// DarwinNotifier 使用 macOS osascript 发送通知
type DarwinNotifier struct{}

// New 创建 macOS 通知实例
func New() Notifier {
	return &DarwinNotifier{}
}

// Send 通过 osascript 显示系统通知
func (n *DarwinNotifier) Send(title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, escapeAppleScript(message), escapeAppleScript(title))
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		logger.Warn("Notify", "推送失败", "error", err, "output", string(out))
		return fmt.Errorf("osascript notification failed: %w\n%s", err, string(out))
	}
	return nil
}

func escapeAppleScript(s string) string {
	result := ""
	for _, c := range s {
		if c == '"' {
			result += `\"`
		} else if c == '\\' {
			result += `\\`
		} else {
			result += string(c)
		}
	}
	return result
}
