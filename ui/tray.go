package ui

import (
	"audio-switch/internal/audio"
	"audio-switch/internal/config"
	"audio-switch/internal/logger"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
)

// Version 由构建脚本通过 ldflags 注入
var Version = "dev"

// TrayApp 管理系统托盘和相关功能
type TrayApp struct {
	audioAPI  audio.Audio
	cfg       *config.Config
	tray      *mygo.Tray
	trayIcon  []byte
	settings  *SettingsWindow
	hotkeys   hotkeyManager
	callback  func()
	quit      chan struct{}
	suppress  atomic.Bool // 录制期间阻止热键回调
	notifyOn  func() bool
}

// NewTrayApp 创建托盘应用
func NewTrayApp(a audio.Audio, cfg *config.Config, trayIcon []byte) *TrayApp {
	t := &TrayApp{
		audioAPI: a,
		cfg:      cfg,
		quit:     make(chan struct{}),
		trayIcon: trayIcon,
	}
	t.notifyOn = func() bool { return cfg.NotificationEnabled }
	return t
}

// Setup 初始化系统托盘与热键（App ready 后在主线程调用）
func (t *TrayApp) Setup() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon:           t.trayIcon,
		IconIsTemplate: true,
		ToolTip:        "Audio Switch",
		Menu:           t.buildMenu(),
	})
	if err != nil {
		logger.Error("Tray", "创建托盘失败", "error", err)
		return
	}
	t.tray = tray
	logger.Info("Tray", "托盘已创建")

	t.InitHotkey()

	// 启动设备热插拔检测
	go t.watchDevices()
}

// buildMenu 构建托盘菜单
func (t *TrayApp) buildMenu() *mygo.Menu {
	var items []*mygo.MenuItem

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
			items = append(items, &mygo.MenuItem{
				Label: label,
				Click: func(*mygo.MenuItem, *mygo.Window) { t.switchDevice(dev) },
			})
		}
	}

	if len(items) > 0 {
		items = append(items, mygo.Separator())
	}

	items = append(items,
		&mygo.MenuItem{
			Label: "快速切换 (A/B)",
			Click: func(*mygo.MenuItem, *mygo.Window) { t.QuickSwitch() },
		},
		&mygo.MenuItem{
			Label: "刷新设备",
			Click: func(*mygo.MenuItem, *mygo.Window) { t.RefreshMenu() },
		},
		mygo.Separator(),
		&mygo.MenuItem{
			Label: "偏好设置...",
			Click: func(*mygo.MenuItem, *mygo.Window) { t.ShowSettings() },
		},
		mygo.Separator(),
		&mygo.MenuItem{
			Label: "关于 Audio Switch",
			Click: func(*mygo.MenuItem, *mygo.Window) { t.showAbout() },
		},
		mygo.Separator(),
		&mygo.MenuItem{Role: mygo.RoleQuit},
	)

	return mygo.NewMenu(items)
}

// RefreshMenu 刷新托盘菜单
func (t *TrayApp) RefreshMenu() {
	if t.tray != nil {
		t.tray.SetMenu(t.buildMenu())
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
		t.sendNotification("切换失败", "无法获取音频设备列表")
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
		if currentID == t.cfg.Device1.ID || currentID == dev1Live {
			targetName = t.cfg.Device2.Name
			targetID = dev2Live
		} else {
			targetName = t.cfg.Device1.Name
			targetID = dev1Live
		}
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
		logger.Warn("Tray", "快速切换失败：找不到目标设备", "target", targetName)
		t.sendNotification("切换失败", "找不到设备: "+targetName+"，请在设置中重新选择")
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
		t.sendNotification("切换失败", switchErr.Error())
		return
	}

	logger.Info("Tray", "设备切换成功", "target", targetName)
	t.sendNotification("音频已切换", targetName)

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

// switchWithVolume 一次调用完成切换+音量设置
func (t *TrayApp) switchWithVolume(deviceID string, vol int) error {
	// 切换设备必须显式执行：SetDeviceVolume 只负责音量（macOS/Linux 实现不含切换）
	if err := t.audioAPI.SetDefaultDevice(deviceID); err != nil {
		return err
	}
	if vol >= 0 {
		if err := t.audioAPI.SetDeviceVolume(deviceID, vol); err != nil {
			// 音量设置失败不影响切换结果（HDMI 等设备音量由硬件控制，属正常情况）
			logger.Warn("Tray", "音量设置失败", "device", deviceID, "error", err)
		}
	}
	return nil
}

// ShowSettings 打开设置窗口（复用已有实例）
func (t *TrayApp) ShowSettings() {
	if err := t.ReloadConfig(); err != nil {
		logger.Warn("Tray", "重新加载配置失败", "error", err)
	}
	if t.settings == nil {
		t.settings = NewSettingsWindow(t.audioAPI, t.cfg, t)
	}
	t.settings.RefreshDevices()
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
	if err := t.hotkeys.register(t.cfg.Hotkey, t.onHotkey); err != nil {
		logger.Warn("Hotkey", "注册热键失败", "hotkey", t.cfg.Hotkey, "error", err)
		return
	}
	logger.Info("Hotkey", "热键已注册", "hotkey", t.cfg.Hotkey)
}

// onHotkey 全局热键回调（mygo 在主线程调用）
func (t *TrayApp) onHotkey() {
	if t.suppress.Load() {
		return
	}
	if t.callback != nil {
		t.callback()
	}
}

// UpdateHotkey 更新热键（设置界面调用）
func (t *TrayApp) UpdateHotkey(hotkeyStr string) error {
	return t.hotkeys.register(hotkeyStr, t.onHotkey)
}

// ClearHotkey 注销热键
func (t *TrayApp) ClearHotkey() {
	t.hotkeys.unregister()
}

// Cleanup 清理资源（退出时调用）
func (t *TrayApp) Cleanup() {
	select {
	case <-t.quit:
	default:
		close(t.quit)
	}
	t.hotkeys.unregister()
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
			mygo.RunOnMain(func() {
				t.RefreshMenu()
				if t.settings != nil {
					t.settings.RefreshDevices()
				}
			})
		}
		prevIDs = curIDs
	}
}

// showAbout 显示"关于"对话框（Dialog 可从任意 goroutine 调用）
func (t *TrayApp) showAbout() {
	go func() {
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Title:   "关于 Audio Switch",
			Message: "Audio Switch " + formatVersion(),
			Detail:  "跨平台音频输出设备快速切换工具\n" +
				"github.com/sggoodman/audio-switch",
		})
	}()
}

// formatVersion 显示版本号：优先取打包信息（mygo.json 的 version），
// 裸二进制回退到 ldflags 注入的 ui.Version；git describe 后缀截断
func formatVersion() string {
	ver := mygo.App.Version()
	if ver == "" {
		ver = Version
	}
	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}
	if idx := strings.Index(ver, "-"); idx > 0 {
		ver = ver[:idx]
	}
	return ver
}

// sendNotification 发送桌面通知（按配置开关）
func (t *TrayApp) sendNotification(title, body string) {
	if !t.notifyOn() {
		return
	}
	n := mygo.NewNotification(mygo.NotificationOptions{Title: title, Body: body})
	if err := n.Show(); err != nil {
		logger.Warn("Notify", "通知发送失败", "error", err)
	}
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
