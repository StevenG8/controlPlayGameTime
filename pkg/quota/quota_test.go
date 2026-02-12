package quota

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/game-control/pkg/config"
)

func createTestConfig() *config.Config {
	return &config.Config{
		DailyLimit:     120, // 2小时
		ResetTime:      "08:00",
		Games:          []string{"game.exe"},
		FirstThreshold: 15,
		FinalThreshold: 5,
		StateFile:      "state.json",
		LogFile:        "game-control.log",
	}
}

func TestNewQuotaState(t *testing.T) {
	cfg := createTestConfig()

	state, err := NewQuotaState(cfg)
	if err != nil {
		t.Fatalf("NewQuotaState 失败: %v", err)
	}

	if state == nil {
		t.Fatal("NewQuotaState 返回 nil")
	}

	if state.GetAccumulatedMinutes() != 0 {
		t.Errorf("新状态累计时间应为0，实际为 %d", state.GetAccumulatedMinutes())
	}
}

func TestNewQuotaState_InvalidResetTime(t *testing.T) {
	cfg := createTestConfig()
	cfg.ResetTime = "25:00" // 无效时间

	_, err := NewQuotaState(cfg)
	if err == nil {
		t.Error("预期无效的重置时间应返回错误")
	}
}

func TestGetAccumulatedMinutes(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 初始应为0
	if minutes := state.GetAccumulatedMinutes(); minutes != 0 {
		t.Errorf("初始累计时间应为0，实际为 %d", minutes)
	}

	// 添加时间
	state.AddTime(300) // 5分钟

	if minutes := state.GetAccumulatedMinutes(); minutes != 5 {
		t.Errorf("添加5分钟后累计时间应为5，实际为 %d", minutes)
	}
}

func TestGetRemainingMinutes(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 初始应为120分钟（每日限制）
	if remaining := state.GetRemainingMinutes(); remaining != 120 {
		t.Errorf("初始剩余时间应为120，实际为 %d", remaining)
	}

	// 添加60分钟
	state.AddTime(3600) // 1小时

	if remaining := state.GetRemainingMinutes(); remaining != 60 {
		t.Errorf("使用1小时后剩余时间应为60，实际为 %d", remaining)
	}

	// 超过限制
	state.AddTime(7200) // 再加2小时

	if remaining := state.GetRemainingMinutes(); remaining != 0 {
		t.Errorf("超过限制后剩余时间应为0，实际为 %d", remaining)
	}
}

func TestIsLimitExceeded(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 初始不应超限
	if state.IsLimitExceeded() {
		t.Error("初始状态不应超限")
	}

	// 刚好达到限制
	state.AddTime(120 * 60) // 120分钟

	if !state.IsLimitExceeded() {
		t.Error("达到限制时应超限")
	}

	// 超过限制
	state.AddTime(60) // 再加1秒

	if !state.IsLimitExceeded() {
		t.Error("超过限制时应超限")
	}
}

func TestAddTime(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 添加1分钟
	state.AddTime(60)

	if accumulated := state.GetAccumulatedMinutes(); accumulated != 1 {
		t.Errorf("添加1分钟后累计时间应为1，实际为 %d", accumulated)
	}

	// 再添加2分钟
	state.AddTime(120)

	if accumulated := state.GetAccumulatedMinutes(); accumulated != 3 {
		t.Errorf("再添加2分钟后累计时间应为3，实际为 %d", accumulated)
	}
}

func TestShouldReset(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 刚创建的状态不应重置
	shouldReset, err := state.ShouldReset()
	if err != nil {
		t.Fatalf("ShouldReset 失败: %v", err)
	}

	if shouldReset {
		t.Error("新创建的状态不应需要重置")
	}
}

func TestReset(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 添加一些时间
	state.AddTime(3600) // 1小时

	// 重置
	if err := state.Reset(); err != nil {
		t.Fatalf("Reset 失败: %v", err)
	}

	// 验证累计时间归零
	if accumulated := state.GetAccumulatedMinutes(); accumulated != 0 {
		t.Errorf("重置后累计时间应为0，实际为 %d", accumulated)
	}

	// 验证剩余时间恢复
	if remaining := state.GetRemainingMinutes(); remaining != 120 {
		t.Errorf("重置后剩余时间应为120，实际为 %d", remaining)
	}
}

