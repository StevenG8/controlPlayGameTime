package process

import (
	"testing"
	"time"
)

func TestNewProcessTracker(t *testing.T) {
	tracker := NewProcessTracker()
	if tracker == nil {
		t.Error("NewProcessTracker() should return a non-nil tracker")
	}
}

func TestStartSession(t *testing.T) {
	tracker := NewProcessTracker()
	tracker.StartSession(1234, "game.exe")

	session, exists := tracker.GetSession(1234)
	if !exists {
		t.Error("Session should exist after StartSession")
	}

	if session.PID != 1234 {
		t.Errorf("Expected PID to be 1234, got %d", session.PID)
	}

	if session.Name != "game.exe" {
		t.Errorf("Expected Name to be 'game.exe', got %s", session.Name)
	}

	if !session.IsActive {
		t.Error("Session should be active after StartSession")
	}
}

func TestEndSession(t *testing.T) {
	tracker := NewProcessTracker()
	tracker.StartSession(1234, "game.exe")

	// 等待一段时间
	time.Sleep(10 * time.Millisecond)

	duration, err := tracker.EndSession(1234)
	if err != nil {
		t.Fatalf("EndSession() failed: %v", err)
	}

	if duration < 10 {
		t.Errorf("Expected duration to be at least 10ms, got %d", duration)
	}

	session, _ := tracker.GetSession(1234)
	if session.IsActive {
		t.Error("Session should not be active after EndSession")
	}
}

func TestEndSession_NotExists(t *testing.T) {
	tracker := NewProcessTracker()
	_, err := tracker.EndSession(9999)
	if err == nil {
		t.Error("EndSession() should return error for non-existent session")
	}
}

func TestGetActiveSessionCount(t *testing.T) {
	tracker := NewProcessTracker()

	tracker.StartSession(1234, "game1.exe")
	tracker.StartSession(5678, "game2.exe")

	count := tracker.GetActiveSessionCount()
	if count != 2 {
		t.Errorf("Expected 2 active sessions, got %d", count)
	}

	tracker.EndSession(1234)
	count = tracker.GetActiveSessionCount()
	if count != 1 {
		t.Errorf("Expected 1 active session after ending one, got %d", count)
	}
}

func TestGetTotalActiveDuration(t *testing.T) {
	tracker := NewProcessTracker()

	// 没有活跃会话时应该返回 0
	duration := tracker.GetTotalActiveDuration()
	if duration != 0 {
		t.Errorf("Expected 0 duration with no active sessions, got %d", duration)
	}

	// 添加一个活跃会话
	tracker.StartSession(1234, "game.exe")
	time.Sleep(10 * time.Millisecond)

	duration = tracker.GetTotalActiveDuration()
	if duration < 10 {
		t.Errorf("Expected duration to be at least 10ms, got %d", duration)
	}

	// 添加第二个活跃会话（应该不增加总时间）
	tracker.StartSession(5678, "game2.exe")
	time.Sleep(10 * time.Millisecond)

	duration2 := tracker.GetTotalActiveDuration()
	// 时间应该基于最早启动的会话
	if duration2 < duration {
		t.Errorf("Duration should not decrease, was %d now %d", duration, duration2)
	}
}

func TestRemoveSession(t *testing.T) {
	tracker := NewProcessTracker()
	tracker.StartSession(1234, "game.exe")

	tracker.RemoveSession(1234)

	_, exists := tracker.GetSession(1234)
	if exists {
		t.Error("Session should not exist after RemoveSession")
	}
}

func TestCleanupInactiveSessions(t *testing.T) {
	tracker := NewProcessTracker()
	tracker.StartSession(1234, "game1.exe")
	tracker.StartSession(5678, "game2.exe")

	tracker.EndSession(1234)

	tracker.CleanupInactiveSessions()

	_, exists := tracker.GetSession(1234)
	if exists {
		t.Error("Inactive session should be removed after CleanupInactiveSessions")
	}

	_, exists = tracker.GetSession(5678)
	if !exists {
		t.Error("Active session should still exist after CleanupInactiveSessions")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name         string
		milliseconds int64
		expected     string
	}{
		{
			name:         "seconds only",
			milliseconds: 5000,
			expected:     "5s",
		},
		{
			name:         "minutes and seconds",
			milliseconds: 60000 + 30000, // 1m 30s
			expected:     "1m 30s",
		},
		{
			name:         "hours, minutes and seconds",
			milliseconds: 3600000 + 120000 + 30000, // 1h 2m 30s
			expected:     "1h 2m 30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.milliseconds)
			if result != tt.expected {
				t.Errorf("FormatDuration() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		name         string
		milliseconds int64
		expected     string
	}{
		{
			name:         "minutes only",
			milliseconds: 30000,
			expected:     "0m",
		},
		{
			name:         "minutes only",
			milliseconds: 60000, // 1m
			expected:     "1m",
		},
		{
			name:         "hours and minutes",
			milliseconds: 3600000 + 30000, // 1h 30s
			expected:     "1h 0m",
		},
		{
			name:         "hours and minutes",
			milliseconds: 3600000 + 120000, // 1h 2m
			expected:     "1h 2m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDurationShort(tt.milliseconds)
			if result != tt.expected {
				t.Errorf("FormatDurationShort() = %s, want %s", result, tt.expected)
			}
		})
	}
}
