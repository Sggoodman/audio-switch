//go:build darwin

package hotkey

import "golang.design/x/hotkey"

// 平台修饰键映射：macOS 下 Option 对应 Alt，Command 对应 Win/Cmd
const (
	modAlt = hotkey.ModOption
	modCmd = hotkey.ModCmd
)
