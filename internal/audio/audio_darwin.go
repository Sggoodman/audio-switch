//go:build darwin

package audio

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// MacOSAudio 通过 SwitchAudioSource CLI 控制音频设备
type MacOSAudio struct{}

// New 创建音频操作实例（macOS 平台）
func New() Audio {
	return &MacOSAudio{}
}

// GetDevices 枚举所有音频输出设备
func (a *MacOSAudio) GetDevices() ([]Device, error) {
	out, err := exec.Command("SwitchAudioSource", "-a", "-t", "output").Output()
	if err != nil {
		return nil, fmt.Errorf("SwitchAudioSource -a failed (install with: brew install switchaudio-osx): %w", err)
	}

	currentName := a.getCurrentDeviceName()

	var devices []Device
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 输出格式: "device_name (output)" 或 "device_name"
		name := regexp.MustCompile(`\s*\(output\)\s*$`).ReplaceAllString(line, "")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		devices = append(devices, Device{
			ID:        name, // macOS 用设备名作为 ID
			Name:      name,
			IsDefault: name == currentName,
		})
	}
	return devices, nil
}

// GetDefaultDevice 获取当前默认音频输出设备
func (a *MacOSAudio) GetDefaultDevice() (*Device, error) {
	name := a.getCurrentDeviceName()
	if name == "" {
		return nil, fmt.Errorf("failed to get default audio device")
	}
	return &Device{
		ID:        name,
		Name:      name,
		IsDefault: true,
	}, nil
}

// SetDefaultDevice 切换到指定音频输出设备
func (a *MacOSAudio) SetDefaultDevice(id string) error {
	out, err := exec.Command("SwitchAudioSource", "-t", "output", "-s", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("SwitchAudioSource -s failed: %w\n%s", err, string(out))
	}
	return nil
}

// SetDeviceVolume 设置音量 (0-100)
func (a *MacOSAudio) SetDeviceVolume(id string, volume int) error {
	script := fmt.Sprintf("set volume output volume %d", volume)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript set volume failed: %w\n%s", err, string(out))
	}
	return nil
}

func (a *MacOSAudio) getCurrentDeviceName() string {
	out, err := exec.Command("SwitchAudioSource", "-c", "-t", "output").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
