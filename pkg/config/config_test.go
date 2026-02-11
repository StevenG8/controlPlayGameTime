package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.TimeLimit.DailyLimit != 120 {
		t.Errorf("Expected DailyLimit to be 120, got %d", cfg.TimeLimit.DailyLimit)
	}

	if cfg.ResetTime != "08:00" {
		t.Errorf("Expected ResetTime to be '08:00', got %s", cfg.ResetTime)
	}

	if len(cfg.Games) == 0 {
		t.Error("Expected Games to have default values")
	}

	if cfg.Warning.FirstThreshold != 15 {
		t.Errorf("Expected FirstThreshold to be 15, got %d", cfg.Warning.FirstThreshold)
	}

	if cfg.Warning.FinalThreshold != 5 {
		t.Errorf("Expected FinalThreshold to be 5, got %d", cfg.Warning.FinalThreshold)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				TimeLimit: TimeLimitConfig{DailyLimit: 120},
				ResetTime: "08:00",
				Games:     []string{"game.exe"},
				Warning: WarningConfig{
					FirstThreshold: 15,
					FinalThreshold: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "negative daily limit",
			config: Config{
				TimeLimit: TimeLimitConfig{DailyLimit: -10},
				ResetTime: "08:00",
				Games:     []string{"game.exe"},
			},
			wantErr: true,
		},
		{
			name: "invalid reset time",
			config: Config{
				TimeLimit: TimeLimitConfig{DailyLimit: 120},
				ResetTime: "25:00",
				Games:     []string{"game.exe"},
			},
			wantErr: true,
		},
		{
			name: "empty games list",
			config: Config{
				TimeLimit: TimeLimitConfig{DailyLimit: 120},
				ResetTime: "08:00",
				Games:     []string{},
			},
			wantErr: true,
		},
		{
			name: "negative warning threshold",
			config: Config{
				TimeLimit: TimeLimitConfig{DailyLimit: 120},
				ResetTime: "08:00",
				Games:     []string{"game.exe"},
				Warning: WarningConfig{
					FirstThreshold: -5,
					FinalThreshold: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "final threshold greater than first",
			config: Config{
				TimeLimit: TimeLimitConfig{DailyLimit: 120},
				ResetTime: "08:00",
				Games:     []string{"game.exe"},
				Warning: WarningConfig{
					FirstThreshold: 5,
					FinalThreshold: 15,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 创建配置
	cfg := Config{
		TimeLimit: TimeLimitConfig{DailyLimit: 180},
		ResetTime: "09:00",
		Games:     []string{"game1.exe", "game2.exe"},
		Warning: WarningConfig{
			FirstThreshold: 20,
			FinalThreshold: 10,
		},
		StateFile: "state.json",
		LogFile:   "game-control.log",
	}

	// 保存配置
	err := cfg.SaveToFile(configPath)
	if err != nil {
		t.Fatalf("SaveToFile() failed: %v", err)
	}

	// 加载配置
	loadedCfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	// 验证配置
	if loadedCfg.TimeLimit.DailyLimit != cfg.TimeLimit.DailyLimit {
		t.Errorf("Expected DailyLimit to be %d, got %d", cfg.TimeLimit.DailyLimit, loadedCfg.TimeLimit.DailyLimit)
	}

	if loadedCfg.ResetTime != cfg.ResetTime {
		t.Errorf("Expected ResetTime to be %s, got %s", cfg.ResetTime, loadedCfg.ResetTime)
	}

	if len(loadedCfg.Games) != len(cfg.Games) {
		t.Errorf("Expected Games to have %d items, got %d", len(cfg.Games), len(loadedCfg.Games))
	}

	if loadedCfg.Warning.FirstThreshold != cfg.Warning.FirstThreshold {
		t.Errorf("Expected FirstThreshold to be %d, got %d", cfg.Warning.FirstThreshold, loadedCfg.Warning.FirstThreshold)
	}
}

func TestLoadFromFile_NotExists(t *testing.T) {
	// 尝试加载不存在的文件
	cfg, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err != nil {
		t.Errorf("LoadFromFile() should not error for non-existent file, got: %v", err)
	}

	if cfg == nil {
		t.Error("LoadFromFile() should return default config for non-existent file")
	}
}

func TestGetConfigPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}

	expectedDir := filepath.Join(homeDir, ".config", "game-control")
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() failed: %v", err)
	}

	if filepath.Dir(configPath) != expectedDir {
		t.Errorf("Expected config directory to be %s, got %s", expectedDir, filepath.Dir(configPath))
	}

	// 验证目录是否已创建
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("GetConfigPath() should create the config directory")
	}
}
