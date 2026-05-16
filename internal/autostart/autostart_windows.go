//go:build windows

package autostart

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	regKey  = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
	regName = "AudioSwitch"
)

type windowsManager struct{}

func newManager() Manager {
	return &windowsManager{}
}

func (m *windowsManager) Enable(exePath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(regName, exePath)
}

func (m *windowsManager) Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(regName)
}

func (m *windowsManager) IsEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer k.Close()

	_, _, err = k.GetStringValue(regName)
	if err != nil {
		return false, nil // 值不存在视为未启用
	}
	return true, nil
}
