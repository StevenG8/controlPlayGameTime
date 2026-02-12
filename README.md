# game-control

Windows 下的游戏时长控制工具。  
支持进程监控、超限自动终止、阈值弹窗提醒，以及开机后自动后台运行。

## 构建

```bash
go build -o game-control.exe ./cmd/game-control
```

## 命令

```bash
game-control <command> [参数]
```

- `start [config] [--background]`: 启动控制器
- `install-autostart [config]`: 安装 Windows 开机自启动任务（登录后触发）
- `remove-autostart`: 移除开机自启动任务
- `status [config]`: 查看当前状态
- `validate [config]`: 校验配置
- `help`: 查看帮助

### `start` 参数

- `--background` 或 `-b`: 后台模式运行（仅 `start` 支持）
- `config`: 可选配置文件路径，默认 `config.yaml`

## 示例

```bash
game-control start
game-control start config.yaml --background
game-control install-autostart config.yaml
game-control remove-autostart
game-control status
game-control validate
```

## 行为说明

- 告警为弹窗通知，不只写日志。
- 每个警告时间点每天仅弹窗一次（每日重置后恢复触发）。
- 超限后会尝试终止游戏进程，并给出超限弹窗。
- 使用单实例保护，重复启动会被拒绝。

## 注意事项

- 仅支持 Windows。
- 终止进程通常需要管理员权限。
