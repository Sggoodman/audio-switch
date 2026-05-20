package util

import (
	"testing"
)

func TestGetExePath(t *testing.T) {
	path, err := GetExePath()
	if err != nil {
		t.Fatalf("GetExePath 失败: %v", err)
	}
	if path == "" {
		t.Error("路径不应为空")
	}
	t.Logf("可执行文件路径: %s", path)
}
