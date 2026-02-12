package quota

import (
	"encoding/json"
	"fmt"
	"github.com/yourusername/game-control/pkg/config"
	"os"
	"sync"
	"time"
)

// QuotaState 配额状态
type QuotaState struct {
	mu  sync.Mutex
	cfg *config.Config

	AccumulatedTime      int64 `json:"accumulatedTime"`      // 累计游戏时间（秒）
	LastResetTime        int64 `json:"lastResetTime"`        // 上次重置时间（Unix 时间戳）
	NextResetTime        int64 `json:"nextResetTime"`        // 下次重置时间（Unix 时间戳）
	FirstWarningNotified bool  `json:"firstWarningNotified"` // 首次警告是否已提示
	FinalWarningNotified bool  `json:"finalWarningNotified"` // 最后警告是否已提示
	LimitNotified        bool  `json:"limitNotified"`        // 超限是否已提示
}

// NewQuotaState 创建新的配额状态
func NewQuotaState(cfg *config.Config) (*QuotaState, error) {
	now := time.Now()

	// 解析重置时间
	resetTimeParsed, err := time.Parse("15:04", cfg.ResetTime)
	if err != nil {
		return nil, fmt.Errorf("无效的重置时间格式: %w", err)
	}

	// 计算下次重置时间
	nextReset := time.Date(now.Year(), now.Month(), now.Day(),
		resetTimeParsed.Hour(), resetTimeParsed.Minute(), 0, 0, now.Location())

	// 如果今天的重置时间已过，则设置为明天
	if now.After(nextReset) {
		nextReset = nextReset.Add(24 * time.Hour)
	}

	return &QuotaState{
		cfg:             cfg,
		AccumulatedTime: 0,
		LastResetTime:   now.Unix(),
		NextResetTime:   nextReset.Unix(),
	}, nil
}

// GetAccumulatedMinutes 获取累计游戏时间（分钟）
func (q *QuotaState) GetAccumulatedMinutes() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return int(q.AccumulatedTime / 60)
}

// GetRemainingMinutes 获取剩余可用时间（分钟）
func (q *QuotaState) GetRemainingMinutes() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	accumulated := int(q.AccumulatedTime / 60)
	remaining := q.cfg.DailyLimit - accumulated
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsLimitExceeded 检查是否超过时间限制
func (q *QuotaState) IsLimitExceeded() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return int(q.AccumulatedTime/60) >= q.cfg.DailyLimit
}

// AddTime 增加累计时间（秒）
func (q *QuotaState) AddTime(seconds int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.AccumulatedTime += seconds
}

// ShouldReset 检查是否应该重置配额
func (q *QuotaState) ShouldReset() (bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 使用已存储的下次重置时间
	return time.Now().After(time.Unix(q.NextResetTime, 0)), nil
}

// Reset 重置配额
func (q *QuotaState) Reset() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	q.AccumulatedTime = 0
	q.LastResetTime = now.Unix()
	q.FirstWarningNotified = false
	q.FinalWarningNotified = false
	q.LimitNotified = false

	// 重新计算下次重置时间
	resetTimeParsed, err := time.Parse("15:04", q.cfg.ResetTime)
	if err != nil {
		return fmt.Errorf("无效的重置时间格式: %w", err)
	}

	nextReset := time.Date(now.Year(), now.Month(), now.Day(),
		resetTimeParsed.Hour(), resetTimeParsed.Minute(), 0, 0, now.Location())

	// 如果今天的重置时间已过，则设置为明天
	if now.After(nextReset) {
		nextReset = nextReset.Add(24 * time.Hour)
	}

	q.NextResetTime = nextReset.Unix()

	return nil
}

// TimeUntilNextReset 获取距离下次重置的时间
func (q *QuotaState) TimeUntilNextReset() time.Duration {
	q.mu.Lock()
	defer q.mu.Unlock()
	return time.Until(time.Unix(q.NextResetTime, 0))
}

// SaveToFile 保存状态到文件
func (q *QuotaState) SaveToFile() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return fmt.Errorf("无法序列化状态: %w", err)
	}

	if err := os.WriteFile(q.cfg.StateFile, data, 0644); err != nil {
		return fmt.Errorf("无法写入状态文件: %w", err)
	}

	return nil
}

// LoadFromFile 从文件加载状态
func LoadFromFile(cfg *config.Config) (*QuotaState, error) {
	path := cfg.StateFile
	// 如果文件不存在，返回错误
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("状态文件不存在: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取状态文件: %w", err)
	}

	var state QuotaState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("无法解析状态文件: %w", err)
	}
	state.cfg = cfg

	return &state, nil
}

// Validate 验证状态完整性
func (q *QuotaState) Validate() error {
	if q.AccumulatedTime < 0 {
		return fmt.Errorf("累计时间不能为负数")
	}

	if q.LastResetTime <= 0 {
		return fmt.Errorf("无效的更新时间")
	}

	if q.NextResetTime <= 0 {
		return fmt.Errorf("无效的下次重置时间")
	}

	return nil
}

// ConsumeWarningNotifications 检查并消费警告阈值，确保每个阈值每天只触发一次
func (q *QuotaState) ConsumeWarningNotifications() (first, final bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	accumulated := int(q.AccumulatedTime / 60)
	remaining := q.cfg.DailyLimit - accumulated
	if remaining < 0 {
		remaining = 0
	}

	if remaining <= q.cfg.FinalThreshold {
		if !q.FinalWarningNotified {
			q.FinalWarningNotified = true
			final = true
		}
		return
	}

	if remaining <= q.cfg.FirstThreshold && remaining > q.cfg.FinalThreshold {
		if !q.FirstWarningNotified {
			q.FirstWarningNotified = true
			first = true
		}
	}

	return
}

// ConsumeLimitNotification 检查并消费超限通知，确保每天只触发一次
func (q *QuotaState) ConsumeLimitNotification() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if int(q.AccumulatedTime/60) < q.cfg.DailyLimit {
		return false
	}
	if q.LimitNotified {
		return false
	}
	q.LimitNotified = true
	return true
}
