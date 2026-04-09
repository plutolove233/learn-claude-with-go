package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// AbsToTilde 把绝对路径转换成 ~ 开头的路径
// 例如：/home/xxx/Desktop → ~/Desktop
// 如果不在家目录下，则返回原路径
func AbsToTilde(absPath string) (string, error) {
	// 1. 获取用户家目录
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// 2. 获取绝对路径的规范化版本（处理 ./ ../ 等）
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", err
	}

	// 3. 判断路径是否以家目录开头
	if strings.HasPrefix(absPath, home) {
		// 替换为 ~
		if absPath == home {
			return "~", nil
		}
		// 拼接 ~/xxx
		return "~" + absPath[len(home):], nil
	}

	// 不在家目录，直接返回
	return absPath, nil
}
