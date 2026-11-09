package ui

import (
	"audio-switch/internal/audio"
	"audio-switch/internal/config"
	"audio-switch/internal/logger"
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// SettingsWindow 偏好设置窗口（mygo native UI）
type SettingsWindow struct {
	tray     *TrayApp
	audioAPI audio.Audio
	cfg      *config.Config
	win      *mygo.Window

	devices []audio.Device
	devNames []string         // select 选项（含空串 = 未选择）
	devMap   map[string]string // 设备名 → 设备 ID

	// 控件绑定值（view 每帧读写）
	devA, devB       string
	volA, volB       float64
	notifyChecked    bool
	autoChecked      bool
	statusMsg        string // 操作结果提示（注册失败等）
	openStates       []bool // 设备手风琴展开状态

	// 热键录制状态
	recording   bool
	pendingHK   string // 已捕获的热键串，等待按键松开后注册
	waitRelease bool

	saveTimer *time.Timer
}

// NewSettingsWindow 创建设置窗口（懒创建，关闭时隐藏复用）
func NewSettingsWindow(a audio.Audio, cfg *config.Config, tray *TrayApp) *SettingsWindow {
	s := &SettingsWindow{
		audioAPI: a,
		cfg:      cfg,
		tray:     tray,
	}
	s.reloadDevices()

	s.win = mygo.NewWindow(mygo.WindowOptions{
		Title:   "音频输出设置",
		Width:   520,
		Height:  560,
		Content: ui.View(s.view),
	})
	// 关闭窗口 = 隐藏，保留实例下次复用
	s.win.OnClose(func(e *mygo.CloseEvent) {
		if s.recording {
			s.cancelRecording()
		}
		e.PreventDefault()
		s.win.Hide()
	})
	return s
}

// Show 显示设置窗口
func (s *SettingsWindow) Show() {
	s.reloadDevices()
	s.syncFromConfig()
	s.win.Show()
	s.win.Focus()
}

// RefreshDevices 设备列表或配置变化后刷新
func (s *SettingsWindow) RefreshDevices() {
	s.reloadDevices()
	s.syncFromConfig()
	if s.win != nil {
		s.win.Invalidate()
	}
}

// reloadDevices 重新拉取设备列表
func (s *SettingsWindow) reloadDevices() {
	devices, err := s.audioAPI.GetDevices()
	if err != nil {
		logger.Warn("Settings", "获取设备列表失败", "error", err)
	}
	s.devices = devices

	s.devNames = []string{""}
	s.devMap = map[string]string{}
	for _, d := range devices {
		s.devNames = append(s.devNames, d.Name)
		s.devMap[d.Name] = d.ID
	}

	// 手风琴展开状态随设备数重建，默认展开第一个
	s.openStates = make([]bool, len(devices))
	if len(s.openStates) > 0 {
		s.openStates[0] = true
	}
}

// syncFromConfig 把配置同步到控件绑定值
func (s *SettingsWindow) syncFromConfig() {
	s.devA, s.devB = "", ""
	s.volA, s.volB = 40, 40
	if s.cfg.Device1 != nil {
		s.devA = s.cfg.Device1.Name
		if s.cfg.Device1.Volume > 0 {
			s.volA = float64(s.cfg.Device1.Volume)
		}
	}
	if s.cfg.Device2 != nil {
		s.devB = s.cfg.Device2.Name
		if s.cfg.Device2.Volume > 0 {
			s.volB = float64(s.cfg.Device2.Volume)
		}
	}
	s.notifyChecked = s.cfg.NotificationEnabled
	s.autoChecked = s.cfg.AutoStart
	s.statusMsg = ""
}

// logConfig 打印当前配置到日志
func (s *SettingsWindow) logConfig(action string) {
	d1Name, d1Vol, d2Name, d2Vol := "<nil>", 0, "<nil>", 0
	if s.cfg.Device1 != nil {
		d1Name = s.cfg.Device1.Name
		d1Vol = s.cfg.Device1.Volume
	}
	if s.cfg.Device2 != nil {
		d2Name = s.cfg.Device2.Name
		d2Vol = s.cfg.Device2.Volume
	}
	logger.Info("Settings", action, "device1_name", d1Name, "device1_vol", d1Vol, "device2_name", d2Name, "device2_vol", d2Vol)
}

// view 构建设置界面（每帧调用）
func (s *SettingsWindow) view(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		// 内容可滚动，内边距交给各区块自己控制
		scroll := ui.Scroll(c)
		scroll.Grow(1).FillWidth().Children(func() { s.buildSections(c) })
		s.buildStatusBar(c)
	})
}

// buildSections 各设置区块
func (s *SettingsWindow) buildSections(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(16).Gap(12).Children(func() {
		s.buildQuickSwitchSection(c)
		ui.Divider(c)
		s.buildHotkeySection(c)
		ui.Divider(c)
		s.buildDeviceListSection(c)
		ui.Divider(c)
		s.buildActionButtons(c)
	})
}

// sectionTitle 区块标题
func sectionTitle(c *ui.Context, title string) {
	ui.Text(c, title).Bold().FontSize(15)
}

