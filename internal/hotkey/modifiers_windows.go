//go:build windows

package hotkey

import "golang.design/x/hotkey"

// 平台修饰键映射：Windows 下 Win 键对应 Cmd
const (
	modAlt = hotkey.ModAlt
	modCmd = hotkey.ModWin
)
