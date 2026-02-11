package process

import (
	"fmt"
	"sync"
	"time"
)

// ProcessTracker 进程追踪器
type ProcessTracker struct {
	mu       sync.Mutex
	sessions map[int]*ProcessSession // PID -> Session
}

// ProcessSession 进程会话
type ProcessSession struct {
	PID       int
	Name      string
	StartTime time.Time
	StopTime  time.Time
	Duration  int64 // 毫秒
	IsActive  bool
}

// NewProcessTracker 创建新的进程追踪器
func NewProcessTracker() *ProcessTracker {
	return &ProcessTracker{
		sessions: make(map[int]*ProcessSession),
	}
}

// StartSession 开始追踪进程会话
func (t *ProcessTracker) StartSession(pid int, name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sessions[pid] = &ProcessSession{
		PID:       pid,
		Name:      name,
		StartTime: time.Now(),
		IsActive:  true,
	}
}

// EndSession 结束追踪进程会话
func (t *ProcessTracker) EndSession(pid int) (int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	session, exists := t.sessions[pid]
	if !exists {
		return 0, fmt.Errorf("进程 %d 的会话不存在", pid)
	}

	now := time.Now()
	duration := now.Sub(session.StartTime).Milliseconds()
	session.StopTime = now
	session.Duration = duration
	session.IsActive = false

	return duration, nil
}

// GetSession 获取进程会话
func (t *ProcessTracker) GetSession(pid int) (*ProcessSession, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	session, exists := t.sessions[pid]
	return session, exists
}

// GetAllSessions 获取所有会话
func (t *ProcessTracker) GetAllSessions() []*ProcessSession {
	t.mu.Lock()
	defer t.mu.Unlock()

	sessions := make([]*ProcessSession, 0, len(t.sessions))
	for _, session := range t.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetActiveSessions 获取活跃会话
func (t *ProcessTracker) GetActiveSessions() []*ProcessSession {
	t.mu.Lock()
	defer t.mu.Unlock()

	sessions := make([]*ProcessSession, 0)
	for _, session := range t.sessions {
		if session.IsActive {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// GetActiveSessionCount 获取活跃会话数量
func (t *ProcessTracker) GetActiveSessionCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	count := 0
	for _, session := range t.sessions {
		if session.IsActive {
			count++
		}
	}
	return count
}

// UpdateActiveSessionDurations 更新活跃会话的运行时间
func (t *ProcessTracker) UpdateActiveSessionDurations() map[int]int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	durations := make(map[int]int64)
	for pid, session := range t.sessions {
		if session.IsActive {
			duration := time.Since(session.StartTime).Milliseconds()
			durations[pid] = duration
		}
	}
	return durations
}

// GetTotalActiveDuration 获取所有活跃进程的总运行时间（去重）
func (t *ProcessTracker) GetTotalActiveDuration() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.sessions) == 0 {
		return 0
	}

	// 如果有活跃进程，返回当前时间到最早启动时间的差值
	var earliestStart time.Time
	hasActive := false

	for _, session := range t.sessions {
		if session.IsActive {
			if !hasActive || session.StartTime.Before(earliestStart) {
				earliestStart = session.StartTime
				hasActive = true
			}
		}
	}

	if !hasActive {
		return 0
	}

	return time.Since(earliestStart).Milliseconds()
}

// GetAccumulatedDuration 获取累计运行时间（包括已结束的会话）
func (t *ProcessTracker) GetAccumulatedDuration() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	var total int64
	for _, session := range t.sessions {
		if session.IsActive {
			total += time.Since(session.StartTime).Milliseconds()
		} else {
			total += session.Duration
		}
	}
	return total
}

// RemoveSession 移除会话
func (t *ProcessTracker) RemoveSession(pid int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.sessions, pid)
}

// CleanupInactiveSessions 清理不活跃的会话
func (t *ProcessTracker) CleanupInactiveSessions() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for pid, session := range t.sessions {
		if !session.IsActive {
			delete(t.sessions, pid)
		}
	}
}

// FormatDuration 格式化持续时间
func FormatDuration(milliseconds int64) string {
	hours := milliseconds / 3600000
	minutes := (milliseconds % 3600000) / 60000
	seconds := (milliseconds % 60000) / 1000

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// FormatDurationShort 简短格式化持续时间 (Xh Ym)
func FormatDurationShort(milliseconds int64) string {
	hours := milliseconds / 3600000
	minutes := (milliseconds % 3600000) / 60000

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}
