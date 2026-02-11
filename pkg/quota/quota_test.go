package quota

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewQuotaState(t *testing.T) {
	state, err := NewQuotaState("08:00")
	if err != nil {
		t.Fatalf("NewQuotaState() failed: %v", err)
	}

	if state.AccumulatedTime != 0 {
		t.Errorf("Expected AccumulatedTime to be 0, got %d", state.AccumulatedTime)
	}

	if state.LastResetDate == "" {
		t.Error("Expected LastResetDate to be set")
	}

	if state.NextResetTime <= 0 {
		t.Error("Expected NextResetTime to be set")
	}
}

func TestGetAccumulatedMinutes(t *testing.T) {
	state := &QuotaState{
		AccumulatedTime: 60000, // 1 minute = 60000 ms
	}

	minutes := state.GetAccumulatedMinutes()
	if minutes != 1 {
		t.Errorf("Expected 1 minute, got %d", minutes)
	}
}

func TestGetRemainingMinutes(t *testing.T) {
	state := &QuotaState{
		AccumulatedTime: 60000, // 1 minute
	}

	remaining := state.GetRemainingMinutes(120) // 2 hours = 120 minutes
	if remaining != 119 {
		t.Errorf("Expected 119 minutes remaining, got %d", remaining)
	}

	// Test when limit is exceeded
	state.AccumulatedTime = 7200000 // 2 hours = 120 minutes
	remaining = state.GetRemainingMinutes(120)
	if remaining != 0 {
		t.Errorf("Expected 0 minutes remaining when limit exceeded, got %d", remaining)
	}
}

func TestIsLimitExceeded(t *testing.T) {
	tests := []struct {
		name             string
		accumulated      int64
		dailyLimit       int
		expectedExceeded bool
	}{
		{
			name:             "under limit",
			accumulated:      60000, // 1 minute
			dailyLimit:       120,
			expectedExceeded: false,
		},
		{
			name:             "at limit",
			accumulated:      7200000, // 2 hours = 120 minutes
			dailyLimit:       120,
			expectedExceeded: true,
		},
		{
			name:             "over limit",
			accumulated:      7800000, // 2 hours 10 minutes = 130 minutes
			dailyLimit:       120,
			expectedExceeded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &QuotaState{AccumulatedTime: tt.accumulated}
			exceeded := state.IsLimitExceeded(tt.dailyLimit)
			if exceeded != tt.expectedExceeded {
				t.Errorf("IsLimitExceeded() = %v, want %v", exceeded, tt.expectedExceeded)
			}
		})
	}
}

func TestAddTime(t *testing.T) {
	state := &QuotaState{AccumulatedTime: 60000}

	state.AddTime(60000) // Add 1 minute

	if state.AccumulatedTime != 120000 {
		t.Errorf("Expected AccumulatedTime to be 120000, got %d", state.AccumulatedTime)
	}
}

func TestShouldReset(t *testing.T) {
	state, _ := NewQuotaState("08:00")

	// 测试新的一天的情况（需要重置）
	state.LastResetDate = "2024-01-01"
	shouldReset, _ := state.ShouldReset("08:00")
	if !shouldReset {
		t.Errorf("ShouldReset() = %v, want true (new day)", shouldReset)
	}

	// 测试同一天的情况（LastResetDate 为今天）
	state.LastResetDate = time.Now().Format("2006-01-02")
	shouldReset, _ = state.ShouldReset("08:00")
	// 这个测试取决于当前时间是否已过重置时间
	// 如果当前时间在重置时间之前，shouldReset 应该是 false
	// 如果当前时间在重置时间之后，shouldReset 应该是 true
	t.Logf("ShouldReset() for same day = %v (current time: %v)", shouldReset, time.Now())
}

func TestReset(t *testing.T) {
	state := &QuotaState{
		AccumulatedTime: 7200000, // 2 hours
	}

	err := state.Reset("08:00")
	if err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}

	if state.AccumulatedTime != 0 {
		t.Errorf("Expected AccumulatedTime to be 0 after reset, got %d", state.AccumulatedTime)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// 创建配额状态
	state, err := NewQuotaState("08:00")
	if err != nil {
		t.Fatalf("NewQuotaState() failed: %v", err)
	}

	state.AddTime(60000) // Add 1 minute

	// 保存状态
	err = state.SaveToFile(statePath)
	if err != nil {
		t.Fatalf("SaveToFile() failed: %v", err)
	}

	// 加载状态
	loadedState, err := LoadFromFile(statePath)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	if loadedState.AccumulatedTime != state.AccumulatedTime {
		t.Errorf("Expected AccumulatedTime to be %d, got %d", state.AccumulatedTime, loadedState.AccumulatedTime)
	}
}

func TestLoadFromFile_NotExists(t *testing.T) {
	state, err := LoadFromFile("/nonexistent/path/state.json")
	if err != nil {
		t.Errorf("LoadFromFile() should not error for non-existent file, got: %v", err)
	}

	if state != nil {
		t.Error("LoadFromFile() should return nil for non-existent file")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		state   QuotaState
		wantErr bool
	}{
		{
			name: "valid state",
			state: QuotaState{
				LastResetDate:   "2024-01-01",
				AccumulatedTime: 60000,
				LastUpdated:     time.Now().Unix(),
				NextResetTime:   time.Now().Add(24 * time.Hour).Unix(),
			},
			wantErr: false,
		},
		{
			name: "missing last reset date",
			state: QuotaState{
				LastResetDate:   "",
				AccumulatedTime: 60000,
				LastUpdated:     time.Now().Unix(),
				NextResetTime:   time.Now().Add(24 * time.Hour).Unix(),
			},
			wantErr: true,
		},
		{
			name: "negative accumulated time",
			state: QuotaState{
				LastResetDate:   "2024-01-01",
				AccumulatedTime: -100,
				LastUpdated:     time.Now().Unix(),
				NextResetTime:   time.Now().Add(24 * time.Hour).Unix(),
			},
			wantErr: true,
		},
		{
			name: "invalid last updated time",
			state: QuotaState{
				LastResetDate:   "2024-01-01",
				AccumulatedTime: 60000,
				LastUpdated:     0,
				NextResetTime:   time.Now().Add(24 * time.Hour).Unix(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckWarningThresholds(t *testing.T) {
	tests := []struct {
		name            string
		dailyLimit      int
		firstThreshold  int
		finalThreshold  int
		accumulatedTime int64
		expectFirst     bool
		expectFinal     bool
	}{
		{
			name:            "no warning",
			dailyLimit:      120,
			firstThreshold:  15,
			finalThreshold:  5,
			accumulatedTime: 60000, // 1 minute
			expectFirst:     false,
			expectFinal:     false,
		},
		{
			name:            "first warning",
			dailyLimit:      120,
			firstThreshold:  15,
			finalThreshold:  5,
			accumulatedTime: 6600000, // 110 minutes
			expectFirst:     true,
			expectFinal:     false,
		},
		{
			name:            "final warning",
			dailyLimit:      120,
			firstThreshold:  15,
			finalThreshold:  5,
			accumulatedTime: 6900000, // 115 minutes
			expectFirst:     false,
			expectFinal:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &QuotaState{AccumulatedTime: tt.accumulatedTime}
			first, final := state.CheckWarningThresholds(tt.dailyLimit, tt.firstThreshold, tt.finalThreshold)
			if first != tt.expectFirst {
				t.Errorf("CheckWarningThresholds() first = %v, want %v", first, tt.expectFirst)
			}
			if final != tt.expectFinal {
				t.Errorf("CheckWarningThresholds() final = %v, want %v", final, tt.expectFinal)
			}
		})
	}
}
