//go:build !darwin

package hotkey

import "golang.design/x/hotkey"

// registerHotkey 非 darwin 平台直接在当前线程注册
func registerHotkey(hk *hotkey.Hotkey) error {
	return hk.Register()
}