func TestGetNextResetTime(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	nextReset := state.GetNextResetTime()
	if nextReset.IsZero() {
		t.Error("下次重置时间不应为零值")
	}

	// 验证时间在未来
	if nextReset.Before(time.Now()) {
		t.Error("下次重置时间应在未来")
	}
}

func TestTimeUntilNextReset(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	duration := state.TimeUntilNextReset()
	if duration <= 0 {
		t.Errorf("距离下次重置时间应为正数，实际为 %v", duration)
	}
}

func TestSaveToFile(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 添加一些时间
	state.AddTime(1800) // 30分钟

	tempFile := filepath.Join(t.TempDir(), "state.json")

	// 保存
	if err := state.SaveToFile(); err != nil {
		t.Fatalf("SaveToFile 失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("状态文件未创建")
	}

	// 验证文件内容
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("读取状态文件失败: %v", err)
	}

	var savedState QuotaState
	if err := json.Unmarshal(data, &savedState); err != nil {
		t.Fatalf("解析状态文件失败: %v", err)
	}

	if savedState.AccumulatedTime != 1800 {
		t.Errorf("保存的累计时间应为1800，实际为 %d", savedState.AccumulatedTime)
	}
}

func TestLoadFromFile(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 添加一些时间
	state.AddTime(2700) // 45分钟

	// 先保存
	if err := state.SaveToFile(); err != nil {
		t.Fatalf("SaveToFile 失败: %v", err)
	}

	// 再加载
	loadedState, err := LoadFromFile(cfg)
	if err != nil {
		t.Fatalf("LoadFromFile 失败: %v", err)
	}

	if loadedState == nil {
		t.Fatal("LoadFromFile 返回 nil")
	}

	// 验证数据
	if loadedState.GetAccumulatedMinutes() != 45 {
		t.Errorf("加载的累计时间应为45，实际为 %d", loadedState.GetAccumulatedMinutes())
	}
}

func TestLoadFromFile_FileNotExist(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "nonexistent.json")

	_, err := loadFromFile(tempFile)
	// LoadFromFile 在文件不存在时返回错误
	if err == nil {
		t.Error("预期加载不存在的文件应返回错误")
	}
}

func TestValidate(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 有效状态
	if err := state.Validate(); err != nil {
		t.Errorf("有效状态验证失败: %v", err)
	}
}

func TestCheckWarningThresholds(t *testing.T) {
	cfg := createTestConfig()
	state, _ := NewQuotaState(cfg)

	// 初始状态（剩余120分钟）不应触发警告
	first, final := state.CheckWarningThresholds()
	if first || final {
		t.Error("初始状态不应触发任何警告")
	}

	// 使用到剩余16分钟（刚好高于第一次阈值15分钟）
	state.AddTime((120 - 16) * 60) // 104分钟

	first, final = state.CheckWarningThresholds()
	// 剩余16分钟 > 15分钟（第一次阈值），不应触发第一次警告
	if first {
		t.Error("剩余16分钟时不应触发第一次警告")
	}
	if final {
		t.Error("剩余16分钟时不应触发最后警告")
	}

	// 使用到剩余14分钟（刚好低于第一次阈值15分钟）
	state.AddTime(2 * 60) // 再加2分钟，从剩余16到剩余14

	first, final = state.CheckWarningThresholds()
	// 剩余14分钟 < 15分钟（第一次阈值），应触发第一次警告
	if !first {
		t.Error("剩余14分钟时应触发第一次警告")
	}
	if final {
		t.Error("剩余14分钟时不应触发最后警告")
	}

	// 使用到剩余4分钟（刚好低于最后阈值5分钟）
	state.AddTime(10 * 60) // 再加10分钟，从剩余14到剩余4

	first, final = state.CheckWarningThresholds()
	// 剩余4分钟 < 5分钟（最后阈值），应触发最后警告
	// 注意：根据 CheckWarningThresholds 实现，当剩余时间 <= FinalThreshold 时，first 应为 false
	// 因为 first = remaining <= FirstThreshold && remaining > FinalThreshold
	if first {
		t.Errorf("剩余4分钟时 first 应为 false，因为 remaining <= FinalThreshold，但实际为 true")
	}
	if !final {
		t.Error("剩余4分钟时应触发最后警告")
	}
}
