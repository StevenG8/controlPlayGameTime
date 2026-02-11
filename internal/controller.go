package internal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/game-control/pkg/config"
	"github.com/yourusername/game-control/pkg/logger"
	"github.com/yourusername/game-control/pkg/process"
	"github.com/yourusername/game-control/pkg/quota"
)

// Controller 主控制器
type Controller struct {
	config        *config.Config
	quotaState    *quota.QuotaState
	scanner       *process.Scanner
	tracker       *process.ProcessTracker
	logger        *logger.Logger
	stateFilePath string
	lastSaveTime  time.Time
}

// NewController 创建新的控制器
func NewController(cfg *config.Config, qState *quota.QuotaState, log *logger.Logger) *Controller {
	return &Controller{
		config:        cfg,
		quotaState:    qState,
		scanner:       process.NewScanner(),
		tracker:       process.NewProcessTracker(),
		logger:        log,
		stateFilePath: cfg.StateFile,
		lastSaveTime:  time.Now(),
	}
}

// Run 运行主控制循环
func (c *Controller) Run() error {
	c.logger.Info("游戏时间控制守护进程启动")
	c.logger.Info(fmt.Sprintf("每日时间限制: %d 分钟", c.config.TimeLimit.DailyLimit))
	c.logger.Info(fmt.Sprintf("游戏进程列表: %v", c.config.Games))

	// 检查是否需要重置
	shouldReset, err := c.quotaState.ShouldReset(c.config.ResetTime)
	if err != nil {
		c.logger.Error(fmt.Sprintf("检查重置状态失败: %v", err))
		return err
	}

	if shouldReset {
		if err := c.quotaState.Reset(c.config.ResetTime); err != nil {
			c.logger.Error(fmt.Sprintf("重置配额失败: %v", err))
			return err
		}
		c.logger.LogQuotaReset()
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 主控制循环
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.tick()

		case sig := <-sigChan:
			c.logger.Info(fmt.Sprintf("接收到信号 %v，正在关闭...", sig))
			c.cleanup()
			return nil
		}
	}
}

// tick 每次循环执行的任务
func (c *Controller) tick() {
	// 1. 检查是否需要重置
	shouldReset, err := c.quotaState.ShouldReset(c.config.ResetTime)
	if err != nil {
		c.logger.Error(fmt.Sprintf("检查重置状态失败: %v", err))
		return
	}

	if shouldReset {
		if err := c.quotaState.Reset(c.config.ResetTime); err != nil {
			c.logger.Error(fmt.Sprintf("重置配额失败: %v", err))
		} else {
			c.logger.LogQuotaReset()
			c.tracker = process.NewProcessTracker() // 重置进程追踪器
		}
	}

	// 2. 扫描游戏进程
	gameProcesses, err := c.scanner.FindGameProcesses(c.config.Games)
	if err != nil {
		c.logger.Error(fmt.Sprintf("扫描游戏进程失败: %v", err))
		return
	}

	// 3. 更新进程状态
	newProcesses := c.scanner.GetNewProcesses(gameProcesses)
	stoppedProcesses := c.scanner.GetStoppedProcesses(gameProcesses)

	// 4. 处理新启动的游戏进程
	for _, proc := range newProcesses {
		c.tracker.StartSession(proc.PID, proc.Name)
		c.logger.LogGameStart(proc.Name)
	}

	// 5. 处理已停止的游戏进程
	for _, proc := range stoppedProcesses {
		duration, err := c.tracker.EndSession(proc.PID)
		if err != nil {
			c.logger.Error(fmt.Sprintf("结束进程会话失败 (PID: %d): %v", proc.PID, err))
			continue
		}
		c.logger.LogGameStop(proc.Name, duration)
		c.quotaState.AddTime(duration)
		c.tracker.RemoveSession(proc.PID)
	}

	// 6. 更新活跃会话的运行时间
	activeDurations := c.tracker.UpdateActiveSessionDurations()
	for pid, duration := range activeDurations {
		session, exists := c.tracker.GetSession(pid)
		if exists {
			// 计算自上次更新以来的增量时间
			// 这里简化处理，实际应该记录上次更新时间
			lastAccumulated := session.Duration
			if duration > lastAccumulated {
				increment := duration - lastAccumulated
				c.quotaState.AddTime(increment)
			}
		}
	}

	// 更新扫描器状态
	c.scanner.UpdateLastProcesses(gameProcesses)

	// 7. 检查时间限制
	if c.quotaState.IsLimitExceeded(c.config.TimeLimit.DailyLimit) {
		c.logger.LogLimitExceeded()

		// 终止所有游戏进程
		activeProcesses := c.tracker.GetActiveSessions()
		for _, session := range activeProcesses {
			if err := c.scanner.RunWithRetry(session.PID, 3, 1*time.Second); err != nil {
				c.logger.Error(fmt.Sprintf("终止进程失败 (PID: %d): %v", session.PID, err))
			}
		}
	} else {
		// 检查警告阈值
		first, final := c.quotaState.CheckWarningThresholds(
			c.config.TimeLimit.DailyLimit,
			c.config.Warning.FirstThreshold,
			c.config.Warning.FinalThreshold,
		)

		if final {
			remaining := c.quotaState.GetRemainingMinutes(c.config.TimeLimit.DailyLimit)
			c.logger.Warn(fmt.Sprintf("最后警告：剩余游戏时间仅剩 %d 分钟！", remaining))
		} else if first {
			remaining := c.quotaState.GetRemainingMinutes(c.config.TimeLimit.DailyLimit)
			c.logger.Warn(fmt.Sprintf("警告：剩余游戏时间不足 %d 分钟（剩余 %d 分钟）",
				c.config.Warning.FirstThreshold, remaining))
		}
	}

	// 8. 定期保存状态
	if time.Since(c.lastSaveTime) >= 1*time.Minute {
		if err := c.quotaState.SaveToFile(c.stateFilePath); err != nil {
			c.logger.Error(fmt.Sprintf("保存状态失败: %v", err))
		} else {
			c.lastSaveTime = time.Now()
		}
	}
}

