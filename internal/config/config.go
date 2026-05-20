package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DeviceConfig 偏好设备的配置
type DeviceConfig struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Volume int    `json:"volume,omitempty"`
}

// Config 应用配置
type Config struct {
	Device1              *DeviceConfig `json:"device1,omitempty"`
	Device2              *DeviceConfig `json:"device2,omitempty"`
	Hotkey               string        `json:"hotkey,omitempty"`
	NotificationEnabled  bool          `json:"notificationEnabled"`
	AutoStart            bool          `json:"autoStart"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Hotkey:              "Ctrl+Alt+S",
		NotificationEnabled: true,
		AutoStart:           false,
	}
}

var (
	configPath string
	configOnce sync.Once
)

// GetConfigPath 返回配置文件路径
func GetConfigPath() string {
	configOnce.Do(func() {
		if configPath != "" {
			return
		}
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		configPath = filepath.Join(home, ".config", "audio-switch", "config.json")
	})
	return configPath
}

// SetConfigPath 设置配置文件路径（用于测试）
func SetConfigPath(p string) {
	configPath = p
	configOnce = sync.Once{}
}

// Load 从文件加载配置
func Load() (*Config, error) {
	path := GetConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			// 尝试创建目录和文件
			_ = Save(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// Save 保存配置到文件
func Save(cfg *Config) error {
	path := GetConfigPath()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
