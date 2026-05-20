package ui

import (
	"audio-switch/internal/audio"
	"audio-switch/internal/autostart"
	"audio-switch/internal/config"
	"audio-switch/internal/hotkey"
	"audio-switch/internal/logger"
	"audio-switch/internal/util"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// SettingsWindow 偏好设置窗口
type SettingsWindow struct {
	fyneApp        fyne.App
	win            fyne.Window
	audioAPI       audio.Audio
	cfg            *config.Config
	tray           *TrayApp
	devices        []audio.Device
	autostartMgr   autostart.Manager
	uiReady        bool
	saveTimer      *time.Timer
	cancelRecording func()
}

// NewSettingsWindow 创建设置窗口
func NewSettingsWindow(app fyne.App, a audio.Audio, cfg *config.Config, tray *TrayApp) *SettingsWindow {
	s := &SettingsWindow{
		fyneApp:      app,
		audioAPI:     a,
		cfg:          cfg,
		tray:         tray,
		autostartMgr: autostart.New(),
	}

	devices, err := a.GetDevices()
	if err != nil {
		logger.Warn("Settings", "获取设备列表失败", "error", err)
	}
	s.devices = devices

	s.logConfig("创建窗口")
	s.win = app.NewWindow("音频输出设置")
	s.win.SetContent(s.buildUI())
	s.win.Resize(fyne.NewSize(500, 450))
	s.win.SetCloseIntercept(func() {
		if s.cancelRecording != nil {
			s.cancelRecording()
		}
		s.win.Hide()
	})

	return s
}

// Show 显示设置窗口
func (s *SettingsWindow) Show() {
	s.uiReady = false
	s.win.Show()
	s.win.RequestFocus()
	go func() {
		time.Sleep(500 * time.Millisecond)
		s.uiReady = true
		logger.Debug("Settings", "UI 初始化完成，滑块回调已启用")
	}()
}

// RefreshUI 重建窗口内容，用于配置变更后刷新
func (s *SettingsWindow) RefreshUI() {
	if s.cancelRecording != nil {
		s.cancelRecording()
	}
	devices, err := s.audioAPI.GetDevices()
	if err != nil {
		logger.Warn("Settings", "刷新设备列表失败", "error", err)
	}
	s.devices = devices
	s.win.SetContent(s.buildUI())
	s.uiReady = false
	go func() {
		time.Sleep(500 * time.Millisecond)
		s.uiReady = true
	}()
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

// buildUI 构建设置界面
func (s *SettingsWindow) buildUI() *fyne.Container {
	return container.NewVBox(
		s.buildQuickSwitchSection(),
		widget.NewSeparator(),
		s.buildDeviceListSection(),
		widget.NewSeparator(),
		s.buildActionButtons(),
	)
}

// buildQuickSwitchSection 构建快捷切换设置区域
func (s *SettingsWindow) buildQuickSwitchSection() *fyne.Container {
	deviceNames := []string{""}
	deviceMap := map[string]string{}
	for _, d := range s.devices {
		deviceNames = append(deviceNames, d.Name)
		deviceMap[d.Name] = d.ID
	}

	dev1Select := s.buildDeviceSelect(deviceNames, deviceMap, "A", s.cfg.Device1)
	dev2Select := s.buildDeviceSelect(deviceNames, deviceMap, "B", s.cfg.Device2)

	vol1Slider, _ := s.buildVolumeSlider("A", s.cfg.Device1)
	vol2Slider, _ := s.buildVolumeSlider("B", s.cfg.Device2)

	notifyCheck := widget.NewCheck("切换时弹出通知", func(checked bool) {
		s.cfg.NotificationEnabled = checked
		s.saveConfig()
	})
	notifyCheck.SetChecked(s.cfg.NotificationEnabled)

	return container.NewVBox(
		widget.NewLabelWithStyle("快捷切换设置", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("设备 A:"), dev1Select),
			container.NewVBox(widget.NewLabel("设备 B:"), dev2Select),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("A 音量:"), vol1Slider),
			container.NewVBox(widget.NewLabel("B 音量:"), vol2Slider),
		),
		container.NewGridWithColumns(2, notifyCheck, s.buildAutoStartSection()),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("快捷键", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		s.buildHotkeySection(),
	)
}

// buildDeviceSelect 构建设备选择下拉框
func (s *SettingsWindow) buildDeviceSelect(deviceNames []string, deviceMap map[string]string, label string, cfgDev *config.DeviceConfig) *widget.Select {
	selectedName := ""
	if cfgDev != nil {
		selectedName = cfgDev.Name
	}

	initialized := false
	sel := widget.NewSelect(deviceNames, func(name string) {
		if !initialized {
			return
		}

		cfgField := &s.cfg.Device1
		if label == "B" {
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
				ID:     deviceMap[name],
				Name:   name,
				Volume: oldVol,
			}
			logger.Info("Settings", "设备选择变更", "device", label, "name", name, "vol", oldVol)
			s.saveConfig()
		}
	})
	sel.PlaceHolder = "选择设备 " + label
	sel.SetSelected(selectedName)
	initialized = true
	return sel
}

