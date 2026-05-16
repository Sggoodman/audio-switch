//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	desktopEntry = `[Desktop Entry]
Type=Application
Name=Audio Switch
Exec=%s
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Comment=Audio output device switcher`
)

type linuxManager struct{}

func newManager() Manager {
	return &linuxManager{}
}

func (m *linuxManager) desktopPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "autostart", "audio-switch.desktop"), nil
}

func (m *linuxManager) Enable(exePath string) error {
	path, err := m.desktopPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf(desktopEntry, exePath)
	return os.WriteFile(path, []byte(content), 0644)
}

func (m *linuxManager) Disable() error {
	path, err := m.desktopPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (m *linuxManager) IsEnabled() (bool, error) {
	path, err := m.desktopPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
