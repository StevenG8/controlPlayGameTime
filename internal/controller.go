package internal

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/game-control/pkg/config"
	"github.com/yourusername/game-control/pkg/logger"
	"github.com/yourusername/game-control/pkg/notifier"
	"github.com/yourusername/game-control/pkg/process"
	"github.com/yourusername/game-control/pkg/quota"
)

type processScanner interface {
	FindGameProcesses(gameNames []string) ([]process.ProcessInfo, error)
	TerminateWithRetry(pid int, maxRetries int, retryDelay time.Duration) error
}

// Controller 主控制器
type Controller struct {
	config       *config.Config
	quotaState   *quota.QuotaState
	scanner      processScanner
	notifier     notifier.Notifier
	lastSaveTime time.Time
}

// NewController 创建新的控制器
func NewController(cfg *config.Config, qState *quota.QuotaState) *Controller {
	return NewControllerWithDeps(cfg, qState, process.NewScanner(), notifier.NewNotifier())
}

// NewControllerWithDeps 创建可注入依赖的控制器（用于测试）
func NewControllerWithDeps(
	cfg *config.Config,
	qState *quota.QuotaState,
	scanner processScanner,
	n notifier.Notifier,
) *Controller {
	if scanner == nil {
		scanner = process.NewScanner()
	}
	if n == nil {
		n = notifier.NewNotifier()
	}
	return &Controller{
		config:       cfg,
		quotaState:   qState,
		scanner:      scanner,
		notifier:     n,
		lastSaveTime: time.Now(),
	}
}

// Run 运行主控制循环
func (c *Controller) Run() error {
	logger.Infof("游戏时间控制守护进程启动")
	logger.Infof("每日时间限制: %d 分钟", c.config.DailyLimit)
	logger.Infof("游戏进程列表: %v", c.config.Games)

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
			logger.Infof("接收到信号 %v，正在关闭...", sig)
			c.cleanup()
			return nil
		}
	}
}

// tick 每次循环执行的任务
func (c *Controller) tick() {
	// 1. 检查是否需要重置
	shouldReset, err := c.quotaState.ShouldReset()
	if err != nil {
		logger.Errorf("检查重置状态失败: %v", err)
		return
	}

	if shouldReset {
		if err := c.quotaState.Reset(); err != nil {
			logger.Errorf("重置配额失败: %v", err)
		} else {
			logger.LogQuotaReset()
		}
	}

	// 2. 扫描游戏进程
	gameProcesses, err := c.scanner.FindGameProcesses(c.config.Games)
	if err != nil {
		logger.Errorf("扫描游戏进程失败: %v", err)
		return
	}

	// 3. 简化：只要检测到有游戏进程就累加扫描间隔时间
	if len(gameProcesses) > 0 {
		// 扫描间隔是5秒
		c.quotaState.AddTime(5)
		logger.Debugf("检测到 %d 个游戏进程，累加5秒时间", len(gameProcesses))
	}

	// 4. 检查时间限制
	if c.quotaState.IsLimitExceeded() {
		logger.LogLimitExceeded()
		if c.quotaState.ConsumeLimitNotification() {
			if err := c.notifier.NotifyLimitExceeded(); err != nil {
				logger.Errorf("超限弹窗失败: %v", err)
			}
		}

		// 终止所有游戏进程
		for _, proc := range gameProcesses {
			if err := c.scanner.TerminateWithRetry(proc.PID, 3, 1*time.Second); err != nil {
				logger.Errorf("终止进程失败 (PID: %d): %v", proc.PID, err)
			}
		}
	} else {
		// 检查警告阈值
		first, final := c.quotaState.ConsumeWarningNotifications()

		if final {
			remaining := c.quotaState.GetRemainingMinutes()
			logger.Warnf("最后警告：剩余游戏时间仅剩 %d 分钟！", remaining)
			if err := c.notifier.NotifyFinalWarning(remaining); err != nil {
				logger.Errorf("最后警告弹窗失败: %v", err)
			}
		} else if first {
			remaining := c.quotaState.GetRemainingMinutes()
			logger.Warnf("警告：剩余游戏时间不足 %d 分钟（剩余 %d 分钟）",
				c.config.FirstThreshold, remaining)
			if err := c.notifier.NotifyFirstWarning(remaining); err != nil {
				logger.Errorf("首次警告弹窗失败: %v", err)
			}
		}
	}

	// 5. 定期保存状态
	if time.Since(c.lastSaveTime) >= 1*time.Minute {
		if err := c.quotaState.SaveToFile(); err != nil {
			logger.Errorf("保存状态失败: %v", err)
		} else {
			c.lastSaveTime = time.Now()
		}
	}
}

// cleanup 清理资源
func (c *Controller) cleanup() {
	logger.Infof("正在保存状态...")

	// 保存状态
	if err := c.quotaState.SaveToFile(); err != nil {
		logger.Errorf("保存状态失败: %v", err)
	}

	logger.Infof("游戏时间控制守护进程已关闭")
	_ = logger.Close()
}

// GetStatus 获取当前状态
func (c *Controller) GetStatus() StatusInfo {
	// 扫描当前游戏进程
	gameProcesses, err := c.scanner.FindGameProcesses(c.config.Games)
	activeProcessCount := 0
	if err == nil {
		activeProcessCount = len(gameProcesses)
	}

	remaining := c.quotaState.GetRemainingMinutes()
	nextReset := c.quotaState.TimeUntilNextReset()

	return StatusInfo{
		AccumulatedTime:    c.quotaState.GetAccumulatedMinutes(),
		RemainingTime:      remaining,
		DailyLimit:         c.config.DailyLimit,
		ActiveProcessCount: activeProcessCount,
		NextResetTime:      nextReset,
	}
}

// StatusInfo 状态信息
type StatusInfo struct {
	AccumulatedTime    int           `json:"accumulatedTime"`    // 累计时间（分钟）
	RemainingTime      int           `json:"remainingTime"`      // 剩余时间（分钟）
	DailyLimit         int           `json:"dailyLimit"`         // 每日限制（分钟）
	ActiveProcessCount int           `json:"activeProcessCount"` // 活跃进程数
	NextResetTime      time.Duration `json:"nextResetTime"`      // 距离下次重置的时间
}
