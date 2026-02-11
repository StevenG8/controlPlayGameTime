package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DailyLimit != 120 {
		t.Errorf("预期每日限制为120分钟，实际为 %d", cfg.DailyLimit)
	}

	if cfg.ResetTime != "08:00" {
		t.Errorf("预期重置时间为08:00，实际为 %s", cfg.ResetTime)
	}

	if len(cfg.Games) < 2 {
		t.Errorf("预期至少2个默认游戏，实际为 %d", len(cfg.Games))
	}
}

func TestLoadFromFile_FileNotExist(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "nonexistent.yaml")
	cfg, err := LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("文件不存在时应返回默认配置，但出现错误: %v", err)
	}

	if cfg.DailyLimit != 120 {
		t.Errorf("文件不存在时应返回默认配置，每日限制应为120，实际为 %d", cfg.DailyLimit)
	}
}

func TestLoadFromFile_ValidFile(t *testing.T) {
	// 创建临时YAML文件
	yamlContent := `dailyLimit: 180
resetTime: "09:00"
games:
  - "game1.exe"
  - "game2.exe"
firstThreshold: 20
finalThreshold: 10
stateFile: "test-state.json"
logFile: "test.log"`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.yaml")

	if err := os.WriteFile(tempFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("无法创建临时文件: %v", err)
	}

	cfg, err := LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("加载有效配置文件失败: %v", err)
	}

	if cfg.DailyLimit != 180 {
		t.Errorf("预期每日限制为180分钟，实际为 %d", cfg.DailyLimit)
	}

	if cfg.ResetTime != "09:00" {
		t.Errorf("预期重置时间为09:00，实际为 %s", cfg.ResetTime)
	}

	if len(cfg.Games) != 2 {
		t.Errorf("预期2个游戏，实际为 %d", len(cfg.Games))
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		DailyLimit:     120,
		ResetTime:      "08:00",
		Games:          []string{"game.exe"},
		FirstThreshold: 15,
		FinalThreshold: 5,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}
}

func TestValidate_InvalidDailyLimit(t *testing.T) {
	cfg := &Config{
		DailyLimit:     0,
		ResetTime:      "08:00",
		Games:          []string{"game.exe"},
		FirstThreshold: 15,
		FinalThreshold: 5,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("预期无效的每日限制应返回错误")
	}
}

func TestValidate_InvalidResetTime(t *testing.T) {
	cfg := &Config{
		DailyLimit:     120,
		ResetTime:      "25:00",
		Games:          []string{"game.exe"},
		FirstThreshold: 15,
		FinalThreshold: 5,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("预期无效的重置时间应返回错误")
	}
}

func TestValidate_EmptyGames(t *testing.T) {
	cfg := &Config{
		DailyLimit:     120,
		ResetTime:      "08:00",
		Games:          []string{},
		FirstThreshold: 15,
		FinalThreshold: 5,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("预期空游戏列表应返回错误")
	}
}

func TestValidate_InvalidThresholds(t *testing.T) {
	cfg := &Config{
		DailyLimit:     120,
		ResetTime:      "08:00",
		Games:          []string{"game.exe"},
		FirstThreshold: 5,
		FinalThreshold: 15, // 最后阈值大于第一次阈值
	}

	if err := cfg.Validate(); err == nil {
		t.Error("预期无效的阈值应返回错误")
	}
}

func TestSaveToFile(t *testing.T) {
	cfg := DefaultConfig()
	tempFile := filepath.Join(t.TempDir(), "config.yaml")

	if err := cfg.SaveToFile(tempFile); err != nil {
		t.Fatalf("保存配置文件失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("配置文件未创建")
	}

	// 重新加载验证
	loadedCfg, err := LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("重新加载保存的配置文件失败: %v", err)
	}

	if loadedCfg.DailyLimit != cfg.DailyLimit {
		t.Errorf("重新加载的配置不匹配，预期 %d，实际 %d", cfg.DailyLimit, loadedCfg.DailyLimit)
	}
}
