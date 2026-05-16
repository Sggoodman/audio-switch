//go:build linux

package audio

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// LinuxAudio 通过 pactl (兼容 PulseAudio 和 PipeWire) 控制音频设备
type LinuxAudio struct{}

// New 创建音频操作实例（Linux 平台）
func New() Audio {
	return &LinuxAudio{}
}

// pulseSink 表示 pactl list sinks 的 JSON 输出结构
type pulseSink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
}

// GetDevices 枚举所有音频输出设备
func (a *LinuxAudio) GetDevices() ([]Device, error) {
	out, err := exec.Command("pactl", "-f", "json", "list", "sinks").Output()
	if err != nil {
		return nil, fmt.Errorf("pactl list sinks failed: %w", err)
	}

	var sinks []pulseSink
	if err := json.Unmarshal(out, &sinks); err != nil {
		// JSON 解析失败，尝试纯文本解析
		return a.parseTextOutput(string(out))
	}

	defaultName := a.getDefaultSinkName()

	var devices []Device
	for _, sink := range sinks {
		devices = append(devices, Device{
			ID:        sink.Name,
			Name:      sink.Description,
			IsDefault: sink.Name == defaultName,
		})
	}
	return devices, nil
}

// GetDefaultDevice 获取当前默认音频输出设备
func (a *LinuxAudio) GetDefaultDevice() (*Device, error) {
	name := a.getDefaultSinkName()
	if name == "" {
		return nil, fmt.Errorf("failed to get default sink")
	}

	// 获取设备列表以找到描述
	devices, err := a.GetDevices()
	if err != nil {
		return &Device{ID: name, Name: name, IsDefault: true}, nil
	}

	for _, d := range devices {
		if d.ID == name {
			d.IsDefault = true
			return &d, nil
		}
	}

	return &Device{ID: name, Name: name, IsDefault: true}, nil
}

// SetDefaultDevice 切换到指定音频输出设备
func (a *LinuxAudio) SetDefaultDevice(id string) error {
	out, err := exec.Command("pactl", "set-default-sink", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pactl set-default-sink failed: %w\n%s", err, string(out))
	}
	return nil
}

// SetDeviceVolume 设置音量 (0-100)
func (a *LinuxAudio) SetDeviceVolume(id string, volume int) error {
	volStr := fmt.Sprintf("%d%%", volume)
	out, err := exec.Command("pactl", "set-sink-volume", id, volStr).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pactl set-sink-volume failed: %w\n%s", err, string(out))
	}
	return nil
}

func (a *LinuxAudio) getDefaultSinkName() string {
	out, err := exec.Command("pactl", "get-default-sink").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// parseTextOutput 解析 pactl 的纯文本输出（旧版本不支持 -f json）
func (a *LinuxAudio) parseTextOutput(text string) ([]Device, error) {
	defaultName := a.getDefaultSinkName()
	var devices []Device

	var currentName, currentDesc string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name:") {
			if currentName != "" {
				devices = append(devices, Device{
					ID:        currentName,
					Name:      currentDesc,
					IsDefault: currentName == defaultName,
				})
			}
			currentName = strings.TrimPrefix(line, "Name:")
			currentName = strings.TrimSpace(currentName)
			currentDesc = currentName
		} else if strings.HasPrefix(line, "Description:") {
			currentDesc = strings.TrimPrefix(line, "Description:")
			currentDesc = strings.TrimSpace(currentDesc)
		}
	}

	if currentName != "" {
		devices = append(devices, Device{
			ID:        currentName,
			Name:      currentDesc,
			IsDefault: currentName == defaultName,
		})
	}

	return devices, nil
}