// buildQuickSwitchSection 快捷切换设置区域
func (s *SettingsWindow) buildQuickSwitchSection(c *ui.Context) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		sectionTitle(c, "快捷切换设置")

		ui.Row(c).Gap(12).Children(func() {
			s.buildDeviceSelect(c, "设备 A:", &s.devA, "A")
			s.buildDeviceSelect(c, "设备 B:", &s.devB, "B")
		})
		ui.Row(c).Gap(12).Children(func() {
			s.buildVolumeSlider(c, "A 音量:", &s.volA, "A")
			s.buildVolumeSlider(c, "B 音量:", &s.volB, "B")
		})
		ui.Row(c).Gap(24).Children(func() {
			if ui.Checkbox(c, &s.notifyChecked, "切换时弹出通知").Changed() {
				s.cfg.NotificationEnabled = s.notifyChecked
				s.saveConfig()
			}
			if ui.Checkbox(c, &s.autoChecked, "开机自动启动").Changed() {
				s.toggleAutoStart()
			}
		})
	})
}

// buildDeviceSelect 设备下拉选择
func (s *SettingsWindow) buildDeviceSelect(c *ui.Context, label string, selected *string, slot string) {
	ui.Column(c).Grow(1).Gap(4).Children(func() {
		ui.Text(c, label)
		chosen := *selected
		ui.Select(c, &chosen, s.devNames).OnChange(func() {
			*selected = chosen
			s.onDeviceSelected(slot, chosen)
		})
	})
}

// onDeviceSelected 设备选择变更
func (s *SettingsWindow) onDeviceSelected(slot, name string) {
	cfgField := &s.cfg.Device1
	if slot == "B" {
		cfgField = &s.cfg.Device2
	}
	if name == "" {
		*cfgField = nil
	} else {
		oldVol := 40
		if *cfgField != nil && (*cfgField).Volume > 0 {
			oldVol = (*cfgField).Volume
		}
		*cfgField = &config.DeviceConfig{
			ID:     s.devMap[name],
			Name:   name,
			Volume: oldVol,
		}
		logger.Info("Settings", "设备选择变更", "device", slot, "name", name, "vol", oldVol)
	}
	s.saveConfig()
}

// buildVolumeSlider 音量滑块
func (s *SettingsWindow) buildVolumeSlider(c *ui.Context, label string, value *float64, slot string) {
	ui.Column(c).Grow(1).Gap(4).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, label)
			ui.Spacer(c).Grow(1)
			ui.Text(c, fmt.Sprintf("%.0f%%", *value)).TextColor(c.Theme().TextMuted)
		})
		ui.Slider(c, value, 0, 100).OnChange(func() {
			s.onVolumeChanged(slot, *value)
		})
	})
}

// onVolumeChanged 音量滑块变更（防抖保存）
func (s *SettingsWindow) onVolumeChanged(slot string, v float64) {
	cfgField := s.cfg.Device1
	if slot == "B" {
		cfgField = s.cfg.Device2
	}
	if cfgField == nil {
		return
	}
	cfgField.Volume = int(v)
	logger.Debug("Settings", "音量滑块变更", "device", slot, "value", int(v))
	s.debouncedSave()
}

// toggleAutoStart 切换开机自启
func (s *SettingsWindow) toggleAutoStart() {
	if err := mygo.App.SetOpenAtLogin(s.autoChecked); err != nil {
		logger.Warn("Settings", "设置开机自启失败", "enabled", s.autoChecked, "error", err)
		s.statusMsg = "设置开机自启失败: " + err.Error()
		s.autoChecked = !s.autoChecked // 回滚显示
		return
	}
	s.cfg.AutoStart = s.autoChecked
	s.saveConfig()
}

// buildHotkeySection 热键设置区域
func (s *SettingsWindow) buildHotkeySection(c *ui.Context) {
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		sectionTitle(c, "快捷键")
		ui.Row(c).Gap(8).Children(func() {
			display := s.cfg.Hotkey
			if s.recording {
				if s.pendingHK != "" {
					display = s.pendingHK + "（松开按键生效）"
				} else {
					display = "请按下组合键…（Esc 取消）"
				}
			} else if display == "" {
				display = "未设置"
			}
			ui.Text(c, display)
			ui.Spacer(c).Grow(1)
			if !s.recording {
				if ui.Button(c, "录制快捷键").Clicked() {
					s.startRecording()
				}
			} else if ui.Button(c, "取消录制").Clicked() {
				s.cancelRecording()
			}
			if ui.Button(c, "清除").Clicked() {
				s.clearHotkey()
			}
		})
		if s.recording {
			s.captureKeys(c)
		}
	})
}

// startRecording 开始录制快捷键
func (s *SettingsWindow) startRecording() {
	logger.Info("Settings", "开始录制快捷键", "currentHotkey", s.cfg.Hotkey)
	s.tray.suppress.Store(true)
	s.tray.ClearHotkey()
	s.recording = true
	s.pendingHK = ""
	s.waitRelease = false
	s.statusMsg = ""
}

