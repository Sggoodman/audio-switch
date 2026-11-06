//go:build !windows && !darwin

package hotkey

import "golang.design/x/hotkey"

// 平台修饰键映射：Linux X11 下 Mod1 为 Alt，Mod4 为 Super/Win
const (
	modAlt = hotkey.Mod1
	modCmd = hotkey.Mod4
)
