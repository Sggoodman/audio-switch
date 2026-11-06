//go:build darwin

package hotkey

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#include <pthread.h>
#include <string.h>

static id keyMonitor = nil;
static pthread_mutex_t g_lock = PTHREAD_MUTEX_INITIALIZER;
static char g_combo[64];

// Esc 取消的哨兵值
#define RECORD_CANCEL "\x1b"

static void setCombo(const char *s) {
	pthread_mutex_lock(&g_lock);
	strlcpy(g_combo, s, sizeof(g_combo));
	pthread_mutex_unlock(&g_lock);
}
static void clearCombo(void) {
	pthread_mutex_lock(&g_lock);
	g_combo[0] = '\0';
	pthread_mutex_unlock(&g_lock);
}
static void copyCombo(char *out, size_t n) {
	pthread_mutex_lock(&g_lock);
	strlcpy(out, g_combo, n);
	pthread_mutex_unlock(&g_lock);
}

// modsToString 修饰键转字符串（顺序固定，与 ParseHotkeyString 兼容）
static NSString *modsToString(NSUInteger flags) {
	NSMutableArray *parts = [NSMutableArray array];
	if (flags & NSEventModifierFlagControl) [parts addObject:@"Ctrl"];
	if (flags & NSEventModifierFlagOption)   [parts addObject:@"Alt"];
	if (flags & NSEventModifierFlagShift)    [parts addObject:@"Shift"];
	if (flags & NSEventModifierFlagCommand)  [parts addObject:@"Cmd"];
	return [parts componentsJoinedByString:@"+"];
}

// keyName Carbon 键码转主键名（与 ParseHotkeyString 支持的名字一致），
// 不支持的键返回 nil
static NSString *keyName(CGKeyCode c) {
	static NSDictionary *m = nil;
	if (!m) {
		m = @{
			@(0x00): @"A", @(0x0B): @"B", @(0x08): @"C", @(0x02): @"D", @(0x0E): @"E",
			@(0x03): @"F", @(0x05): @"G", @(0x04): @"H", @(0x22): @"I", @(0x26): @"J",
			@(0x28): @"K", @(0x25): @"L", @(0x2E): @"M", @(0x2D): @"N", @(0x1F): @"O",
			@(0x23): @"P", @(0x0C): @"Q", @(0x0F): @"R", @(0x01): @"S", @(0x11): @"T",
			@(0x20): @"U", @(0x09): @"V", @(0x0D): @"W", @(0x07): @"X", @(0x10): @"Y",
			@(0x06): @"Z",
			@(0x12): @"1", @(0x13): @"2", @(0x14): @"3", @(0x15): @"4", @(0x17): @"5",
			@(0x16): @"6", @(0x1A): @"7", @(0x1C): @"8", @(0x19): @"9", @(0x1D): @"0",
			@(0x7A): @"F1",  @(0x78): @"F2",  @(0x63): @"F3",  @(0x76): @"F4",
			@(0x60): @"F5",  @(0x61): @"F6",  @(0x62): @"F7",  @(0x64): @"F8",
			@(0x65): @"F9",  @(0x6D): @"F10", @(0x67): @"F11", @(0x6F): @"F12",
			@(0x31): @"Space", @(0x30): @"Tab", @(0x24): @"Enter", @(0x33): @"Delete",
			@(0x7B): @"Left", @(0x7C): @"Right", @(0x7E): @"Up", @(0x7D): @"Down",
		};
	}
	return m[@(c)];
}

// startRecordMonitor 在主线程安装 keyDown 本地监听（本应用窗口内的按键，
// 无需辅助功能权限）。使用 dispatch_async 避免从主线程调用时死锁。
static void startRecordMonitor(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (keyMonitor != nil) return;
		keyMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskKeyDown handler:^NSEvent *(NSEvent *e) {
			// Esc(0x35) 取消录制
			if (e.keyCode == 0x35) {
				setCombo(RECORD_CANCEL);
				return nil;
			}
			NSString *kn = keyName(e.keyCode);
			NSString *ms = modsToString(e.modifierFlags);
			// 忽略无修饰键或不支持的键，继续等待有效组合
			if (kn == nil || ms.length == 0) return e;
			NSString *combo = [[ms stringByAppendingString:@"+"] stringByAppendingString:kn];
			setCombo(combo.UTF8String);
			return nil; // 吞掉已录制的按键
		}];
	});
}

// stopRecordMonitor 在主线程移除监听
static void stopRecordMonitor(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (keyMonitor != nil) {
			[NSEvent removeMonitor:keyMonitor];
			keyMonitor = nil;
		}
	});
}
*/
import "C"

import (
	"bytes"
	"time"
	"unsafe"
)

// RecordHotkey 录制快捷键：监听本应用窗口内的按键组合（设置窗口需保持焦点），
// 返回 "Ctrl+Alt+S" 格式；按 Esc 或关闭 quit 取消，返回空字符串。
func RecordHotkey(quit <-chan struct{}) string {
	C.clearCombo()
	C.startRecordMonitor()
	defer C.stopRecordMonitor()

	buf := make([]byte, 64)
	for {
		select {
		case <-quit:
			return ""
		default:
		}

		C.copyCombo((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
		if i := bytes.IndexByte(buf, 0); i >= 0 {
			combo := string(buf[:i])
			if combo == "\x1b" { // Esc 取消
				C.clearCombo()
				return ""
			}
			if combo != "" {
				C.clearCombo()
				// 等待用户松开修饰键，避免注册后残留按键立即触发回调
				time.Sleep(300 * time.Millisecond)
				return combo
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
}
