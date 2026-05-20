package ui

import (
	"audio-switch/internal/audio"
	"audio-switch/internal/config"
	"audio-switch/internal/hotkey"
	"audio-switch/internal/logger"
	"audio-switch/internal/notify"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Version 由构建脚本通过 ldflags 注入
var Version = "dev"

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
	quit      chan struct{}
	suppress  atomic.Bool // 录制期间阻止热键回调
}

// NewTrayApp 创建托盘应用
func NewTrayApp(app fyne.App, a audio.Audio, n notify.Notifier, cfg *config.Config) *TrayApp {
	t := &TrayApp{
		fyneApp:  app,
		audioAPI: a,
		notifier: n,
		cfg:      cfg,
		quit:     make(chan struct{}),
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

	// 启动设备热插拔检测
	go t.watchDevices()
}

// buildMenu 构建托盘菜单
func (t *TrayApp) buildMenu() *fyne.Menu {
	var items []*fyne.MenuItem

	devices, err := t.audioAPI.GetDevices()
	if err != nil {
		logger.Warn("Tray", "获取设备列表失败", "error", err)
	} else {
		for i := range devices {
			dev := devices[i]
			label := deviceIcon(dev.FormFactor) + " " + dev.Name
			if dev.IsDefault {
				label += " (当前)"
			}
			items = append(items, fyne.NewMenuItem(label, func() {
				t.switchDevice(dev)
			}))
		}
	}

	if len(items) > 0 {
		items = append(items, fyne.NewMenuItemSeparator())
	}

	items = append(items, fyne.NewMenuItem("快速切换 (A/B)", func() {
		t.QuickSwitch()
	}))

	items = append(items, fyne.NewMenuItem("刷新设备", func() {
		t.RefreshMenu()
	}))

	items = append(items, fyne.NewMenuItemSeparator())

	items = append(items, fyne.NewMenuItem("偏好设置...", func() {
		t.ShowSettings()
	}))

	items = append(items, fyne.NewMenuItemSeparator())

	items = append(items, fyne.NewMenuItem("关于 Audio Switch", func() {
		t.showAbout()
	}))

	items = append(items, fyne.NewMenuItemSeparator())

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
	if t.suppress.Load() {
		return
	}
	devices, err := t.audioAPI.GetDevices()
	if err != nil {
		logger.Warn("Tray", "快速切换获取设备失败", "error", err)
		_ = t.notifier.Send("切换失败", "无法获取音频设备列表")
		return
	}

	var currentID string
	for _, d := range devices {
		if d.IsDefault {
			currentID = d.ID
			break
		}
	}

	var targetName, targetID string
	if t.cfg.Device1 != nil && t.cfg.Device2 != nil {
		dev1Live := resolveDeviceID(devices, t.cfg.Device1)
		dev2Live := resolveDeviceID(devices, t.cfg.Device2)
		logger.Info("Tray", "快速切换调试",
			"currentID", currentID,
			"cfg1_id", t.cfg.Device1.ID, "cfg1_name", t.cfg.Device1.Name, "dev1Live", dev1Live,
			"cfg2_id", t.cfg.Device2.ID, "cfg2_name", t.cfg.Device2.Name, "dev2Live", dev2Live,
		)
		if currentID == t.cfg.Device1.ID || currentID == dev1Live {
			targetName = t.cfg.Device2.Name
			targetID = dev2Live
		} else {
			targetName = t.cfg.Device1.Name
			targetID = dev1Live
		}
		logger.Info("Tray", "快速切换目标", "targetName", targetName, "targetID", targetID)
	} else {
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
		logger.Warn("Tray", "快速切换失败：找不到目标设备", "targetName", targetName)
		_ = t.notifier.Send("切换失败", "找不到设备: "+targetName+"，请在设置中重新选择")
		return
	}

	vol := t.getVolumePreset(targetID)
	switchErr := t.switchWithVolume(targetID, vol)
	t.notifyAndRefresh(targetName, switchErr)
}

// switchDevice 切换到指定设备
func (t *TrayApp) switchDevice(dev audio.Device) {
	vol := t.getVolumePreset(dev.ID)
	switchErr := t.switchWithVolume(dev.ID, vol)
	t.notifyAndRefresh(dev.Name, switchErr)
}

// notifyAndRefresh 统一处理通知发送和 UI 刷新
func (t *TrayApp) notifyAndRefresh(targetName string, switchErr error) {
	if switchErr != nil {
		logger.Warn("Tray", "设备切换失败", "target", targetName, "error", switchErr)
		if t.cfg.NotificationEnabled {
			_ = t.notifier.Send("切换失败", switchErr.Error())
		}
		return
	}

	logger.Info("Tray", "设备切换成功", "target", targetName)
	if t.cfg.NotificationEnabled {
		_ = t.notifier.Send("音频已切换", targetName)
	}

	t.RefreshMenu()
	if t.settings != nil {
		t.settings.RefreshUI()
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

// switchWithVolume 一次调用完成切换+音量设置
func (t *TrayApp) switchWithVolume(deviceID string, vol int) error {
	if vol >= 0 {
		return t.audioAPI.SetDeviceVolume(deviceID, vol)
	}
	return t.audioAPI.SetDefaultDevice(deviceID)
}

// ShowSettings 打开设置窗口（复用已有实例）
func (t *TrayApp) ShowSettings() {
	if err := t.ReloadConfig(); err != nil {
		logger.Warn("Tray", "重新加载配置失败", "error", err)
	}
	if t.settings != nil {
		t.settings.RefreshUI()
		t.settings.Show()
		return
	}
	t.settings = NewSettingsWindow(t.fyneApp, t.audioAPI, t.cfg, t)
	t.settings.Show()
}

// ReloadConfig 从文件重新加载配置到 t.cfg
func (t *TrayApp) ReloadConfig() error {
	logger.Info("Tray", "开始重新加载配置...")
	newCfg, err := config.Load()
	if err != nil {
		logger.Warn("Tray", "加载配置失败", "error", err)
		return err
	}
	d1Vol, d2Vol := 0, 0
	if newCfg.Device1 != nil {
		d1Vol = newCfg.Device1.Volume
	}
	if newCfg.Device2 != nil {
		d2Vol = newCfg.Device2.Volume
	}
	logger.Info("Tray", "加载的配置", "device1_vol", d1Vol, "device2_vol", d2Vol)
	t.cfg.Device1 = newCfg.Device1
	t.cfg.Device2 = newCfg.Device2
	t.cfg.Hotkey = newCfg.Hotkey
	t.cfg.NotificationEnabled = newCfg.NotificationEnabled
	t.cfg.AutoStart = newCfg.AutoStart
	logger.Info("Tray", "配置已更新到内存")
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
		logger.Warn("Hotkey", "注册热键失败", "hotkey", t.cfg.Hotkey, "error", err)
		return
	}
	t.hotkeyMgr = mgr
	logger.Info("Hotkey", "热键已注册", "hotkey", t.cfg.Hotkey)
}

// UpdateHotkey 更新热键（设置界面调用）
func (t *TrayApp) UpdateHotkey(hotkeyStr string) error {
	oldHotkey := t.cfg.Hotkey
	oldMgr := t.hotkeyMgr

	// 先注销旧热键，避免注册新热键时立即捕获残留按键事件
	if oldMgr != nil {
		oldMgr.Unregister()
		t.hotkeyMgr = nil
	}

	mgr, err := hotkey.Register(hotkeyStr, t.callback)
	if err != nil {
		// 注册失败，尝试恢复旧热键
		if oldHotkey != "" {
			if reMgr, reErr := hotkey.Register(oldHotkey, t.callback); reErr == nil {
				t.hotkeyMgr = reMgr
			}
		}
		return err
	}

	t.hotkeyMgr = mgr
	t.cfg.Hotkey = hotkeyStr
	logger.Info("Hotkey", "热键已更新", "hotkey", hotkeyStr)
	return nil
}

// Cleanup 清理资源（退出时调用）
func (t *TrayApp) Cleanup() {
	close(t.quit)
	if t.hotkeyMgr != nil {
		t.hotkeyMgr.Unregister()
	}
}

// watchDevices 后台轮询检测设备热插拔
func (t *TrayApp) watchDevices() {
	var prevIDs string
	for {
		select {
		case <-t.quit:
			return
		case <-time.After(3 * time.Second):
		}

		devices, err := t.audioAPI.GetDevices()
		if err != nil {
			continue
		}

		ids := make([]string, len(devices))
		for i, d := range devices {
			ids[i] = d.ID
		}
		curIDs := strings.Join(ids, ",")

		if prevIDs != "" && prevIDs != curIDs {
			logger.Info("Tray", "检测到设备列表变化")
			t.RefreshMenu()
			if t.settings != nil {
				t.settings.RefreshUI()
			}
		}
		prevIDs = curIDs
	}
}

// aboutWin 复用的"关于"窗口
var aboutWin fyne.Window

// showAbout 显示"关于"对话框
func (t *TrayApp) showAbout() {
	ver := Version
	// 简化版本号显示：v1.0.2-1-gd602dbd-dirty → v1.0.2+dev
	if idx := strings.Index(ver, "-"); idx > 0 && ver[0] == 'v' {
		ver = ver[:idx]
	}

	repoURL, _ := url.Parse("https://github.com/sggoodman/audio-switch")
	link := widget.NewHyperlink("github.com/sggoodman/audio-switch", repoURL)

	content := container.NewVBox(
		widget.NewLabelWithStyle("Audio Switch", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("跨平台音频输出设备快速切换工具"),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("版本: %s", ver)),
		container.NewHBox(widget.NewLabel("项目主页:"), link),
	)

	if aboutWin == nil {
		aboutWin = t.fyneApp.NewWindow("关于 Audio Switch")
		aboutWin.SetContent(content)
		aboutWin.SetFixedSize(true)
		aboutWin.Resize(fyne.NewSize(350, 200))
		aboutWin.SetCloseIntercept(func() {
			aboutWin.Hide()
		})
	} else {
		aboutWin.SetContent(content)
	}
	aboutWin.Show()
	aboutWin.RequestFocus()
}

// resolveDeviceID 根据配置中的设备信息，在当前活跃设备列表中查找对应的设备 ID。
// 先精确匹配 ID，找不到再按名称匹配（设备重连后 ID 可能变化）。
func resolveDeviceID(devices []audio.Device, cfg *config.DeviceConfig) string {
	if cfg == nil {
		return ""
	}
	for _, d := range devices {
		if d.ID == cfg.ID {
			return d.ID
		}
	}
	for _, d := range devices {
		if d.Name == cfg.Name {
			return d.ID
		}
	}
	return ""
}

// deviceIcon 根据 FormFactor 返回设备类型图标前缀
func deviceIcon(f audio.FormFactor) string {
	switch f {
	case audio.FormFactorSpeakers:
		return "🔊"
	case audio.FormFactorHeadphones, audio.FormFactorHeadset:
		return "🎧"
	case audio.FormFactorHDMI, audio.FormFactorDisplay:
		return "🖥"
	default:
		return "🔉"
	}
}