// cancelRecording 取消录制并恢复原热键
func (s *SettingsWindow) cancelRecording() {
	s.recording = false
	s.pendingHK = ""
	s.waitRelease = false
	s.tray.suppress.Store(false)
	s.restoreHotkey()
}

// clearHotkey 清除热键
func (s *SettingsWindow) clearHotkey() {
	if s.recording {
		s.cancelRecording()
	}
	s.tray.ClearHotkey()
	s.cfg.Hotkey = ""
	s.saveConfig()
}

// restoreHotkey 恢复配置中保存的热键
func (s *SettingsWindow) restoreHotkey() {
	if s.cfg.Hotkey == "" {
		return
	}
	if err := s.tray.UpdateHotkey(s.cfg.Hotkey); err != nil {
		logger.Warn("Settings", "恢复热键失败", "hotkey", s.cfg.Hotkey, "error", err)
	}
}

// captureKeys 录制期间捕获按键组合（view 每帧调用）
func (s *SettingsWindow) captureKeys(c *ui.Context) {
	mods, keys := c.Modifiers(), c.KeysDown()

	// Esc 取消录制
	if c.Shortcut(0, ui.KeyEscape) {
		s.cancelRecording()
		return
	}

	if !s.waitRelease {
		// 寻找主键（KeysDown 不含修饰键）
		for _, k := range keys {
			if hk, ok := hotkeyFromKeys(mods, k); ok {
				s.pendingHK = hk
				s.waitRelease = true
				logger.Debug("Settings", "捕获热键组合", "hotkey", hk)
				break
			}
		}
		return
	}

	// 已捕获，等待所有键松开后注册（避免注册的瞬间按键仍在按住而误触发）
	if len(keys) == 0 && mods == 0 {
		hk := s.pendingHK
		s.recording = false
		s.pendingHK = ""
		s.waitRelease = false
		s.tray.suppress.Store(false)
		if err := s.tray.UpdateHotkey(hk); err != nil {
			logger.Warn("Settings", "注册热键失败", "hotkey", hk, "error", err)
			s.statusMsg = "注册失败: " + err.Error()
			s.restoreHotkey()
			return
		}
		logger.Info("Settings", "热键已录制", "hotkey", hk)
		s.cfg.Hotkey = hk
		s.saveConfig()
	}
}

// buildDeviceListSection 设备列表区域
func (s *SettingsWindow) buildDeviceListSection(c *ui.Context) {
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		sectionTitle(c, "音频输出设备")
		if len(s.devices) == 0 {
			ui.Text(c, "未检测到输出设备").TextColor(c.Theme().TextMuted)
			return
		}
		ui.Accordion(c, func() {
			for i := range s.devices {
				dev := s.devices[i]
				title := deviceIcon(dev.FormFactor) + " " + dev.Name
				if dev.IsDefault {
					title += " (当前)"
				}
				// 展开状态存于 s.openStates，指针稳定即可
				ui.AccordionItem(c, title, &s.openStates[i], func() {
					if ui.Button(c, "切换到此设备").Clicked() {
						vol := s.tray.getVolumePreset(dev.ID)
						if err := s.tray.switchWithVolume(dev.ID, vol); err != nil {
							s.statusMsg = "切换失败: " + err.Error()
							return
						}
						s.statusMsg = "已切换到 " + dev.Name
						s.RefreshDevices()
						s.tray.RefreshMenu()
					}
				})
			}
		})
	})
}

// buildActionButtons 操作按钮
func (s *SettingsWindow) buildActionButtons(c *ui.Context) {
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c, "快速切换").Clicked() {
			s.tray.QuickSwitch()
			s.RefreshDevices()
		}
		if ui.Button(c, "刷新").Clicked() {
			s.RefreshDevices()
			s.tray.RefreshMenu()
		}
	})
}

// buildStatusBar 底部状态栏
func (s *SettingsWindow) buildStatusBar(c *ui.Context) {
	ui.Box(c).FillWidth().Padding(8, 12).Children(func() {
		msg := s.statusMsg
		if msg == "" {
			msg = "配置保存于 " + config.GetConfigPath()
		}
		ui.Text(c, msg).FontSize(12).TextColor(c.Theme().TextMuted)
	})
}

// saveConfig 立即保存配置
func (s *SettingsWindow) saveConfig() {
	d1Vol, d2Vol := 0, 0
	if s.cfg.Device1 != nil {
		d1Vol = s.cfg.Device1.Volume
	}
	if s.cfg.Device2 != nil {
		d2Vol = s.cfg.Device2.Volume
	}
	logger.Info("Settings", "保存配置", "device1_vol", d1Vol, "device2_vol", d2Vol)

	if err := config.Save(s.cfg); err != nil {
		logger.Warn("Settings", "保存配置失败", "error", err)
		s.statusMsg = "保存配置失败: " + err.Error()
		return
	}
	logger.Info("Settings", "配置已保存", "path", config.GetConfigPath())
}

// debouncedSave 防抖保存，300ms 内只触发一次写盘
func (s *SettingsWindow) debouncedSave() {
	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}
	s.saveTimer = time.AfterFunc(300*time.Millisecond, func() {
		mygo.RunOnMain(func() { s.saveConfig() })
	})
}
