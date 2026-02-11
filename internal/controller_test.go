package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/game-control/pkg/config"
	"github.com/yourusername/game-control/pkg/logger"
	"github.com/yourusername/game-control/pkg/quota"
)

func TestNewController(t *testing.T) {
	cfg := &config.Config{
		TimeLimit: config.TimeLimitConfig{DailyLimit: 120},
		ResetTime: "08:00",
		Games:     []string{"game.exe"},
		StateFile: "test-state.json",
	}

	qState, _ := quota.NewQuotaState("08:00")
	log, _ := logger.NewLogger("")

	controller := NewController(cfg, qState, log)

	if controller == nil {
		t.Error("NewController() should return a non-nil controller")
	}
	log.Close()
}

func TestControllerGetStatus(t *testing.T) {
	cfg := &config.Config{
		TimeLimit: config.TimeLimitConfig{DailyLimit: 120},
		ResetTime: "08:00",
		Games:     []string{"game.exe"},
		StateFile: "test-state.json",
	}

	qState, _ := quota.NewQuotaState("08:00")
	qState.AddTime(60000) // Add 1 minute

	log, _ := logger.NewLogger("")
	controller := NewController(cfg, qState, log)

	status := controller.GetStatus()

	if status.AccumulatedTime != 1 {
		t.Errorf("Expected AccumulatedTime to be 1 minute, got %d", status.AccumulatedTime)
	}

	if status.RemainingTime != 119 {
		t.Errorf("Expected RemainingTime to be 119 minutes, got %d", status.RemainingTime)
	}

	if status.DailyLimit != 120 {
		t.Errorf("Expected DailyLimit to be 120 minutes, got %d", status.DailyLimit)
	}

	log.Close()
}

func TestControllerRunAndStop(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")

	// 创建配置文件
	cfg := config.DefaultConfig()
	cfg.TimeLimit.DailyLimit = 1 // 1 minute for quick testing
	cfg.StateFile = statePath
	cfg.LogFile = filepath.Join(tmpDir, "test.log")

	if err := cfg.SaveToFile(configPath); err != nil {
		t.Fatalf("SaveToFile() failed: %v", err)
	}

	// 创建日志记录器
	log, err := logger.NewLogger(cfg.LogFile)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer log.Close()

	// 创建配额状态
	qState, _ := quota.NewQuotaState(cfg.ResetTime)

	// 创建控制器
	controller := NewController(cfg, qState, log)

	// 启动控制器（在单独的 goroutine 中）
	done := make(chan error, 1)
	go func() {
		done <- controller.Run()
	}()

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	// 发送停止信号
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess() failed: %v", err)
	}

	// 模拟 Ctrl+C 信号
	p.Signal(os.Interrupt)

	// 等待控制器停止
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Controller.Run() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Controller did not stop within 5 seconds")
	}

	// 验证状态文件是否被保存
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file should exist after controller stops")
	}
}

func TestControllerLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	cfg := &config.Config{
		TimeLimit: config.TimeLimitConfig{DailyLimit: 120},
		ResetTime: "08:00",
		Games:     []string{"game.exe"},
		StateFile: statePath,
	}

	log, _ := logger.NewLogger("")
	defer log.Close()

	// 创建初始配额状态
	qState, _ := quota.NewQuotaState(cfg.ResetTime)

	// 创建控制器
	controller := NewController(cfg, qState, log)

	// 获取初始状态
	status1 := controller.GetStatus()
	if status1.AccumulatedTime != 0 {
		t.Errorf("Expected initial AccumulatedTime to be 0, got %d", status1.AccumulatedTime)
	}

	// 模拟添加时间
	controller.quotaState.AddTime(60000)

	// 获取更新后的状态
	status2 := controller.GetStatus()
	if status2.AccumulatedTime != 1 {
		t.Errorf("Expected AccumulatedTime to be 1 minute, got %d", status2.AccumulatedTime)
	}
}

func TestStatusInfo(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	cfg := &config.Config{
		TimeLimit: config.TimeLimitConfig{DailyLimit: 120},
		ResetTime: "08:00",
		Games:     []string{"game.exe"},
		StateFile: statePath,
	}

	qState, _ := quota.NewQuotaState(cfg.ResetTime)
	qState.AddTime(60000) // Add 1 minute

	log, _ := logger.NewLogger("")
	controller := NewController(cfg, qState, log)
	status := controller.GetStatus()

	// 验证所有字段
	if status.AccumulatedTime != 1 {
		t.Errorf("Expected AccumulatedTime to be 1, got %d", status.AccumulatedTime)
	}

	if status.RemainingTime != 119 {
		t.Errorf("Expected RemainingTime to be 119, got %d", status.RemainingTime)
	}

	if status.DailyLimit != 120 {
		t.Errorf("Expected DailyLimit to be 120, got %d", status.DailyLimit)
	}

	if status.ActiveProcessCount != 0 {
		t.Errorf("Expected ActiveProcessCount to be 0, got %d", status.ActiveProcessCount)
	}

	if status.NextResetTime < 0 {
		t.Errorf("Expected NextResetTime to be positive, got %v", status.NextResetTime)
	}

	log.Close()
}
