package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelDebug LogLevel = "debug"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Event     string    `json:"event,omitempty"`
	Process   string    `json:"process,omitempty"`
	Duration  int64     `json:"duration,omitempty"` // 毫秒
}

// Logger 日志记录器
type Logger struct {
	output *os.File
}

// NewLogger 创建新的日志记录器
func NewLogger(outputPath string) (*Logger, error) {
	var output *os.File
	var err error

	if outputPath == "" {
		output = os.Stdout
	} else {
		output, err = os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("无法打开日志文件: %w", err)
		}
	}

	return &Logger{output: output}, nil
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	if l.output != os.Stdout && l.output != os.Stderr {
		return l.output.Close()
	}
	return nil
}

// log 记录日志
func (l *Logger) log(entry LogEntry) {
	entry.Timestamp = time.Now()
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.output, "无法序列化日志: %v\n", err)
		return
	}
	fmt.Fprintln(l.output, string(data))
}

// Info 记录信息日志
func (l *Logger) Info(message string) {
	l.log(LogEntry{
		Level:   LevelInfo,
		Message: message,
	})
}

// Warn 记录警告日志
func (l *Logger) Warn(message string) {
	l.log(LogEntry{
		Level:   LevelWarn,
		Message: message,
	})
}

// Error 记录错误日志
func (l *Logger) Error(message string) {
	l.log(LogEntry{
		Level:   LevelError,
		Message: message,
	})
}

// Debug 记录调试日志
func (l *Logger) Debug(message string) {
	l.log(LogEntry{
		Level:   LevelDebug,
		Message: message,
	})
}

// LogGameStart 记录游戏启动事件
func (l *Logger) LogGameStart(processName string) {
	l.log(LogEntry{
		Level:   LevelInfo,
		Message: fmt.Sprintf("游戏进程启动: %s", processName),
		Event:   "game_start",
		Process: processName,
	})
}

// LogGameStop 记录游戏停止事件
func (l *Logger) LogGameStop(processName string, duration int64) {
	l.log(LogEntry{
		Level:    LevelInfo,
		Message:  fmt.Sprintf("游戏进程停止: %s, 运行时长: %dms", processName, duration),
		Event:    "game_stop",
		Process:  processName,
		Duration: duration,
	})
}

// LogQuotaReset 记录配额重置事件
func (l *Logger) LogQuotaReset() {
	l.log(LogEntry{
		Level:   LevelInfo,
		Message: "每日游戏时间配额已重置",
		Event:   "quota_reset",
	})
}

// LogLimitExceeded 记录时间限制超限事件
func (l *Logger) LogLimitExceeded() {
	l.log(LogEntry{
		Level:   LevelWarn,
		Message: "每日游戏时间限制已超限，终止游戏进程",
		Event:   "limit_exceeded",
	})
}
