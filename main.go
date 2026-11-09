package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"audio-switch/internal/audio"
	"audio-switch/internal/config"
	"audio-switch/internal/logger"
	"audio-switch/ui"

	"github.com/egoist/mygo"
)

//go:embed assets/tray.png
var assetsFS embed.FS

func init() {
	logPath := filepath.Join(os.TempDir(), "audio-switch", "app.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)
	logger.Init(logPath, false)
	logger.Info("Main", "=== Audio Switch 启动 ===")
}

func main() {
	// 单实例：重复启动时唤起已运行实例后退出
	if !mygo.App.RequestSingleInstanceLock() {
		logger.Info("Main", "已有实例运行，退出")
		return
	}

	// 纯托盘应用：无 Dock 图标、无主窗口（替代原 hideDockIcon hack）
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)

	// 标记退出流程：设置窗口的"关闭即隐藏"须放行退出（OnBeforeQuit 与 OnClose 同在主线程）
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { ui.MarkQuitting() })

	// 最后一个窗口关闭时保持运行（托盘常驻）
	mygo.App.OnWindowAllClosed(func() {})

	logger.Info("Main", "开始加载配置...")
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

	audioAPI := audio.New()

	trayIcon, err := assetsFS.ReadFile("assets/tray.png")
	if err != nil {
		logger.Warn("Main", "加载托盘图标失败", "error", err)
	}

	tray := ui.NewTrayApp(audioAPI, cfg, trayIcon)

	defer logger.Sync()
	defer tray.Cleanup()

	// 同步开机自启状态：确保配置与系统注册一致
	syncAutoStart(cfg)

	mygo.App.WhenReady(func() {
		tray.Setup()
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// syncAutoStart 同步配置与系统开机自启注册状态
func syncAutoStart(cfg *config.Config) {
	enabled := mygo.App.OpenAtLogin()
	if cfg.AutoStart && !enabled {
		if err := mygo.App.SetOpenAtLogin(true); err != nil {
			logger.Warn("Autostart", "同步开机自启失败", "error", err)
		} else {
			logger.Info("Autostart", "已补注册开机自启")
		}
	} else if !cfg.AutoStart && enabled {
		if err := mygo.App.SetOpenAtLogin(false); err != nil {
			logger.Warn("Autostart", "清理开机自启残留失败", "error", err)
		} else {
			logger.Info("Autostart", "已清理开机自启残留")
		}
	}
}
