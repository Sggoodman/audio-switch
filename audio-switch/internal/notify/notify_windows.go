//go:build windows

package notify

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// WindowsNotifier 使用 .NET Forms 气球通知（无需外部模块）
type WindowsNotifier struct{}

// New 创建 Windows 通知实例
func New() Notifier {
	return &WindowsNotifier{}
}

// Send 通过 PowerShell 调用内置 .NET Forms 发送气球通知
func (n *WindowsNotifier) Send(title, message string) error {
	t := strings.ReplaceAll(title, "'", "''")
	m := strings.ReplaceAll(message, "'", "''")

	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms
$ni = New-Object System.Windows.Forms.NotifyIcon
$ni.Icon = [System.Drawing.SystemIcons]::Information
$ni.BalloonTipTitle = '%s'
$ni.BalloonTipText = '%s'
$ni.Visible = $true
$ni.ShowBalloonTip(3000)
Start-Sleep -Seconds 4
$ni.Dispose()`, t, m)

	encoded := encodePS(script)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-EncodedCommand", encoded)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

// encodePS 将脚本编码为 UTF-16 LE + Base64（PowerShell -EncodedCommand 格式）
func encodePS(script string) string {
	runes := []rune(script)
	buf := make([]byte, len(runes)*2)
	for i, r := range runes {
		buf[i*2] = byte(r)
		buf[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(buf)
}
