package hotkey

import (
	"fmt"
	"strings"

	"golang.design/x/hotkey"
)

// HotkeyMgr 管理全局热键
type HotkeyMgr struct {
	hk       *hotkey.Hotkey
	keyStr   string
	callback func()
	quit     chan struct{}
}

// ParseHotkeyString 解析热键字符串（如 "Ctrl+Alt+S"）为 hotkey.Key 和修饰键
func ParseHotkeyString(s string) ([]hotkey.Modifier, hotkey.Key, error) {
	if s == "" {
		return nil, 0, fmt.Errorf("empty hotkey string")
	}

	parts := strings.Split(s, "+")
	if len(parts) < 2 {
		return nil, 0, fmt.Errorf("invalid hotkey format: %s (expected modifiers+key)", s)
	}

	var mods []hotkey.Modifier
	var key hotkey.Key

	for i, p := range parts {
		p = strings.TrimSpace(p)
		switch strings.ToLower(p) {
		case "ctrl", "control":
			mods = append(mods, hotkey.ModCtrl)
		case "alt":
			mods = append(mods, hotkey.ModAlt)
		case "shift":
			mods = append(mods, hotkey.ModShift)
		case "super", "win", "cmd", "command":
			mods = append(mods, hotkey.ModWin)
		default:
			if i != len(parts)-1 {
				return nil, 0, fmt.Errorf("unknown modifier: %s", p)
			}
			// 最后一个部分是主键
			k, err := parseKey(p)
			if err != nil {
				return nil, 0, err
			}
			key = k
		}
	}

	if key == 0 {
		return nil, 0, fmt.Errorf("no key specified in hotkey: %s", s)
	}

	return mods, key, nil
}

// Register 注册全局热键
func Register(hotkeyStr string, callback func()) (*HotkeyMgr, error) {
	mods, key, err := ParseHotkeyString(hotkeyStr)
	if err != nil {
		return nil, fmt.Errorf("parse hotkey: %w", err)
	}

	hk := hotkey.New(mods, key)
	mgr := &HotkeyMgr{
		hk:       hk,
		keyStr:   hotkeyStr,
		callback: callback,
		quit:     make(chan struct{}),
	}

	err = hk.Register()
	if err != nil {
		return nil, fmt.Errorf("register hotkey %s: %w", hotkeyStr, err)
	}

	// 在 goroutine 中监听热键
	go func() {
		for {
			select {
			case <-mgr.quit:
				return
			case <-hk.Keydown():
				if mgr.callback != nil {
					mgr.callback()
				}
			}
		}
	}()

	return mgr, nil
}

// Unregister 注销热键
func (m *HotkeyMgr) Unregister() {
	if m.hk != nil {
		m.hk.Unregister()
	}
	if m.quit != nil {
		close(m.quit)
	}
}

// String 返回热键描述
func (m *HotkeyMgr) String() string {
	return m.keyStr
}

// parseKey 解析单个按键名称
func parseKey(s string) (hotkey.Key, error) {
	switch strings.ToUpper(s) {
	case "A":
		return hotkey.KeyA, nil
	case "B":
		return hotkey.KeyB, nil
	case "C":
		return hotkey.KeyC, nil
	case "D":
		return hotkey.KeyD, nil
	case "E":
		return hotkey.KeyE, nil
	case "F":
		return hotkey.KeyF, nil
	case "G":
		return hotkey.KeyG, nil
	case "H":
		return hotkey.KeyH, nil
	case "I":
		return hotkey.KeyI, nil
	case "J":
		return hotkey.KeyJ, nil
	case "K":
		return hotkey.KeyK, nil
	case "L":
		return hotkey.KeyL, nil
	case "M":
		return hotkey.KeyM, nil
	case "N":
		return hotkey.KeyN, nil
	case "O":
		return hotkey.KeyO, nil
	case "P":
		return hotkey.KeyP, nil
	case "Q":
		return hotkey.KeyQ, nil
	case "R":
		return hotkey.KeyR, nil
	case "S":
		return hotkey.KeyS, nil
	case "T":
		return hotkey.KeyT, nil
	case "U":
		return hotkey.KeyU, nil
	case "V":
		return hotkey.KeyV, nil
	case "W":
		return hotkey.KeyW, nil
	case "X":
		return hotkey.KeyX, nil
	case "Y":
		return hotkey.KeyY, nil
	case "Z":
		return hotkey.KeyZ, nil
	case "0":
		return hotkey.Key0, nil
	case "1":
		return hotkey.Key1, nil
	case "2":
		return hotkey.Key2, nil
	case "3":
		return hotkey.Key3, nil
	case "4":
		return hotkey.Key4, nil
	case "5":
		return hotkey.Key5, nil
	case "6":
		return hotkey.Key6, nil
	case "7":
		return hotkey.Key7, nil
	case "8":
		return hotkey.Key8, nil
	case "9":
		return hotkey.Key9, nil
	case "F1":
		return hotkey.KeyF1, nil
	case "F2":
		return hotkey.KeyF2, nil
	case "F3":
		return hotkey.KeyF3, nil
	case "F4":
		return hotkey.KeyF4, nil
	case "F5":
		return hotkey.KeyF5, nil
	case "F6":
		return hotkey.KeyF6, nil
	case "F7":
		return hotkey.KeyF7, nil
	case "F8":
		return hotkey.KeyF8, nil
	case "F9":
		return hotkey.KeyF9, nil
	case "F10":
		return hotkey.KeyF10, nil
	case "F11":
		return hotkey.KeyF11, nil
	case "F12":
		return hotkey.KeyF12, nil
	case "SPACE":
		return hotkey.KeySpace, nil
	case "TAB":
		return hotkey.KeyTab, nil
	case "ENTER", "RETURN":
		return hotkey.KeyReturn, nil
	case "ESC", "ESCAPE":
		return hotkey.KeyEscape, nil
	case "DELETE":
		return hotkey.KeyDelete, nil
	case "UP":
		return hotkey.KeyUp, nil
	case "DOWN":
		return hotkey.KeyDown, nil
	case "LEFT":
		return hotkey.KeyLeft, nil
	case "RIGHT":
		return hotkey.KeyRight, nil
	default:
		return 0, fmt.Errorf("unsupported key: %s", s)
	}
}

// SupportedKeys 返回所有支持的主键名称列表
func SupportedKeys() []string {
	return []string{
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
		"K", "L", "M", "N", "O", "P", "Q", "R", "S", "T",
		"U", "V", "W", "X", "Y", "Z",
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"F1", "F2", "F3", "F4", "F5", "F6",
		"F7", "F8", "F9", "F10", "F11", "F12",
		"Space", "Tab", "Esc", "Delete",
		"Up", "Down", "Left", "Right",
	}
}

// SupportedModifiers 返回所有支持的修饰键名称列表
func SupportedModifiers() []string {
	return []string{"Ctrl", "Alt", "Shift"}
}
