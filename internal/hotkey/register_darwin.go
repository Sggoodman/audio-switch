//go:build darwin

package hotkey

import (
	"time"

	"golang.design/x/hotkey"

	"audio-switch/internal/logger"
)

// registerHotkey 将热键注册放到后台线程执行。
// golang.design/x/hotkey 的 darwin 实现内部会 dispatch_sync 到主队列：
// 若在主线程调用（如启动时 main goroutine 注册热键），libdispatch 的
// 死锁检测会直接 SIGTRAP 崩溃。放到后台线程后，主线程进入 Fyne 事件
// 循环时注册即可完成。
func registerHotkey(hk *hotkey.Hotkey) error {
	done := make(chan error, 1)
	go func() {
		done <- hk.Register()
	}()

	select {
	case err := <-done:
		// 主线程事件循环已在运行（如运行中更改热键），同步拿到结果
		return err
	case <-time.After(500 * time.Millisecond):
		// 主线程尚未进入事件循环（启动场景），放行启动流程，
		// 注册结果稍后异步记录日志
		go func() {
			if err := <-done; err != nil {
				logger.Warn("Hotkey", "热键注册失败", "error", err)
			} else {
				logger.Info("Hotkey", "热键注册完成")
			}
		}()
		return nil
	}
}
