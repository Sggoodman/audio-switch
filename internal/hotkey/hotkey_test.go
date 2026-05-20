package hotkey

import (
	"testing"
)

func TestParseHotkeyString(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		mods    int
	}{
		{"Ctrl+Alt+S", false, 2},
		{"Ctrl+Shift+F12", false, 2},
		{"Alt+Space", false, 1},
		{"", true, 0},
		{"S", true, 0},
		{"Ctrl+", true, 0},
		{"Unknown+A", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mods, key, err := ParseHotkeyString(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("期望返回错误，但没有")
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望错误: %v", err)
			}
			if len(mods) != tt.mods {
				t.Errorf("修饰键数量: 得到 %d, 期望 %d", len(mods), tt.mods)
			}
			if key == 0 {
				t.Error("主键不应为 0")
			}
		})
	}
}

func TestSupportedKeys(t *testing.T) {
	keys := SupportedKeys()
	if len(keys) == 0 {
		t.Error("支持的主键列表不应为空")
	}
	found := false
	for _, k := range keys {
		if k == "S" {
			found = true
			break
		}
	}
	if !found {
		t.Error("支持的主键列表应包含 S")
	}
}

func TestSupportedModifiers(t *testing.T) {
	mods := SupportedModifiers()
	if len(mods) != 3 {
		t.Errorf("应有 3 个修饰键，实际 %d", len(mods))
	}
}
