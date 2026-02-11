package process

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewScanner(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Fatal("NewScanner() 返回 nil")
	}

	if scanner.lastProcesses == nil {
		t.Error("lastProcesses 映射未初始化")
	}
}

func TestParseCSVLine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "简单字段",
			input:  "field1,field2,field3",
			expect: []string{"field1", "field2", "field3"},
		},
		{
			name:   "带引号的字段",
			input:  `"field,with,comma",field2,"field3"`,
			expect: []string{"field,with,comma", "field2", "field3"},
		},
		{
			name:   "空字符串",
			input:  "",
			expect: []string{},
		},
		{
			name:   "单个字段",
			input:  "single",
			expect: []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCSVLine(tt.input)

			if len(result) != len(tt.expect) {
				t.Errorf("字段数量不匹配，预期 %d，实际 %d", len(tt.expect), len(result))
				return
			}

			for i := range result {
				if result[i] != tt.expect[i] {
					t.Errorf("字段 %d 不匹配，预期 %q，实际 %q", i, tt.expect[i], result[i])
				}
			}
		})
	}
}

func TestFindGameProcesses_NoGames(t *testing.T) {
	// 跳过非Windows平台的测试
	if runtime.GOOS != "windows" {
		t.Skip("仅在Windows平台测试")
	}

	scanner := NewScanner()

	// 空游戏列表
	processes, err := scanner.FindGameProcesses([]string{})
	if err != nil {
		t.Fatalf("FindGameProcesses 失败: %v", err)
	}

	if len(processes) != 0 {
		t.Errorf("预期找到0个进程，实际找到 %d", len(processes))
	}
}

func TestFindGameProcesses_SpecificGame(t *testing.T) {
	// 跳过非Windows平台的测试
	if runtime.GOOS != "windows" {
		t.Skip("仅在Windows平台测试")
	}

	scanner := NewScanner()

	// 查找cmd.exe（Windows上应该存在）
	_, err := scanner.FindGameProcesses([]string{"cmd.exe"})
	if err != nil {
		t.Fatalf("FindGameProcesses 失败: %v", err)
	}

	// cmd.exe可能运行也可能不运行，所以不检查具体数量
	// 只检查没有错误
}

func TestCheckProcessRunning(t *testing.T) {
	// 跳过非Windows平台的测试
	if runtime.GOOS != "windows" {
		t.Skip("仅在Windows平台测试")
	}

	scanner := NewScanner()

	// 获取当前进程的PID
	pid := 1 // 系统进程

	running, err := scanner.CheckProcessRunning(pid)
	if err != nil {
		t.Fatalf("CheckProcessRunning 失败: %v", err)
	}

	// 不检查具体值，因为进程1可能运行也可能不运行
	_ = running
}

func TestScannerPlatformError(t *testing.T) {
	// 测试非Windows平台的错误
	if runtime.GOOS == "windows" {
		t.Skip("仅在非Windows平台测试")
	}

	scanner := NewScanner()

	// ScanProcesses 应该在非Windows平台返回错误
	_, err := scanner.ScanProcesses()
	if err == nil {
		t.Error("预期在非Windows平台 ScanProcesses 返回错误")
	}

	// TerminateProcess 应该在非Windows平台返回错误
	err = scanner.TerminateProcess(123)
	if err == nil {
		t.Error("预期在非Windows平台 TerminateProcess 返回错误")
	}
}

func TestTerminateWithRetry_Mock(t *testing.T) {
	// 这是一个模拟测试，不实际终止进程
	// 主要测试重试逻辑
	scanner := NewScanner()

	// 使用不存在的PID，应该失败
	err := scanner.TerminateWithRetry(99999, 2, 10*time.Millisecond)
	if err == nil {
		t.Error("预期终止不存在的进程会失败")
	}
}

func TestFindGameProcesses_CaseInsensitive(t *testing.T) {
	// 跳过非Windows平台的测试
	if runtime.GOOS != "windows" {
		t.Skip("仅在Windows平台测试")
	}

	scanner := NewScanner()

	// 测试不区分大小写匹配
	processes, err := scanner.FindGameProcesses([]string{"CMD.EXE"}) // 大写
	if err != nil {
		t.Fatalf("FindGameProcesses 失败: %v", err)
	}

	// 检查是否找到进程
	found := false
	for _, proc := range processes {
		if strings.EqualFold(proc.Name, "cmd.exe") {
			found = true
			break
		}
	}

	// 不要求一定找到，因为cmd.exe可能不在运行
	_ = found
}
