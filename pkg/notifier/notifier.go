package notifier

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Notifier interface {
	NotifyFirstWarning(remainingMinutes int) error
	NotifyFinalWarning(remainingMinutes int) error
	NotifyLimitExceeded() error
}

type WindowsNotifier struct{}

func NewNotifier() Notifier {
	return &WindowsNotifier{}
}

func (n *WindowsNotifier) NotifyFirstWarning(remainingMinutes int) error {
	msg := fmt.Sprintf("游戏剩余时间不足，当前还剩 %d 分钟。", remainingMinutes)
	return showPopup("游戏时间提醒", msg)
}

func (n *WindowsNotifier) NotifyFinalWarning(remainingMinutes int) error {
	msg := fmt.Sprintf("最后提醒：游戏剩余时间仅 %d 分钟。", remainingMinutes)
	return showPopup("游戏时间最后提醒", msg)
}

func (n *WindowsNotifier) NotifyLimitExceeded() error {
	return showPopup("游戏时间已用尽", "今日游戏时间已达上限，系统将终止游戏进程。")
}

func showPopup(title, message string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("桌面弹窗仅支持 Windows")
	}

	title = escapeSingleQuotes(title)
	message = escapeSingleQuotes(message)
	script := fmt.Sprintf("Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show('%s','%s') | Out-Null", message, title)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("弹窗通知失败: %w, 输出: %s", err, string(output))
	}
	return nil
}

func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
