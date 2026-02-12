package quota

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/game-control/pkg/config"
)

func createTestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		DailyLimit:     120,
		ResetTime:      "08:00",
		Games:          []string{"game.exe"},
		FirstThreshold: 15,
		FinalThreshold: 5,
		StateFile:      filepath.Join(t.TempDir(), "state.json"),
		LogFile:        filepath.Join(t.TempDir(), "game-control.log"),
	}
}

func TestNewQuotaState(t *testing.T) {
	cfg := createTestConfig(t)
	state, err := NewQuotaState(cfg)
	if err != nil {
		t.Fatalf("NewQuotaState 失败: %v", err)
	}
	if state.GetAccumulatedMinutes() != 0 {
		t.Fatalf("新状态累计时间应为0，实际为 %d", state.GetAccumulatedMinutes())
	}
}

func TestResetClearsNotificationFlags(t *testing.T) {
	cfg := createTestConfig(t)
	state, _ := NewQuotaState(cfg)

	state.FirstWarningNotified = true
	state.FinalWarningNotified = true
	state.LimitNotified = true

	if err := state.Reset(); err != nil {
		t.Fatalf("Reset 失败: %v", err)
	}
	if state.FirstWarningNotified || state.FinalWarningNotified || state.LimitNotified {
		t.Fatal("Reset 后通知去重标记应清空")
	}
}

func TestConsumeWarningNotificationsOnce(t *testing.T) {
	cfg := createTestConfig(t)
	state, _ := NewQuotaState(cfg)

	state.AddTime(int64((120 - 14) * 60))
	first, final := state.ConsumeWarningNotifications()
	if !first || final {
		t.Fatalf("剩余14分钟应触发首次警告，first=%v final=%v", first, final)
	}

	first, final = state.ConsumeWarningNotifications()
	if first || final {
		t.Fatalf("同一阈值重复检查不应重复触发，first=%v final=%v", first, final)
	}
}

func TestConsumeFinalWarningOnce(t *testing.T) {
	cfg := createTestConfig(t)
	state, _ := NewQuotaState(cfg)

	state.AddTime(int64((120 - 4) * 60))
	first, final := state.ConsumeWarningNotifications()
	if first || !final {
		t.Fatalf("剩余4分钟应触发最后警告，first=%v final=%v", first, final)
	}

	_, final = state.ConsumeWarningNotifications()
	if final {
		t.Fatal("最后警告应只触发一次")
	}
}

func TestConsumeLimitNotificationOnce(t *testing.T) {
	cfg := createTestConfig(t)
	state, _ := NewQuotaState(cfg)

	state.AddTime(120 * 60)
	if !state.ConsumeLimitNotification() {
		t.Fatal("首次超限应触发通知")
	}
	if state.ConsumeLimitNotification() {
		t.Fatal("超限通知应只触发一次")
	}
}

func TestSaveAndLoadCompatibility(t *testing.T) {
	cfg := createTestConfig(t)
	state, _ := NewQuotaState(cfg)

	state.AddTime(1800)
	state.FirstWarningNotified = true
	if err := state.SaveToFile(); err != nil {
		t.Fatalf("SaveToFile 失败: %v", err)
	}

	loaded, err := LoadFromFile(cfg)
	if err != nil {
		t.Fatalf("LoadFromFile 失败: %v", err)
	}
	if loaded.GetAccumulatedMinutes() != 30 {
		t.Fatalf("加载后累计时间应为30分钟，实际 %d", loaded.GetAccumulatedMinutes())
	}
	if !loaded.FirstWarningNotified {
		t.Fatal("应保留已触发的首次警告标记")
	}
}

func TestLoadOldStateWithoutFlags(t *testing.T) {
	cfg := createTestConfig(t)
	oldState := map[string]any{
		"accumulatedTime": int64(600),
		"lastResetTime":   time.Now().Add(-time.Hour).Unix(),
		"nextResetTime":   time.Now().Add(time.Hour).Unix(),
	}
	data, _ := json.Marshal(oldState)
	if err := os.WriteFile(cfg.StateFile, data, 0644); err != nil {
		t.Fatalf("写入旧状态失败: %v", err)
	}

	loaded, err := LoadFromFile(cfg)
	if err != nil {
		t.Fatalf("加载旧状态失败: %v", err)
	}
	if loaded.FirstWarningNotified || loaded.FinalWarningNotified || loaded.LimitNotified {
		t.Fatal("旧状态加载后新增标记字段应默认 false")
	}
}