// cleanup 清理资源
func (c *Controller) cleanup() {
	c.logger.Info("正在保存状态...")

	// 保存所有活跃会话的运行时间
	activeDurations := c.tracker.UpdateActiveSessionDurations()
	for pid, duration := range activeDurations {
		session, exists := c.tracker.GetSession(pid)
		if exists {
			lastAccumulated := session.Duration
			if duration > lastAccumulated {
				increment := duration - lastAccumulated
				c.quotaState.AddTime(increment)
			}
		}
	}

	// 保存状态
	if err := c.quotaState.SaveToFile(c.stateFilePath); err != nil {
		c.logger.Error(fmt.Sprintf("保存状态失败: %v", err))
	}

	c.logger.Info("游戏时间控制守护进程已关闭")
	c.logger.Close()
}

// GetStatus 获取当前状态
func (c *Controller) GetStatus() StatusInfo {
	activeProcesses := c.tracker.GetActiveSessions()
	remaining := c.quotaState.GetRemainingMinutes(c.config.TimeLimit.DailyLimit)
	nextReset := c.quotaState.TimeUntilNextReset()

	return StatusInfo{
		AccumulatedTime:    c.quotaState.GetAccumulatedMinutes(),
		RemainingTime:      remaining,
		DailyLimit:         c.config.TimeLimit.DailyLimit,
		ActiveProcessCount: len(activeProcesses),
		ActiveProcesses:    activeProcesses,
		NextResetTime:      nextReset,
	}
}

// StatusInfo 状态信息
type StatusInfo struct {
	AccumulatedTime    int                       `json:"accumulatedTime"`    // 累计时间（分钟）
	RemainingTime      int                       `json:"remainingTime"`      // 剩余时间（分钟）
	DailyLimit         int                       `json:"dailyLimit"`         // 每日限制（分钟）
	ActiveProcessCount int                       `json:"activeProcessCount"` // 活跃进程数
	ActiveProcesses    []*process.ProcessSession `json:"activeProcesses"`    // 活跃进程列表
	NextResetTime      time.Duration             `json:"nextResetTime"`      // 距离下次重置的时间
}
