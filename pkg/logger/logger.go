package logger

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
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
	zap    *zap.Logger
}

var LogHandle *Logger
var once sync.Once

// NewLogger 创建新的日志记录器
func NewLogger(outputPath string) (*Logger, error) {
	once.Do(func() {
		var output *os.File
		var err error
		if outputPath == "" {
			output = os.Stdout
		} else {
			output, err = os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				panic(fmt.Sprintf("无法打开日志文件: %v", err))
			}
		}

		encoderCfg := zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			MessageKey:     "message",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeDuration: zapcore.MillisDurationEncoder,
		}
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(output),
			zapcore.DebugLevel,
		)

		LogHandle = &Logger{
			output: output,
			zap:    zap.New(core),
		}
	})

	return LogHandle, nil
}

func GetLogger() *Logger {
	if LogHandle == nil {
		panic("not init logger")
	}
	return LogHandle
}

// Infof 使用全局单例记录信息日志
func Infof(format string, args ...any) {
	GetLogger().Infof(format, args...)
}

// Warnf 使用全局单例记录警告日志
func Warnf(format string, args ...any) {
	GetLogger().Warnf(format, args...)
}

// Errorf 使用全局单例记录错误日志
func Errorf(format string, args ...any) {
	GetLogger().Errorf(format, args...)
}

// Debugf 使用全局单例记录调试日志
func Debugf(format string, args ...any) {
	GetLogger().Debugf(format, args...)
}

// LogQuotaReset 使用全局单例记录配额重置事件
func LogQuotaReset() {
	GetLogger().LogQuotaReset()
}

// LogLimitExceeded 使用全局单例记录超限事件
func LogLimitExceeded() {
	GetLogger().LogLimitExceeded()
}

// Close 关闭全局单例日志器
func Close() error {
	return GetLogger().Close()
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	if l != nil && l.zap != nil {
		_ = l.zap.Sync()
	}
	if l.output != os.Stdout && l.output != os.Stderr {
		return l.output.Close()
	}
	return nil
}

// log 记录日志
func (l *Logger) log(entry LogEntry) {
	fields := []zap.Field{}
	if entry.Event != "" {
		fields = append(fields, zap.String("event", entry.Event))
	}
	if entry.Process != "" {
		fields = append(fields, zap.String("process", entry.Process))
	}
	if entry.Duration > 0 {
		fields = append(fields, zap.Int64("duration", entry.Duration))
	}

	switch entry.Level {
	case LevelWarn:
		l.zap.Warn(entry.Message, fields...)
	case LevelError:
		l.zap.Error(entry.Message, fields...)
	case LevelDebug:
		l.zap.Debug(entry.Message, fields...)
	default:
		l.zap.Info(entry.Message, fields...)
	}
}

// Infof 记录信息日志
func (l *Logger) Infof(format string, args ...any) {
	l.log(LogEntry{
		Level:   LevelInfo,
		Message: fmt.Sprintf(format, args...),
	})
}

// Warnf 记录警告日志
func (l *Logger) Warnf(format string, args ...any) {
	l.log(LogEntry{
		Level:   LevelWarn,
		Message: fmt.Sprintf(format, args...),
	})
}

// Errorf 记录错误日志
func (l *Logger) Errorf(format string, args ...any) {
	l.log(LogEntry{
		Level:   LevelError,
		Message: fmt.Sprintf(format, args...),
	})
}

// Debugf 记录调试日志
func (l *Logger) Debugf(format string, args ...any) {
	l.log(LogEntry{
		Level:   LevelDebug,
		Message: fmt.Sprintf(format, args...),
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
