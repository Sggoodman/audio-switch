//go:build windows

package hotkey

import (
	"strings"
	"time"
)

// virtualKeys 非修饰键的虚拟键码映射
var virtualKeys = []struct {
	vk   uintptr
	name string
}{
	{0x41, "A"}, {0x42, "B"}, {0x43, "C"}, {0x44, "D"}, {0x45, "E"},
	{0x46, "F"}, {0x47, "G"}, {0x48, "H"}, {0x49, "I"}, {0x4A, "J"},
	{0x4B, "K"}, {0x4C, "L"}, {0x4D, "M"}, {0x4E, "N"}, {0x4F, "O"},
	{0x50, "P"}, {0x51, "Q"}, {0x52, "R"}, {0x53, "S"}, {0x54, "T"},
	{0x55, "U"}, {0x56, "V"}, {0x57, "W"}, {0x58, "X"}, {0x59, "Y"},
	{0x5A, "Z"},
	{0x30, "0"}, {0x31, "1"}, {0x32, "2"}, {0x33, "3"}, {0x34, "4"},
	{0x35, "5"}, {0x36, "6"}, {0x37, "7"}, {0x38, "8"}, {0x39, "9"},
	{0x70, "F1"}, {0x71, "F2"}, {0x72, "F3"}, {0x73, "F4"},
	{0x74, "F5"}, {0x75, "F6"}, {0x76, "F7"}, {0x77, "F8"},
	{0x78, "F9"}, {0x79, "F10"}, {0x7A, "F11"}, {0x7B, "F12"},
	{0x20, "Space"}, {0x09, "Tab"}, {0x0D, "Enter"}, {0x2E, "Delete"},
	{0x26, "Up"}, {0x28, "Down"}, {0x25, "Left"}, {0x27, "Right"},
}

const vkEscape uintptr = 0x1B

// RecordHotkey 轮询检测按键组合，返回热键字符串（如 "Ctrl+Alt+S"）。
// 通过 quit channel 取消，返回空字符串。
func RecordHotkey(quit <-chan struct{}) string {
	// 阶段1：等所有按键释放 + 额外缓冲，避免点击按钮残留
	waitForAllReleased(quit)
	time.Sleep(200 * time.Millisecond)

	// 阶段2：等待新的按键事件
	var lastKey uintptr
	for {
		select {
		case <-quit:
			return ""
		default:
		}

		// 检查 Escape 取消
		if isKeyDown(vkEscape) {
			waitForKeyReleased(vkEscape, quit)
			return ""
		}

		// 检查非修饰键
		pressedVK := uintptr(0)
		pressedName := ""
		for _, k := range virtualKeys {
			if isKeyDown(k.vk) {
				pressedVK = k.vk
				pressedName = k.name
				break
			}
		}

		if pressedVK == 0 {
			lastKey = 0
			time.Sleep(30 * time.Millisecond)
			continue
		}

		// 防抖：同一按键持续按住不重复触发
		if pressedVK == lastKey {
			time.Sleep(30 * time.Millisecond)
			continue
		}
		lastKey = pressedVK

		// 检查修饰键
		mods := HeldModifiers()
		if len(mods) == 0 {
			time.Sleep(30 * time.Millisecond)
			continue
		}

		hotkeyStr := strings.Join(mods, "+") + "+" + pressedName

		// 等所有键释放后再返回，防止注册时残留按键立即触发回调
		waitForAllReleased(quit)

		return hotkeyStr
	}
}

// waitForAllReleased 等待所有按键（修饰键 + 非修饰键）完全释放
func waitForAllReleased(quit <-chan struct{}) {
	for {
		select {
		case <-quit:
			return
		default:
		}
		if !anyKeyPressed() {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
}

// waitForKeyReleased 等待指定按键释放
func waitForKeyReleased(vk uintptr, quit <-chan struct{}) {
	for {
		select {
		case <-quit:
			return
		default:
		}
		if !isKeyDown(vk) {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
}

// anyKeyPressed 检查是否有任意键（修饰键或非修饰键）被按下
func anyKeyPressed() bool {
	if isKeyDown(0x11) || isKeyDown(0x12) || isKeyDown(0x10) {
		return true
	}
	for _, k := range virtualKeys {
		if isKeyDown(k.vk) {
			return true
		}
	}
	return false
}
