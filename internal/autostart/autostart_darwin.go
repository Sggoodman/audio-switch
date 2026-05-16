//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	launchAgentID = "com.audioswitch.app"
	plistContent  = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>`
)

type darwinManager struct{}

func newManager() Manager {
	return &darwinManager{}
}

func (m *darwinManager) plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentID+".plist"), nil
}

func (m *darwinManager) Enable(exePath string) error {
	path, err := m.plistPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf(plistContent, launchAgentID, exePath)
	return os.WriteFile(path, []byte(content), 0644)
}

func (m *darwinManager) Disable() error {
	path, err := m.plistPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (m *darwinManager) IsEnabled() (bool, error) {
	path, err := m.plistPath()
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
