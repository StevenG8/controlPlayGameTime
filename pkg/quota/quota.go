package quota

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// QuotaState 配额状态
type QuotaState struct {
	LastResetDate   string `json:"lastResetDate"`   // 上次重置日期 (YYYY-MM-DD)
	AccumulatedTime int64  `json:"accumulatedTime"` // 累计游戏时间（毫秒）TODO: 单位直接秒就好
	LastUpdated     int64  `json:"lastUpdated"`     // 上次更新时间（Unix 时间戳）
	NextResetTime   int64  `json:"nextResetTime"`   // 下次重置时间（Unix 时间戳）
}

// NewQuotaState 创建新的配额状态
func NewQuotaState(resetTime string) (*QuotaState, error) {
	now := time.Now()
	today := now.Format("2006-01-02")

	// 解析重置时间
	resetTimeParsed, err := time.Parse("15:04", resetTime)
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
		LastResetDate:   today,
		AccumulatedTime: 0,
		LastUpdated:     now.Unix(),
		NextResetTime:   nextReset.Unix(),
	}, nil
}

// GetAccumulatedMinutes 获取累计游戏时间（分钟）
func (q *QuotaState) GetAccumulatedMinutes() int {
	return int(q.AccumulatedTime / 60000)
}

// GetRemainingMinutes 获取剩余可用时间（分钟）
func (q *QuotaState) GetRemainingMinutes(dailyLimit int) int { // TODO: 为什么不直接读取配置呢. 而是由外部传进来？
	accumulated := q.GetAccumulatedMinutes()
	remaining := dailyLimit - accumulated
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsLimitExceeded 检查是否超过时间限制
func (q *QuotaState) IsLimitExceeded(dailyLimit int) bool {
	return q.GetAccumulatedMinutes() >= dailyLimit // TODO: 为什么不直接读取配置呢. 而是由外部传进来？
}

// AddTime 增加累计时间（毫秒）
func (q *QuotaState) AddTime(milliseconds int64) { // TODO: 线程安全问题?
	q.AccumulatedTime += milliseconds
	q.LastUpdated = time.Now().Unix()
}

// ShouldReset 检查是否应该重置配额
func (q *QuotaState) ShouldReset(resetTime string) (bool, error) { // TODO: 不是已经有存下一次重置时间吗？有必要要外部传resetTime吗?
	now := time.Now()
	today := now.Format("2006-01-02")

	// 如果日期已改变，需要重置
	if q.LastResetDate != today {
		return true, nil
	}

	// 检查是否已过重置时间
	resetTimeParsed, err := time.Parse("15:04", resetTime)
	if err != nil {
		return false, fmt.Errorf("无效的重置时间格式: %w", err)
	}

	todayReset := time.Date(now.Year(), now.Month(), now.Day(),
		resetTimeParsed.Hour(), resetTimeParsed.Minute(), 0, 0, now.Location())

	return now.After(todayReset), nil
}

// Reset 重置配额
func (q *QuotaState) Reset(resetTime string) error {
	now := time.Now()
	q.LastResetDate = now.Format("2006-01-02")
	q.AccumulatedTime = 0
	q.LastUpdated = now.Unix()

	// 重新计算下次重置时间
	resetTimeParsed, err := time.Parse("15:04", resetTime)
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

// GetNextResetTime 获取下次重置时间
func (q *QuotaState) GetNextResetTime() time.Time {
	return time.Unix(q.NextResetTime, 0)
}

// TimeUntilNextReset 获取距离下次重置的时间
func (q *QuotaState) TimeUntilNextReset() time.Duration {
	return time.Until(q.GetNextResetTime())
}

// SaveToFile 保存状态到文件
func (q *QuotaState) SaveToFile(path string) error {
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return fmt.Errorf("无法序列化状态: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("无法写入状态文件: %w", err)
	}

	return nil
}

// LoadFromFile 从文件加载状态
func LoadFromFile(path string) (*QuotaState, error) {
	// 如果文件不存在，返回 nil
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取状态文件: %w", err)
	}

	var state QuotaState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("无法解析状态文件: %w", err)
	}

	return &state, nil
}

// Validate 验证状态完整性
func (q *QuotaState) Validate() error {
	if q.LastResetDate == "" {
		return fmt.Errorf("缺少上次重置日期")
	}

	if q.AccumulatedTime < 0 {
		return fmt.Errorf("累计时间不能为负数")
	}

	if q.LastUpdated <= 0 {
		return fmt.Errorf("无效的更新时间")
	}

	if q.NextResetTime <= 0 {
		return fmt.Errorf("无效的下次重置时间")
	}

	return nil
}

// CheckWarningThresholds 检查警告阈值
func (q *QuotaState) CheckWarningThresholds(dailyLimit, firstThreshold, finalThreshold int) (first, final bool) {
	remaining := q.GetRemainingMinutes(dailyLimit)

	first = remaining <= firstThreshold && remaining > finalThreshold
	final = remaining <= finalThreshold

	return
}
