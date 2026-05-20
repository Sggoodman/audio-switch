//go:build !windows

package hotkey

// RecordHotkey 非 Windows 平台暂不支持录制
func RecordHotkey(quit <-chan struct{}) string {
	<-quit
	return ""
}
