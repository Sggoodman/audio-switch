package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Hotkey != "Ctrl+Alt+S" {
		t.Errorf("默认热键应为 Ctrl+Alt+S，实际为 %s", cfg.Hotkey)
	}
	if cfg.NotificationEnabled != true {
		t.Error("默认应启用通知")
	}
	if cfg.AutoStart != false {
		t.Error("默认不应启用自启")
	}
	if cfg.Device1 != nil || cfg.Device2 != nil {
		t.Error("默认设备配置应为 nil")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	SetConfigPath(filepath.Join(dir, "config.json"))

	cfg := &Config{
		Device1: &DeviceConfig{
			ID:     "dev1",
			Name:   "Speaker",
			Volume: 60,
		},
		Device2: &DeviceConfig{
			ID:     "dev2",
			Name:   "Headphone",
			Volume: 80,
		},
		Hotkey:              "Ctrl+Alt+H",
		NotificationEnabled: true,
		AutoStart:           false,
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	if loaded.Device1.ID != "dev1" || loaded.Device1.Volume != 60 {
		t.Errorf("Device1 加载不正确: %+v", loaded.Device1)
	}
	if loaded.Device2.ID != "dev2" || loaded.Device2.Volume != 80 {
		t.Errorf("Device2 加载不正确: %+v", loaded.Device2)
	}
	if loaded.Hotkey != "Ctrl+Alt+H" {
		t.Errorf("Hotkey 不正确: %s", loaded.Hotkey)
	}
	if !loaded.NotificationEnabled {
		t.Error("NotificationEnabled 应为 true")
	}
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	SetConfigPath(filepath.Join(dir, "no-such-dir", "config.json"))
	// 清除之前测试可能创建的缓存路径
	configOnce = sync.Once{}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("文件不存在时应返回默认配置，不应报错: %v", err)
	}
	if cfg.Hotkey != "Ctrl+Alt+S" {
		t.Errorf("应返回默认配置，实际 Hotkey=%s", cfg.Hotkey)
	}
}

func TestGetConfigPath(t *testing.T) {
	// 重置为默认路径
	configPath = ""
	configOnce = sync.Once{}

	path := GetConfigPath()
	if path == "" {
		t.Error("配置路径不应为空")
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Errorf("配置目录不存在: %v", err)
	}
}
