//go:build darwin

package audio

import (
	"testing"
	"time"
)

func TestDarwinSwitchRoundTrip(t *testing.T) {
	a := New()
	orig, err := a.GetDefaultDevice()
	if err != nil {
		t.Fatalf("GetDefaultDevice: %v", err)
	}

	devices, err := a.GetDevices()
	if err != nil {
		t.Fatalf("GetDevices: %v", err)
	}
	var target string
	for _, d := range devices {
		if !d.IsDefault {
			target = d.ID
			break
		}
	}
	if target == "" {
		t.Skip("无其他输出设备可切换")
	}

	if err := a.SetDefaultDevice(target); err != nil {
		t.Fatalf("SetDefaultDevice(%s): %v", target, err)
	}
	time.Sleep(300 * time.Millisecond)
	now, err := a.GetDefaultDevice()
	if err != nil {
		t.Fatalf("切换后 GetDefaultDevice: %v", err)
	}
	if now.Name != target {
		t.Errorf("切换未生效: 期望 %s, 实际 %s", target, now.Name)
	}

	// 恢复原设备
	if err := a.SetDefaultDevice(orig.ID); err != nil {
		t.Fatalf("恢复 SetDefaultDevice(%s): %v", orig.ID, err)
	}
	time.Sleep(300 * time.Millisecond)
	back, _ := a.GetDefaultDevice()
	if back.Name != orig.Name {
		t.Errorf("恢复失败: 期望 %s, 实际 %s", orig.Name, back.Name)
	}
}

func TestDarwinGetDevices(t *testing.T) {
	a := New()
	devices, err := a.GetDevices()
	if err != nil {
		t.Fatalf("GetDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("未枚举到任何输出设备")
	}
	found := false
	for _, d := range devices {
		t.Logf("设备: %s (默认: %v)", d.Name, d.IsDefault)
		if d.IsDefault {
			found = true
		}
	}
	if !found {
		t.Error("未标记默认设备")
	}
}

func TestDarwinGetDefaultDevice(t *testing.T) {
	a := New()
	dev, err := a.GetDefaultDevice()
	if err != nil {
		t.Fatalf("GetDefaultDevice: %v", err)
	}
	if dev.Name == "" {
		t.Error("默认设备名为空")
	}
	t.Logf("默认设备: %s", dev.Name)
}
