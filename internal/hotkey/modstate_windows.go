//go:build windows

package hotkey

import "syscall"

var user32 = syscall.NewLazyDLL("user32.dll")
var getAsyncKeyState = user32.NewProc("GetAsyncKeyState")

// HeldModifiers 返回当前按住的修饰键列表
func HeldModifiers() []string {
	var mods []string
	if isKeyDown(0x11) { // VK_CONTROL
		mods = append(mods, "Ctrl")
	}
	if isKeyDown(0x12) { // VK_MENU (Alt)
		mods = append(mods, "Alt")
	}
	if isKeyDown(0x10) { // VK_SHIFT
		mods = append(mods, "Shift")
	}
	return mods
}

func isKeyDown(vk uintptr) bool {
	ret, _, _ := getAsyncKeyState.Call(vk)
	return ret&0x8000 != 0
}
