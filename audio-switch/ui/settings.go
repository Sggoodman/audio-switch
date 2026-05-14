package ui

import (
	"audio-switch/internal/audio"
	"audio-switch/internal/config"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// SettingsWindow 偏好设置窗口
type SettingsWindow struct {
	fyneApp  fyne.App
	win      fyne.Window
	audioAPI audio.Audio
	cfg      *config.Config
	tray     *TrayApp
	devices  []audio.Device
}

// NewSettingsWindow 创建设置窗口
func NewSettingsWindow(app fyne.App, a audio.Audio, cfg *config.Config, tray *TrayApp) *SettingsWindow {
	s := &SettingsWindow{
		fyneApp:  app,
		audioAPI: a,
		cfg:      cfg,
		tray:     tray,
	}

	s.win = app.NewWindow("音频输出设置")
	s.win.SetContent(s.buildUI())
	s.win.Resize(fyne.NewSize(500, 450))

	// 窗口关闭时隐藏而不是退出
	s.win.SetCloseIntercept(func() {
		s.win.Hide()
	})

	return s
}

// Show 显示设置窗口
func (s *SettingsWindow) Show() {
	s.RefreshDevices()
	s.win.Show()
	s.win.RequestFocus()
}

// RefreshDevices 刷新设备列表
func (s *SettingsWindow) RefreshDevices() {
	devices, err := s.audioAPI.GetDevices()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	s.devices = devices
}

// buildUI 构建设置界面
func (s *SettingsWindow) buildUI() *fyne.Container {
	// ---- 快捷切换设置 ----
	quickSwitchCard := s.buildQuickSwitchSection()

	// ---- 设备列表 ----
	deviceListCard := s.buildDeviceListSection()

	// ---- 操作按钮 ----
	buttons := s.buildActionButtons()

	return container.NewVBox(
		quickSwitchCard,
		widget.NewSeparator(),
		deviceListCard,
		widget.NewSeparator(),
		buttons,
	)
}

// buildQuickSwitchSection 构建快捷切换设置区域
func (s *SettingsWindow) buildQuickSwitchSection() *fyne.Container {
	// 获取设备名称列表
	devices, _ := s.audioAPI.GetDevices()
	s.devices = devices

	deviceNames := []string{""}
	deviceMap := map[string]string{} // name -> id
	for _, d := range devices {
		deviceNames = append(deviceNames, d.Name)
		deviceMap[d.Name] = d.ID
	}

	// 设备 A 选择
	dev1Name := ""
	if s.cfg.Device1 != nil {
		dev1Name = s.cfg.Device1.Name
	}
	dev1Select := widget.NewSelect(deviceNames, func(name string) {
		if name == "" {
			s.cfg.Device1 = nil
		} else {
			s.cfg.Device1 = &config.DeviceConfig{
				ID:   deviceMap[name],
				Name: name,
			}
			if s.cfg.Device1.Volume == 0 {
				s.cfg.Device1.Volume = 80
			}
		}
		s.saveConfig()
	})
	dev1Select.PlaceHolder = "选择设备 A"
	dev1Select.SetSelected(dev1Name)

	// 设备 B 选择
	dev2Name := ""
	if s.cfg.Device2 != nil {
		dev2Name = s.cfg.Device2.Name
	}
	dev2Select := widget.NewSelect(deviceNames, func(name string) {
		if name == "" {
			s.cfg.Device2 = nil
		} else {
			s.cfg.Device2 = &config.DeviceConfig{
				ID:   deviceMap[name],
				Name: name,
			}
			if s.cfg.Device2.Volume == 0 {
				s.cfg.Device2.Volume = 50
			}
		}
		s.saveConfig()
	})
	dev2Select.PlaceHolder = "选择设备 B"
	dev2Select.SetSelected(dev2Name)

	// 音量 A 滑块
	vol1Label := widget.NewLabel("80%")
	vol1 := 80
	if s.cfg.Device1 != nil && s.cfg.Device1.Volume > 0 {
		vol1 = s.cfg.Device1.Volume
	}
	vol1Slider := widget.NewSlider(0, 100)
	vol1Slider.SetValue(float64(vol1))
	vol1Slider.OnChanged = func(v float64) {
		vol1Label.SetText(formatPercent(int(v)))
		if s.cfg.Device1 != nil {
			s.cfg.Device1.Volume = int(v)
			s.saveConfig()
		}
	}

	// 音量 B 滑块
	vol2Label := widget.NewLabel("50%")
	vol2 := 50
	if s.cfg.Device2 != nil && s.cfg.Device2.Volume > 0 {
		vol2 = s.cfg.Device2.Volume
	}
	vol2Slider := widget.NewSlider(0, 100)
	vol2Slider.SetValue(float64(vol2))
	vol2Slider.OnChanged = func(v float64) {
		vol2Label.SetText(formatPercent(int(v)))
		if s.cfg.Device2 != nil {
			s.cfg.Device2.Volume = int(v)
			s.saveConfig()
		}
	}

	// 通知开关
	notifyCheck := widget.NewCheck("切换时弹出通知", func(checked bool) {
		s.cfg.NotificationEnabled = checked
		s.saveConfig()
	})
	notifyCheck.SetChecked(s.cfg.NotificationEnabled)

	// 开机自启
	autoStartCheck := widget.NewCheck("开机自动启动", func(checked bool) {
		s.cfg.AutoStart = checked
		s.saveConfig()
	})
	autoStartCheck.SetChecked(s.cfg.AutoStart)

	return container.NewVBox(
		widget.NewLabelWithStyle("快捷切换设置", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("设备 A:"), dev1Select),
			container.NewVBox(widget.NewLabel("设备 B:"), dev2Select),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("A 音量:"), container.NewBorder(nil, nil, nil, vol1Label, vol1Slider)),
			container.NewVBox(widget.NewLabel("B 音量:"), container.NewBorder(nil, nil, nil, vol2Label, vol2Slider)),
		),
		container.NewGridWithColumns(2, notifyCheck, autoStartCheck),
	)
}

// buildDeviceListSection 构建设备列表区域
func (s *SettingsWindow) buildDeviceListSection() *fyne.Container {
	devices, _ := s.audioAPI.GetDevices()

	var items []*widget.AccordionItem
	for _, dev := range devices {
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
			s.RefreshDevices()
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
		s.RefreshDevices()
	})

	refreshBtn := widget.NewButton("刷新", func() {
		s.RefreshDevices()
		s.tray.RefreshMenu()
		// 重建 UI 比较复杂，直接关闭再打开
		s.win.Hide()
		s.tray.ShowSettings()
	})

	return container.NewGridWithColumns(2, switchBtn, refreshBtn)
}

// saveConfig 保存配置
func (s *SettingsWindow) saveConfig() {
	if err := config.Save(s.cfg); err != nil {
		dialog.ShowError(err, s.win)
	}
}

func formatPercent(v int) string {
	return fmt.Sprintf("%d%%", v)
}
