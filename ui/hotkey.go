package ui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 热键字符串格式与历史配置保持一致："Ctrl+Alt+S"。
// mygo GlobalShortcut 的 accelerator 语法直接兼容该格式，
// 用户已保存的配置无需迁移。

// hotkeyFromKeys 把录制到的修饰键与主键组合为热键字符串
func hotkeyFromKeys(mods ui.Modifiers, key ui.Key) (string, bool) {
	if !isPrimaryKey(key) {
		return "", false
	}
	var parts []string
	if mods&ui.Ctrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&ui.Alt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&ui.Shift != 0 {
		parts = append(parts, "Shift")
	}
	if mods&ui.Super != 0 {
		parts = append(parts, "Super")
	}
	if len(parts) == 0 {
		return "", false // 至少一个修饰键，避免吞掉普通按键
	}
	parts = append(parts, keyName(key))
	return strings.Join(parts, "+"), true
}

// isPrimaryKey 判断按键是否可作为热键主键（排除修饰键与无法命名的键）
func isPrimaryKey(key ui.Key) bool {
	_, ok := primaryKeyNames[key]
	return ok
}

// primaryKeyNames 主键 → 显示名（与历史 parseKey 的拼写一致）
var primaryKeyNames = buildPrimaryKeyNames()

func buildPrimaryKeyNames() map[ui.Key]string {
	m := make(map[ui.Key]string)
	for ch := 'A'; ch <= 'Z'; ch++ {
		m[keyForLetter(ch)] = string(ch)
	}
	for ch := '0'; ch <= '9'; ch++ {
		m[keyForDigit(ch)] = string(ch)
	}
	names := []struct {
		key  ui.Key
		name string
	}{
		{ui.KeyF1, "F1"}, {ui.KeyF2, "F2"}, {ui.KeyF3, "F3"}, {ui.KeyF4, "F4"},
		{ui.KeyF5, "F5"}, {ui.KeyF6, "F6"}, {ui.KeyF7, "F7"}, {ui.KeyF8, "F8"},
		{ui.KeyF9, "F9"}, {ui.KeyF10, "F10"}, {ui.KeyF11, "F11"}, {ui.KeyF12, "F12"},
		{ui.KeySpace, "Space"}, {ui.KeyTab, "Tab"},
		{ui.KeyEnter, "Enter"}, {ui.KeyEscape, "Esc"},
		{ui.KeyDelete, "Delete"}, {ui.KeyBackspace, "Backspace"},
		{ui.KeyUp, "Up"}, {ui.KeyDown, "Down"},
		{ui.KeyLeft, "Left"}, {ui.KeyRight, "Right"},
		{ui.KeyMinus, "-"}, {ui.KeyEqual, "="},
		{ui.KeyComma, ","}, {ui.KeyPeriod, "."},
		{ui.KeySlash, "/"}, {ui.KeySemicolon, ";"},
		{ui.KeyQuote, "'"}, {ui.KeyBracketLeft, "["},
		{ui.KeyBracketRight, "]"}, {ui.KeyBackslash, "\\"},
		{ui.KeyBackquote, "`"},
	}
	for _, n := range names {
		m[n.key] = n.name
	}
	return m
}

// keyForLetter 返回字母键常量
func keyForLetter(ch rune) ui.Key {
	switch ch {
	case 'A':
		return ui.KeyA
	case 'B':
		return ui.KeyB
	case 'C':
		return ui.KeyC
	case 'D':
		return ui.KeyD
	case 'E':
		return ui.KeyE
	case 'F':
		return ui.KeyF
	case 'G':
		return ui.KeyG
	case 'H':
		return ui.KeyH
	case 'I':
		return ui.KeyI
	case 'J':
		return ui.KeyJ
	case 'K':
		return ui.KeyK
	case 'L':
		return ui.KeyL
	case 'M':
		return ui.KeyM
	case 'N':
		return ui.KeyN
	case 'O':
		return ui.KeyO
	case 'P':
		return ui.KeyP
	case 'Q':
		return ui.KeyQ
	case 'R':
		return ui.KeyR
	case 'S':
		return ui.KeyS
	case 'T':
		return ui.KeyT
	case 'U':
		return ui.KeyU
	case 'V':
		return ui.KeyV
	case 'W':
		return ui.KeyW
	case 'X':
		return ui.KeyX
	case 'Y':
		return ui.KeyY
	case 'Z':
		return ui.KeyZ
	}
	return ui.KeyUnknown
}

// keyForDigit 返回数字键常量
func keyForDigit(ch rune) ui.Key {
	switch ch {
	case '0':
		return ui.Key0
	case '1':
		return ui.Key1
	case '2':
		return ui.Key2
	case '3':
		return ui.Key3
	case '4':
		return ui.Key4
	case '5':
		return ui.Key5
	case '6':
		return ui.Key6
	case '7':
		return ui.Key7
	case '8':
		return ui.Key8
	case '9':
		return ui.Key9
	}
	return ui.KeyUnknown
}

// keyName 返回主键显示名
func keyName(key ui.Key) string {
	if n, ok := primaryKeyNames[key]; ok {
		return n
	}
	return fmt.Sprintf("Key(%d)", uint16(key))
}

// hotkeyManager 基于 mygo GlobalShortcut 的热键管理
type hotkeyManager struct {
	registered string // 当前已注册的热键串，空串表示未注册
}

// register 注册热键（自动替换旧热键；失败时尝试恢复旧热键）
func (h *hotkeyManager) register(acc string, callback func()) error {
	old := h.registered
	if old == acc {
		return nil
	}
	if old != "" {
		mygo.GlobalShortcut.Unregister(old)
		h.registered = ""
	}
	if err := mygo.GlobalShortcut.Register(acc, callback); err != nil {
		// 注册失败，尝试恢复旧热键
		if old != "" {
			if reErr := mygo.GlobalShortcut.Register(old, callback); reErr == nil {
				h.registered = old
			}
		}
		return fmt.Errorf("register hotkey %s: %w", acc, err)
	}
	h.registered = acc
	return nil
}

// unregister 注销当前热键
func (h *hotkeyManager) unregister() {
	if h.registered != "" {
		mygo.GlobalShortcut.Unregister(h.registered)
		h.registered = ""
	}
}
