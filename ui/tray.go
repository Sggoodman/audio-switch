package ui

import (
	"audio-switch/internal/audio"
	"audio-switch/internal/config"
	"audio-switch/internal/hotkey"
	"audio-switch/internal/notify"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// TrayApp 管理系统托盘和相关功能
type TrayApp struct {
	fyneApp   fyne.App
	audioAPI  audio.Audio
	notifier  notify.Notifier
	cfg       *config.Config
	desk      desktop.App
	settings  *SettingsWindow
	hotkeyMgr *hotkey.HotkeyMgr
	callback  func()
}

// NewTrayApp 创建托盘应用
func NewTrayApp(app fyne.App, a audio.Audio, n notify.Notifier, cfg *config.Config) *TrayApp {
	t := &TrayApp{
		fyneApp:  app,
		audioAPI: a,
		notifier: n,
		cfg:      cfg,
	}
	if desk, ok := app.(desktop.App); ok {
		t.desk = desk
	}
	return t
}

// Setup 初始化系统托盘
func (t *TrayApp) Setup() {
	if t.desk == nil {
		return
	}
	t.desk.SetSystemTrayMenu(t.buildMenu())
}

// buildMenu 构建托盘菜单
func (t *TrayApp) buildMenu() *fyne.Menu {
	var items []*fyne.MenuItem

	// 设备列表
	devices, err := t.audioAPI.GetDevices()
	if err == nil {
		for _, dev := range devices {
			dev := dev // capture loop var
			label := dev.Name
			if dev.IsDefault {
				label = "√ " + label + " (当前)"
			}
			items = append(items, fyne.NewMenuItem(label, func() {
				t.switchDevice(dev)
			}))
		}
	}

	if len(items) > 0 {
		items = append(items, fyne.NewMenuItemSeparator())
	}

	// 快速切换
	items = append(items, fyne.NewMenuItem("快速切换 (A/B)", func() {
		t.QuickSwitch()
	}))

	// 刷新设备
	items = append(items, fyne.NewMenuItem("刷新设备", func() {
		t.RefreshMenu()
	}))

	items = append(items, fyne.NewMenuItemSeparator())

	// 偏好设置
	items = append(items, fyne.NewMenuItem("偏好设置...", func() {
		t.ShowSettings()
	}))

	items = append(items, fyne.NewMenuItemSeparator())

	// 退出
	items = append(items, fyne.NewMenuItem("退出", func() {
		t.fyneApp.Quit()
	}))

	return fyne.NewMenu("Audio Switch", items...)
}

// RefreshMenu 刷新托盘菜单
func (t *TrayApp) RefreshMenu() {
	if t.desk != nil {
		t.desk.SetSystemTrayMenu(t.buildMenu())
	}
}

// QuickSwitch 在偏好设备之间快速切换
func (t *TrayApp) QuickSwitch() {
	devices, err := t.audioAPI.GetDevices()
	if err != nil {
		return
	}

	// 找到当前默认设备
	var currentID string
	for _, d := range devices {
		if d.IsDefault {
			currentID = d.ID
			break
		}
	}

	// 确定目标设备
	var targetName string
	var targetID string

	if t.cfg.Device1 != nil && t.cfg.Device2 != nil {
		// 两个偏好设备间切换
		if currentID == t.cfg.Device1.ID {
			targetID = t.cfg.Device2.ID
			targetName = t.cfg.Device2.Name
		} else {
			targetID = t.cfg.Device1.ID
			targetName = t.cfg.Device1.Name
		}
	} else {
		// 无偏好设备，循环切换
		for i, d := range devices {
			if d.IsDefault {
				next := (i + 1) % len(devices)
				targetID = devices[next].ID
				targetName = devices[next].Name
				break
			}
		}
	}

	if targetID == "" {
		return
	}

	// 一次 exe 调用完成切换 + 音量设置
	vol := t.getVolumePreset(targetID)
	if err := t.switchWithVolume(targetID, vol); err != nil {
		if t.cfg.NotificationEnabled {
			_ = t.notifier.Send("切换失败", err.Error())
		}
		return
	}

	if t.cfg.NotificationEnabled {
		_ = t.notifier.Send("音频已切换", targetName)
	}

	t.RefreshMenu()
	if t.settings != nil {
		t.settings.RefreshDevices()
	}
}

// switchDevice 切换到指定设备
func (t *TrayApp) switchDevice(dev audio.Device) {
	vol := t.getVolumePreset(dev.ID)
	if err := t.switchWithVolume(dev.ID, vol); err != nil {
		if t.cfg.NotificationEnabled {
			_ = t.notifier.Send("切换失败", err.Error())
		}
		return
	}

	if t.cfg.NotificationEnabled {
		_ = t.notifier.Send("音频已切换", dev.Name)
	}

	t.RefreshMenu()
	if t.settings != nil {
		t.settings.RefreshDevices()
	}
}

// getVolumePreset 返回设备的音量预设，无预设返回 -1
func (t *TrayApp) getVolumePreset(deviceID string) int {
	if t.cfg.Device1 != nil && t.cfg.Device1.ID == deviceID && t.cfg.Device1.Volume > 0 {
		return t.cfg.Device1.Volume
	}
	if t.cfg.Device2 != nil && t.cfg.Device2.ID == deviceID && t.cfg.Device2.Volume > 0 {
		return t.cfg.Device2.Volume
	}
	return -1
}

// switchWithVolume 一次调用完成切换+音量设置。
// vol < 0 仅切换，0-100 切换并设音量。
func (t *TrayApp) switchWithVolume(deviceID string, vol int) error {
	if vol >= 0 {
		return t.audioAPI.SetDeviceVolume(deviceID, vol)
	}
	return t.audioAPI.SetDefaultDevice(deviceID)
}

// ShowSettings 打开设置窗口
func (t *TrayApp) ShowSettings() {
	// 每次打开设置窗口时重新加载配置，确保显示最新值
	if err := t.ReloadConfig(); err != nil {
		log.Printf("重新加载配置失败: %v", err)
	}
	// 每次都创建新的设置窗口，确保 UI 显示最新配置值
	t.settings = NewSettingsWindow(t.fyneApp, t.audioAPI, t.cfg, t)
	t.settings.Show()
}

// ReloadConfig 从文件重新加载配置到 t.cfg
func (t *TrayApp) ReloadConfig() error {
	log.Printf("[Tray] 开始重新加载配置...")
	newCfg, err := config.Load()
	if err != nil {
		log.Printf("[Tray] 加载配置失败: %v", err)
		return err
	}
	log.Printf("[Tray] 加载的配置: Device1 vol=%d, Device2 vol=%d",
		func() int {
			if newCfg.Device1 != nil {
				return newCfg.Device1.Volume
			}
			return 0
		}(),
		func() int {
			if newCfg.Device2 != nil {
				return newCfg.Device2.Volume
			}
			return 0
		}())
	// 更新所有配置字段
	t.cfg.Device1 = newCfg.Device1
	t.cfg.Device2 = newCfg.Device2
	t.cfg.Hotkey = newCfg.Hotkey
	t.cfg.NotificationEnabled = newCfg.NotificationEnabled
	t.cfg.AutoStart = newCfg.AutoStart
	log.Printf("[Tray] 配置已更新到 t.cfg")
	return nil
}

// InitHotkey 初始化热键（启动时调用）
func (t *TrayApp) InitHotkey() {
	t.callback = t.QuickSwitch
	if t.cfg.Hotkey == "" {
		return
	}
	mgr, err := hotkey.Register(t.cfg.Hotkey, t.callback)
	if err != nil {
		log.Printf("注册热键 %s 失败: %v", t.cfg.Hotkey, err)
		return
	}
	t.hotkeyMgr = mgr
	log.Printf("热键 %s 已注册", t.cfg.Hotkey)
}

// UpdateHotkey 更新热键（设置界面调用）
func (t *TrayApp) UpdateHotkey(hotkeyStr string) error {
	// 先尝试注册新热键
	mgr, err := hotkey.Register(hotkeyStr, t.callback)
	if err != nil {
		return err
	}

	// 注册成功，注销旧热键
	if t.hotkeyMgr != nil {
		t.hotkeyMgr.Unregister()
	}

	t.hotkeyMgr = mgr
	t.cfg.Hotkey = hotkeyStr
	log.Printf("热键已更新为 %s", hotkeyStr)
	return nil
}

// Cleanup 清理资源（退出时调用）
func (t *TrayApp) Cleanup() {
	if t.hotkeyMgr != nil {
		t.hotkeyMgr.Unregister()
	}
}
