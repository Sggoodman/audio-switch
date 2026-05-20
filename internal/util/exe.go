package util

import (
	"os"
	"path/filepath"
)

// GetExePath 返回当前可执行文件的绝对路径。
func GetExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}
