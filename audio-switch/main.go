package main

import (
	"log"
	"os"
	"path/filepath"

	"audio-switch/internal/audio"
	"audio-switch/internal/config"
	"audio-switch/internal/hotkey"
	"audio-switch/internal/notify"
	"audio-switch/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Printf("加载配置失败: %v，使用默认配置", err)
		cfg = config.DefaultConfig()
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

	// 注册全局热键
	if cfg.Hotkey != "" {
		hk, err := hotkey.Register(cfg.Hotkey, func() {
			tray.QuickSwitch()
		})
		if err != nil {
			log.Printf("注册热键 %s 失败: %v", cfg.Hotkey, err)
		} else {
			defer hk.Unregister()
			log.Printf("热键 %s 已注册", cfg.Hotkey)
		}
	}

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
			log.Printf("加载图标: %s (%d bytes)", p, len(data))
			return fyne.NewStaticResource("Icon.png", data)
		}
	}

	log.Println("未找到图标文件，使用 Fyne 默认图标")
	return nil
}
