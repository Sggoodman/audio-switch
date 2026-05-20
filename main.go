package main

import (
	"os"
	"path/filepath"

	"audio-switch/internal/audio"
	"audio-switch/internal/autostart"
	"audio-switch/internal/config"
	"audio-switch/internal/logger"
	"audio-switch/internal/notify"
	"audio-switch/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

func init() {
	logPath := filepath.Join(os.TempDir(), "audio-switch", "app.log")
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err == nil {
		logger.Init(logPath, false)
		logger.Info("Main", "=== Audio Switch 启动 ===")
	}
}

func main() {
	logger.Info("Main", "开始加载配置...")
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.Warn("Main", "加载配置失败，使用默认配置", "error", err)
		cfg = config.DefaultConfig()
	} else {
		d1Vol, d2Vol := 0, 0
		if cfg.Device1 != nil {
			d1Vol = cfg.Device1.Volume
		}
		if cfg.Device2 != nil {
			d2Vol = cfg.Device2.Volume
		}
		logger.Info("Main", "配置加载成功", "device1_vol", d1Vol, "device2_vol", d2Vol)
	}

	// 初始化平台音频接口
	audioAPI := audio.New()

	// 初始化通知
	notifier := notify.New()

	// 创建 Fyne 应用
	a := app.NewWithID("com.audioswitch.app")
	a.SetIcon(loadIcon())

	// 主窗口（隐藏，仅用于托盘应用的生命周期管理）
	w := a.NewWindow("Audio Switch")
	w.SetContent(widget.NewLabel("")) // 占位内容
	w.Resize(fyne.NewSize(1, 1))
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// 创建托盘应用
	tray := ui.NewTrayApp(a, audioAPI, notifier, cfg)
	tray.Setup()

	// 同步开机自启状态：确保配置与注册表/文件一致
	{
		autostartMgr := autostart.New()
		enabled, err := autostartMgr.IsEnabled()
		if err == nil {
			if cfg.AutoStart && !enabled {
				exePath, exeErr := getExePath()
				if exeErr == nil {
					if regErr := autostartMgr.Enable(exePath); regErr != nil {
						logger.Warn("Autostart", "同步开机自启失败", "error", regErr)
					} else {
						logger.Info("Autostart", "已补注册开机自启")
					}
				}
			} else if !cfg.AutoStart && enabled {
				if regErr := autostartMgr.Disable(); regErr != nil {
					logger.Warn("Autostart", "清理开机自启残留失败", "error", regErr)
				} else {
					logger.Info("Autostart", "已清理开机自启残留")
				}
			}
		}
	}

	// 注册全局热键
	tray.InitHotkey()
	defer tray.Cleanup()

	a.Run()
}

// loadIcon 从可执行文件同目录或 assets 目录加载图标
func loadIcon() fyne.Resource {
	// 候选图标路径（优先级从高到低）
	candidates := []string{}
	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exePath), "assets", "Icon.png"),
			filepath.Join(filepath.Dir(exePath), "Icon.png"),
		)
	}
	candidates = append(candidates, "assets/Icon.png", "Icon.png")

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil && len(data) > 0 {
			logger.Info("Main", "加载图标", "path", p, "size", len(data))
			return fyne.NewStaticResource("Icon.png", data)
		}
	}

	logger.Warn("Main", "未找到图标文件，使用 Fyne 默认图标")
	return nil
}

// getExePath 返回当前可执行文件的绝对路径
func getExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}
