package process

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID       int       `json:"pid"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"startTime"`
}

// Scanner 进程扫描器
type Scanner struct {
	lastProcesses map[int]ProcessInfo // 上次扫描的进程
}

// NewScanner 创建新的进程扫描器
func NewScanner() *Scanner {
	return &Scanner{
		lastProcesses: make(map[int]ProcessInfo),
	}
}

// ScanProcesses 扫描当前运行的进程
func (s *Scanner) ScanProcesses() ([]ProcessInfo, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("当前只支持 Windows 平台")
	}

	// 使用 tasklist 命令获取进程列表
	cmd := exec.Command("tasklist", "/fo", "csv", "/nh")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 tasklist 命令失败: %w", err)
	}

	// 解析输出
	lines := strings.Split(string(output), "\n")
	processes := make([]ProcessInfo, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 CSV 格式的行
		fields := parseCSVLine(line)
		if len(fields) < 2 {
			continue
		}

		// fields[0] 是进程名称，fields[1] 是 PID
		name := strings.Trim(fields[0], "\"")
		pidStr := strings.Trim(fields[1], "\"")

		var pid int
		if _, err := fmt.Sscanf(pidStr, "%d", &pid); err != nil {
			continue
		}

		processes = append(processes, ProcessInfo{
			PID:       pid,
			Name:      name,
			StartTime: time.Now(), // 这里简化处理，实际可以从进程创建时间获取
		})
	}

	return processes, nil
}

// parseCSVLine 解析 CSV 行（处理带引号的字段）
func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				fields = append(fields, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

// FindGameProcesses 查找游戏进程
func (s *Scanner) FindGameProcesses(gameNames []string) ([]ProcessInfo, error) {
	allProcesses, err := s.ScanProcesses()
	if err != nil {
		return nil, err
	}

	gameProcesses := make([]ProcessInfo, 0)
	for _, proc := range allProcesses {
		for _, gameName := range gameNames {
			// 精确匹配（不区分大小写）
			if strings.EqualFold(proc.Name, gameName) {
				gameProcesses = append(gameProcesses, proc)
				break
			}
		}
	}

	return gameProcesses, nil
}

// TerminateProcess 终止进程
func (s *Scanner) TerminateProcess(pid int) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("当前只支持 Windows 平台")
	}

	// 使用 taskkill 命令终止进程
	cmd := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", pid))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("终止进程失败 (PID: %d): %w, 输出: %s", pid, err, string(output))
	}

	return nil
}

// CheckProcessRunning 检查指定 PID 的进程是否正在运行
func (s *Scanner) CheckProcessRunning(pid int) (bool, error) {
	processes, err := s.ScanProcesses()
	if err != nil {
		return false, err
	}

	for _, proc := range processes {
		if proc.PID == pid {
			return true, nil
		}
	}

	return false, nil
}

// TerminateWithRetry 带重试的进程终止
func (s *Scanner) TerminateWithRetry(pid int, maxRetries int, retryDelay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := s.TerminateProcess(pid)
		if err == nil {
			// 验证进程是否真正终止
			time.Sleep(100 * time.Millisecond)
			running, _ := s.CheckProcessRunning(pid)
			if !running {
				return nil
			}
		}
		lastErr = err
		time.Sleep(retryDelay)
	}
	return fmt.Errorf("进程终止失败 (PID: %d)，已重试 %d 次: %w", pid, maxRetries, lastErr)
}
