//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// hideDockIcon 将应用设为 accessory 激活策略：
// 不显示 Dock 图标，仅保留菜单栏托盘（LSUIElement 的运行时等价实现，
// 可防止 GLFW/Fyne 初始化把进程提升为 regular 应用）
static void hideDockIcon(void) {
	[NSApplication sharedApplication];
	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}
*/
import "C"

// hideDockIcon 隐藏 Dock 图标，仅保留菜单栏托盘（在 Fyne/GLFW 初始化前调用）
func hideDockIcon() {
	C.hideDockIcon()
}
