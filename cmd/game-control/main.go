package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/game-control/internal"
	"github.com/yourusername/game-control/pkg/config"
	"github.com/yourusername/game-control/pkg/logger"
	"github.com/yourusername/game-control/pkg/process"
	"github.com/yourusername/game-control/pkg/quota"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "start":
		if err := runStart(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := runStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "reset":
		if err := runReset(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := runValidate(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Printf("未知命令: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func runStart() error {
	// 检查管理员权限
	if !process.IsAdmin() {
		return fmt.Errorf("需要管理员权限来终止游戏进程，请以管理员身份运行")
	}

	// 获取配置文件路径
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	// 加载配置
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建日志记录器
	log, err := logger.NewLogger(cfg.LogFile)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}
	defer log.Close()

	// 加载或创建配额状态
	var qState *quota.QuotaState
	loadedState, err := quota.LoadFromFile(cfg.StateFile)
	if err != nil || loadedState == nil {
		qState, err = quota.NewQuotaState(cfg.ResetTime)
		if err != nil {
			return fmt.Errorf("创建配额状态失败: %w", err)
		}
	} else {
		qState = loadedState
		// 验证状态
		if err := qState.Validate(); err != nil {
			log.Warn(fmt.Sprintf("状态验证失败，创建新状态: %v", err))
			qState, err = quota.NewQuotaState(cfg.ResetTime)
			if err != nil {
				return fmt.Errorf("创建配额状态失败: %w", err)
			}
		}
	}

	// 创建控制器
	controller := internal.NewController(cfg, qState, log)

	// 运行主控制循环
	return controller.Run()
}

func runStatus() error {
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	// 加载配置
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 加载配额状态
	qState, err := quota.LoadFromFile(cfg.StateFile)
	if err != nil {
		return fmt.Errorf("加载状态失败: %w", err)
	}

	if qState == nil {
		return fmt.Errorf("没有找到状态文件，请先运行 start 命令")
	}

	// 创建控制器
	log, _ := logger.NewLogger("") // 使用标准输出
	controller := internal.NewController(cfg, qState, log)

	// 获取状态
	status := controller.GetStatus()

	// 显示状态
	fmt.Println("=== 游戏时间控制状态 ===")
	fmt.Printf("累计游戏时间: %d 分钟\n", status.AccumulatedTime)
	fmt.Printf("剩余游戏时间: %d 分钟\n", status.RemainingTime)
	fmt.Printf("每日时间限制: %d 分钟\n", status.DailyLimit)

	if status.ActiveProcessCount > 0 {
		fmt.Printf("\n活跃游戏进程 (%d):\n", status.ActiveProcessCount)
		for _, proc := range status.ActiveProcesses {
			duration := process.FormatDurationShort(time.Since(proc.StartTime).Milliseconds())
			fmt.Printf("  - %s (PID: %d, 运行时长: %s)\n", proc.Name, proc.PID, duration)
		}
	} else {
		fmt.Println("\n当前没有活跃的游戏进程")
	}

	nextReset := status.NextResetTime
	hours := int(nextReset.Hours())
	minutes := int(nextReset.Minutes()) % 60
	fmt.Printf("\n距离下次重置: %d 小时 %d 分钟\n", hours, minutes)

	log.Close()
	return nil
}

func runReset() error {
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	// 加载配置
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 加载配额状态
	qState, err := quota.LoadFromFile(cfg.StateFile)
	if err != nil {
		return fmt.Errorf("加载状态失败: %w", err)
	}

	if qState == nil {
		return fmt.Errorf("没有找到状态文件，请先运行 start 命令")
	}

	// 重置配额
	if err := qState.Reset(cfg.ResetTime); err != nil {
		return fmt.Errorf("重置配额失败: %w", err)
	}

	// 保存状态
	if err := qState.SaveToFile(cfg.StateFile); err != nil {
		return fmt.Errorf("保存状态失败: %w", err)
	}

	fmt.Println("每日游戏时间配额已重置")
	return nil
}

func runValidate() error {
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	// 加载配置
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	fmt.Println("配置文件验证通过")
	fmt.Printf("每日时间限制: %d 分钟\n", cfg.TimeLimit.DailyLimit)
	fmt.Printf("重置时间: %s\n", cfg.ResetTime)
	fmt.Printf("游戏进程列表: %v\n", cfg.Games)
	fmt.Printf("警告阈值: %d 分钟 (第一次), %d 分钟 (最后)\n",
		cfg.Warning.FirstThreshold, cfg.Warning.FinalThreshold)

	return nil
}

func printHelp() {
	fmt.Println("游戏时间控制工具")
	fmt.Println()
	fmt.Println("使用方法:")
	fmt.Println("  game-control <command> [options]")
	fmt.Println()
	fmt.Println("可用命令:")
	fmt.Println("  start [config]  启动游戏时间控制守护进程")
	fmt.Println("  status [config] 查询当前游戏时间状态")
	fmt.Println("  reset [config]  手动重置每日游戏时间配额")
	fmt.Println("  validate [config] 验证配置文件")
	fmt.Println("  help           显示此帮助信息")
	fmt.Println()
	fmt.Println("说明:")
	fmt.Println("  - 默认配置文件路径: config.yaml")
	fmt.Println("  - 需要管理员权限来终止游戏进程")
	fmt.Println("  - 仅支持 Windows 系统")
}