// buildVolumeSlider 构建音量滑块，返回容器和标签（用于外部更新）。
func (s *SettingsWindow) buildVolumeSlider(label string, cfgDev *config.DeviceConfig) (*fyne.Container, *widget.Label) {
	vol := 40
	if cfgDev != nil && cfgDev.Volume > 0 {
		vol = cfgDev.Volume
	}
	logger.Debug("Settings", "音量滑块初始化", "device", label, "vol", vol)

	pctLabel := widget.NewLabel(formatPercent(vol))
	slider := widget.NewSlider(0, 100)
	slider.SetValue(float64(vol))
	slider.OnChanged = func(v float64) {
		pctLabel.SetText(formatPercent(int(v)))

		cfgField := s.cfg.Device1
		if label == "B" {
			cfgField = s.cfg.Device2
		}
		if cfgField != nil && s.uiReady {
			logger.Debug("Settings", "音量滑块变更", "device", label, "value", v)
			cfgField.Volume = int(v)
			s.debouncedSave()
		}
	}

	return container.NewBorder(nil, nil, nil, pctLabel, slider), pctLabel
}

// buildAutoStartSection 构建开机自启区域
func (s *SettingsWindow) buildAutoStartSection() *widget.Check {
	var updating bool
	var check *widget.Check
	check = widget.NewCheck("开机自动启动", func(checked bool) {
		if updating {
			return
		}
		if checked {
			exePath, err := util.GetExePath()
			if err != nil {
				dialog.ShowError(fmt.Errorf("获取程序路径失败: %w", err), s.win)
				updating = true
				check.SetChecked(false)
				updating = false
				return
			}
			if err := s.autostartMgr.Enable(exePath); err != nil {
				dialog.ShowError(fmt.Errorf("启用开机自启失败: %w", err), s.win)
				updating = true
				check.SetChecked(false)
				updating = false
				return
			}
		} else {
			if err := s.autostartMgr.Disable(); err != nil {
				dialog.ShowError(fmt.Errorf("禁用开机自启失败: %w", err), s.win)
				updating = true
				check.SetChecked(true)
				updating = false
				return
			}
		}
		s.cfg.AutoStart = checked
		s.saveConfig()
	})
	check.SetChecked(s.cfg.AutoStart)
	return check
}

// buildHotkeySection 构建热键设置区域（按键录制模式）
func (s *SettingsWindow) buildHotkeySection() *fyne.Container {
	displayText := s.cfg.Hotkey
	if displayText == "" {
		displayText = "未设置"
	}
	hotkeyLabel := widget.NewLabel(displayText)

	var recordBtn *widget.Button
	recordBtn = widget.NewButton("录制快捷键", func() {
		if s.cancelRecording != nil {
			s.cancelRecording()
			return
		}
		s.startHotkeyRecording(hotkeyLabel, recordBtn)
	})

	clearBtn := widget.NewButton("清除", func() {
		if s.cancelRecording != nil {
			s.cancelRecording()
		}
		if s.tray.hotkeyMgr != nil {
			s.tray.hotkeyMgr.Unregister()
			s.tray.hotkeyMgr = nil
		}
		s.cfg.Hotkey = ""
		hotkeyLabel.SetText("未设置")
		s.saveConfig()
	})

	return container.NewBorder(nil, nil, nil,
		container.NewHBox(recordBtn, clearBtn),
		hotkeyLabel,
	)
}

