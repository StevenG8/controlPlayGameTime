package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/game-control/internal"
	"github.com/yourusername/game-control/pkg/autostart"
	"github.com/yourusername/game-control/pkg/config"
	"github.com/yourusername/game-control/pkg/logger"
	"github.com/yourusername/game-control/pkg/quota"
	"github.com/yourusername/game-control/pkg/singleinstance"
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
	case "install-autostart":
		if err := runInstallAutostart(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "remove-autostart":
		if err := runRemoveAutostart(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := runStatus(); err != nil {
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
	configPath, background, err := parseStartArgs(os.Args[2:])
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	guard, err := singleinstance.Acquire("game-control-main")
	if err != nil {
		if errors.Is(err, singleinstance.ErrAlreadyRunning) {
			return fmt.Errorf("控制器已在运行")
		}
		return fmt.Errorf("获取单实例锁失败: %w", err)
	}
	defer guard.Release()

	log, err := logger.NewLogger(cfg.LogFile)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}
	defer log.Close()

	if background {
		log.Info("以后台模式启动")
	}

	var qState *quota.QuotaState
	loadedState, err := quota.LoadFromFile(cfg)
	if err != nil || loadedState == nil {
		qState, err = quota.NewQuotaState(cfg)
		if err != nil {
			return fmt.Errorf("创建配额状态失败: %w", err)
		}
	} else {
		qState = loadedState
		if err := qState.Validate(); err != nil {
			log.Warn(fmt.Sprintf("状态验证失败，创建新状态: %v", err))
			qState, err = quota.NewQuotaState(cfg)
			if err != nil {
				return fmt.Errorf("创建配额状态失败: %w", err)
			}
		}
	}

	controller := internal.NewController(cfg, qState, log)
	return controller.Run()
}

func parseStartArgs(args []string) (string, bool, error) {
	configPath := "config.yaml"
	background := false
	configSet := false

	for _, arg := range args {
		switch arg {
		case "--background", "-b":
			background = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("未知参数: %s", arg)
			}
			if configSet {
				return "", false, fmt.Errorf("仅允许指定一个配置文件路径")
			}
			configPath = arg
			configSet = true
		}
	}

	return configPath, background, nil
}

func runInstallAutostart() error {
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("解析可执行文件路径失败: %w", err)
	}

	if err := autostart.InstallTask(exePath, configPath); err != nil {
		return err
	}

	fmt.Println("已安装开机自启动任务")
	return nil
}

func runRemoveAutostart() error {
	if err := autostart.RemoveTask(); err != nil {
		return err
	}
	fmt.Println("已移除开机自启动任务")
	return nil
}

func runStatus() error {
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	qState, err := quota.LoadFromFile(cfg)
	if err != nil {
		return fmt.Errorf("加载状态失败: %w", err)
	}
	if qState == nil {
		return fmt.Errorf("没有找到状态文件，请先运行 start 命令")
	}

	log, _ := logger.NewLogger("")
	controller := internal.NewController(cfg, qState, log)

	shouldReset, err := qState.ShouldReset()
	if err != nil {
		return fmt.Errorf("检查重置状态失败: %v", err)
	}

	if shouldReset {
		if err := qState.Reset(); err != nil {
			return fmt.Errorf("重置配额失败: %v", err)
		}
		log.LogQuotaReset()
		if err := qState.SaveToFile(); err != nil {
			return fmt.Errorf("保存重置状态失败: %v", err)
		}
	}

	status := controller.GetStatus()

	fmt.Println("=== 游戏时间控制状态 ===")
	fmt.Printf("累计游戏时间: %d 分钟\n", status.AccumulatedTime)
	fmt.Printf("剩余游戏时间: %d 分钟\n", status.RemainingTime)
	fmt.Printf("每日时间限制: %d 分钟\n", status.DailyLimit)

	if status.ActiveProcessCount > 0 {
		fmt.Printf("\n活跃游戏进程: %d 个\n", status.ActiveProcessCount)
		fmt.Println("  (进程详情需要实时扫描，此处只显示数量)")
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

func runValidate() error {
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	fmt.Println("配置文件验证通过")
	fmt.Printf("每日时间限制: %d 分钟\n", cfg.DailyLimit)
	fmt.Printf("重置时间: %s\n", cfg.ResetTime)
	fmt.Printf("游戏进程列表: %v\n", cfg.Games)
	fmt.Printf("警告阈值: %d 分钟 (第一次), %d 分钟 (最后)\n",
		cfg.FirstThreshold, cfg.FinalThreshold)

	return nil
}

func printHelp() {
	fmt.Println("游戏时间控制工具")
	fmt.Println()
	fmt.Println("使用方法:")
	fmt.Println("  game-control <command> [参数]")
	fmt.Println()
	fmt.Println("可用命令:")
	fmt.Println("  start [config] [--background]     启动游戏时间控制守护进程")
	fmt.Println("  install-autostart [config]        安装 Windows 开机自启动任务")
	fmt.Println("  remove-autostart                  移除 Windows 开机自启动任务")
	fmt.Println("  status [config]                   查询当前游戏时间状态")
	fmt.Println("  validate [config]                 验证配置文件")
	fmt.Println("  help                              显示此帮助信息")
	fmt.Println()
	fmt.Println("参数说明:")
	fmt.Println("  --background, -b  以后台模式运行（仅 start 命令支持）")
	fmt.Println()
	fmt.Println("说明:")
	fmt.Println("  - 默认配置文件路径: config.yaml")
	fmt.Println("  - 需要管理员权限来终止游戏进程")
	fmt.Println("  - 仅支持 Windows 系统")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  game-control start")
	fmt.Println("  game-control start config.yaml --background")
	fmt.Println("  game-control install-autostart config.yaml")
	fmt.Println("  game-control remove-autostart")
}
