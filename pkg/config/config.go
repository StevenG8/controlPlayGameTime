package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	TimeLimit TimeLimitConfig `yaml:"timeLimit"`
	ResetTime string          `yaml:"resetTime"` // 格式: "08:00"
	Games     []string        `yaml:"games"`     // 游戏进程名称列表
	Warning   WarningConfig   `yaml:"warning"`
	StateFile string          `yaml:"stateFile"` // 状态文件路径
	LogFile   string          `yaml:"logFile"`   // 日志文件路径
}

// TimeLimitConfig 时间限制配置
type TimeLimitConfig struct {
	DailyLimit int `yaml:"dailyLimit"` // 每日游戏时间限制（分钟）
}

// WarningConfig 警告配置
type WarningConfig struct {
	FirstThreshold int `yaml:"firstThreshold"` // 第一次警告阈值（分钟）
	FinalThreshold int `yaml:"finalThreshold"` // 最后警告阈值（分钟）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		TimeLimit: TimeLimitConfig{
			DailyLimit: 120, // 默认 2 小时
		},
		ResetTime: "08:00",
		Games: []string{
			"game.exe",
			"steam.exe",
		},
		Warning: WarningConfig{
			FirstThreshold: 15, // 剩余 15 分钟时警告
			FinalThreshold: 5,  // 剩余 5 分钟时警告
		},
		StateFile: "state.json",
		LogFile:   "game-control.log",
	}
}

// GetConfigPath 获取默认配置文件路径
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "game-control")
	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建配置目录: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// LoadFromFile 从文件加载配置
func LoadFromFile(path string) (*Config, error) {
	// 如果文件不存在，返回默认配置
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("无法解析配置文件: %w", err)
	}

	return &config, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证每日时间限制
	if c.TimeLimit.DailyLimit <= 0 {
		return fmt.Errorf("每日时间限制必须大于 0")
	}

	// 验证重置时间格式
	_, err := time.Parse("15:04", c.ResetTime)
	if err != nil {
		return fmt.Errorf("重置时间格式无效，应为 HH:MM 格式: %w", err)
	}

	// 验证游戏列表
	if len(c.Games) == 0 {
		return fmt.Errorf("游戏进程列表不能为空")
	}

	// 验证警告阈值
	if c.Warning.FirstThreshold < 0 || c.Warning.FinalThreshold < 0 {
		return fmt.Errorf("警告阈值不能为负数")
	}

	if c.Warning.FinalThreshold > c.Warning.FirstThreshold {
		return fmt.Errorf("最后警告阈值不能大于第一次警告阈值")
	}

	return nil
}

// SaveToFile 保存配置到文件
func (c *Config) SaveToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("无法序列化配置: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("无法创建配置目录: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("无法写入配置文件: %w", err)
	}

	return nil
}
