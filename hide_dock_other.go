//go:build !darwin

package main

// hideDockIcon 非 darwin 平台无需处理（Windows 托盘应用本就不在任务栏常驻）
func hideDockIcon() {}
