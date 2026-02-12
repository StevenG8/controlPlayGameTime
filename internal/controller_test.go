package internal

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/game-control/pkg/config"
	"github.com/yourusername/game-control/pkg/logger"
	"github.com/yourusername/game-control/pkg/process"
	"github.com/yourusername/game-control/pkg/quota"
)

type mockScanner struct {
	findGameProcessesFunc func([]string) ([]process.ProcessInfo, error)
	terminateWithRetryFn  func(int, int, time.Duration) error
}

func (m *mockScanner) FindGameProcesses(games []string) ([]process.ProcessInfo, error) {
	if m.findGameProcessesFunc != nil {
		return m.findGameProcessesFunc(games)
	}
	return []process.ProcessInfo{}, nil
}

func (m *mockScanner) TerminateWithRetry(pid int, maxRetries int, retryDelay time.Duration) error {
	if m.terminateWithRetryFn != nil {
		return m.terminateWithRetryFn(pid, maxRetries, retryDelay)
	}
	return nil
}

type fakeNotifier struct {
	firstCalls int
	finalCalls int
	limitCalls int
}

func (f *fakeNotifier) NotifyFirstWarning(remainingMinutes int) error {
	f.firstCalls++
	return nil
}

func (f *fakeNotifier) NotifyFinalWarning(remainingMinutes int) error {
	f.finalCalls++
	return nil
}

func (f *fakeNotifier) NotifyLimitExceeded() error {
	f.limitCalls++
	return nil
}

func createTestController(t *testing.T) (*Controller, *mockScanner, *fakeNotifier, *quota.QuotaState) {
	t.Helper()

	tempDir := t.TempDir()
	cfg := &config.Config{
		DailyLimit:     120,
		ResetTime:      "08:00",
		Games:          []string{"game.exe"},
		FirstThreshold: 15,
		FinalThreshold: 5,
		StateFile:      filepath.Join(tempDir, "state.json"),
		LogFile:        filepath.Join(tempDir, "test.log"),
	}

	qState, err := quota.NewQuotaState(cfg)
	if err != nil {
		t.Fatalf("创建测试配额状态失败: %v", err)
	}
	log, err := logger.NewLogger(cfg.LogFile)
	if err != nil {
		t.Fatalf("创建测试日志器失败: %v", err)
	}
	mock := &mockScanner{}
	n := &fakeNotifier{}
	c := NewControllerWithDeps(cfg, qState, log, mock, n)
	return c, mock, n, qState
}

func TestControllerTick_FirstWarningNotifyOnce(t *testing.T) {
	controller, mock, n, qState := createTestController(t)
	defer controller.logger.Close()

	mock.findGameProcessesFunc = func(games []string) ([]process.ProcessInfo, error) {
		return []process.ProcessInfo{}, nil
	}

	qState.AddTime(int64((120 - 14) * 60)) // remaining = 14
	controller.tick()
	controller.tick()

	if n.firstCalls != 1 {
		t.Fatalf("首次警告应只弹一次，实际 %d", n.firstCalls)
	}
	if n.finalCalls != 0 {
		t.Fatalf("不应触发最后警告，实际 %d", n.finalCalls)
	}
}

func TestControllerTick_FinalWarningNotifyOnce(t *testing.T) {
	controller, mock, n, qState := createTestController(t)
	defer controller.logger.Close()

	mock.findGameProcessesFunc = func(games []string) ([]process.ProcessInfo, error) {
		return []process.ProcessInfo{}, nil
	}

	qState.AddTime(int64((120 - 4) * 60)) // remaining = 4
	controller.tick()
	controller.tick()

	if n.finalCalls != 1 {
		t.Fatalf("最后警告应只弹一次，实际 %d", n.finalCalls)
	}
}

func TestControllerTick_LimitExceededNotifyAndTerminate(t *testing.T) {
	controller, mock, n, qState := createTestController(t)
	defer controller.logger.Close()

	mock.findGameProcessesFunc = func(games []string) ([]process.ProcessInfo, error) {
		return []process.ProcessInfo{{PID: 1234, Name: "game.exe", StartTime: time.Now()}}, nil
	}

	terminateCalls := 0
	mock.terminateWithRetryFn = func(pid int, maxRetries int, retryDelay time.Duration) error {
		terminateCalls++
		return nil
	}

	qState.AddTime(120 * 60)
	controller.tick()
	controller.tick()

	if n.limitCalls != 1 {
		t.Fatalf("超限弹窗应只弹一次，实际 %d", n.limitCalls)
	}
	if terminateCalls == 0 {
		t.Fatal("超限后应尝试终止进程")
	}
}

func TestControllerStatus(t *testing.T) {
	controller, mock, _, qState := createTestController(t)
	defer controller.logger.Close()

	mock.findGameProcessesFunc = func(games []string) ([]process.ProcessInfo, error) {
		return []process.ProcessInfo{{PID: 1, Name: "game.exe", StartTime: time.Now()}}, nil
	}

	qState.AddTime(1800)
	status := controller.GetStatus()

	if status.AccumulatedTime != 30 {
		t.Errorf("状态累计时间应为30，实际为 %d", status.AccumulatedTime)
	}
	if status.RemainingTime != 90 {
		t.Errorf("状态剩余时间应为90，实际为 %d", status.RemainingTime)
	}
	if status.ActiveProcessCount != 1 {
		t.Errorf("活跃进程数量应为1，实际为 %d", status.ActiveProcessCount)
	}
}
