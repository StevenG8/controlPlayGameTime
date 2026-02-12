package autostart

import (
	"fmt"
	"os/exec"
	"runtime"
)

const taskName = "GameControlAutoStart"

func InstallTask(exePath, configPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("仅支持在 Windows 上安装自启动任务")
	}

	command := fmt.Sprintf("\"%s\" start --background \"%s\"", exePath, configPath)
	cmd := exec.Command("schtasks", "/Create", "/F", "/SC", "ONLOGON", "/TN", taskName, "/TR", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建自启动任务失败: %w, 输出: %s", err, string(output))
	}
	return nil
}

func RemoveTask() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("仅支持在 Windows 上移除自启动任务")
	}

	cmd := exec.Command("schtasks", "/Delete", "/F", "/TN", taskName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除自启动任务失败: %w, 输出: %s", err, string(output))
	}
	return nil
}