// startHotkeyRecording 开始录制快捷键（Windows API 轮询方式）
func (s *SettingsWindow) startHotkeyRecording(label *widget.Label, btn *widget.Button) {
	logger.Info("Settings", "开始录制快捷键", "currentHotkey", s.cfg.Hotkey, "hotkeyMgr", s.tray.hotkeyMgr != nil)

	// 先抑制回调，防止 Unregister 竞态触发 QuickSwitch
	s.tray.suppress.Store(true)

	if s.tray.hotkeyMgr != nil {
		s.tray.hotkeyMgr.Unregister()
		s.tray.hotkeyMgr = nil
	}

	btn.SetText("按下快捷键组合... (Esc取消)")
	btn.Disable()
	label.SetText("_")

	quit := make(chan struct{})
	resultCh := make(chan string, 1)

	s.cancelRecording = func() {
		select {
		case <-quit:
		default:
			close(quit)
		}
	}

	// 后台轮询按键
	go func() {
		hotkeyStr := hotkey.RecordHotkey(quit)
		select {
		case <-quit:
		default:
			resultCh <- hotkeyStr
		}
	}()

	// 定时检查结果
	var checkTimer *time.Timer
	checkTimer = time.AfterFunc(80*time.Millisecond, func() {
		select {
		case hotkeyStr := <-resultCh:
			if hotkeyStr == "" {
				// 取消
				btn.SetText("录制快捷键")
				btn.Enable()
				s.cancelRecording = nil
				s.tray.suppress.Store(false)
				s.restoreHotkey()
				displayText := s.cfg.Hotkey
				if displayText == "" {
					displayText = "未设置"
				}
				label.SetText(displayText)
				return
			}

			if err := s.tray.UpdateHotkey(hotkeyStr); err != nil {
				logger.Warn("Settings", "注册热键失败", "hotkey", hotkeyStr, "error", err)
				label.SetText("注册失败: " + err.Error())
				s.restoreHotkey()
				displayText := s.cfg.Hotkey
				if displayText == "" {
					displayText = "未设置"
				}
				label.SetText(displayText)
			} else {
				logger.Info("Settings", "热键已录制", "hotkey", hotkeyStr)
				label.SetText(hotkeyStr)
				s.saveConfig()
			}
			btn.SetText("录制快捷键")
			btn.Enable()
			s.cancelRecording = nil
			s.tray.suppress.Store(false)
		default:
			select {
			case <-quit:
				btn.SetText("录制快捷键")
				btn.Enable()
				s.cancelRecording = nil
				s.tray.suppress.Store(false)
				s.restoreHotkey()
				displayText := s.cfg.Hotkey
				if displayText == "" {
					displayText = "未设置"
				}
				label.SetText(displayText)
			default:
				checkTimer.Reset(80 * time.Millisecond)
			}
		}
	})
}

// restoreHotkey 恢复配置中保存的热键
func (s *SettingsWindow) restoreHotkey() {
	if s.cfg.Hotkey != "" && s.tray.callback != nil && s.tray.hotkeyMgr == nil {
		if mgr, err := hotkey.Register(s.cfg.Hotkey, s.tray.callback); err == nil {
			s.tray.hotkeyMgr = mgr
		} else {
			logger.Warn("Settings", "恢复热键失败", "hotkey", s.cfg.Hotkey, "error", err)
		}
	}
}

// buildDeviceListSection 构建设备列表区域
func (s *SettingsWindow) buildDeviceListSection() *fyne.Container {
	var items []*widget.AccordionItem
	for _, dev := range s.devices {
		dev := dev
		status := ""
		if dev.IsDefault {
			status = " (当前)"
		}
		title := dev.Name + status
		content := widget.NewButton("切换到此设备", func() {
			vol := s.tray.getVolumePreset(dev.ID)
			if err := s.tray.switchWithVolume(dev.ID, vol); err != nil {
				dialog.ShowError(err, s.win)
				return
			}
			s.RefreshUI()
			s.tray.RefreshMenu()
		})
		items = append(items, &widget.AccordionItem{
			Title:  title,
			Detail: content,
		})
	}

	accordion := widget.NewAccordion(items...)
	if len(items) > 0 {
		accordion.Open(0)
	}

	return container.NewVBox(
		widget.NewLabelWithStyle("音频输出设备", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		accordion,
	)
}

// buildActionButtons 构建操作按钮
func (s *SettingsWindow) buildActionButtons() *fyne.Container {
	switchBtn := widget.NewButton("快速切换", func() {
		s.tray.QuickSwitch()
		s.RefreshUI()
	})

	refreshBtn := widget.NewButton("刷新", func() {
		s.RefreshUI()
		s.tray.RefreshMenu()
	})

	return container.NewGridWithColumns(2, switchBtn, refreshBtn)
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
		dialog.ShowError(err, s.win)
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
		s.saveConfig()
	})
}

func formatPercent(v int) string {
	return fmt.Sprintf("%d%%", v)
}
