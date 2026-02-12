# game-control

Windows 下的游戏时长控制工具。  
支持进程监控、超限自动终止、阈值弹窗提醒，以及后台运行。

## 构建

```bash
go build -o game-control.exe ./cmd/game-control
```

## 命令

```bash
game-control <command> [参数]
```

- `start [config]`: 启动控制器
- `status [config]`: 查看当前状态
- `validate [config]`: 校验配置
- `help`: 查看帮助

### `start` 参数

- `config`: 可选配置文件路径，默认 `config.yaml`

## 后台运行（PowerShell）

后台运行由用户在启动命令中决定，推荐使用 PowerShell：

```powershell
Start-Process -FilePath ".\game-control.exe" -ArgumentList 'start','config.yaml' -WindowStyle Hidden
```

检查是否已运行：

```powershell
Get-Process game-control -ErrorAction SilentlyContinue
```

## 自启动

通过分发目录中的脚本管理（Windows Task Scheduler）：

```powershell
.\add-autostart.bat
```

移除自启动任务：

```powershell
.\remove-autostart.bat
```

说明：默认自启动任务会调用分发目录中的 `start-background.bat`，该脚本内部使用绝对路径后台启动 `game-control.exe start config.yaml`。

## 示例

```bash
game-control start
add-autostart.bat
remove-autostart.bat
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
