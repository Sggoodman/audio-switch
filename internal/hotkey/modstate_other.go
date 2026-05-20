//go:build !windows

package hotkey

// HeldModifiers 返回当前按住的修饰键列表（非 Windows 暂不支持）
func HeldModifiers() []string {
	return nil
}
