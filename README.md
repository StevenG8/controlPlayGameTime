# game-control

Windows 下的游戏时长控制工具。
支持进程监控、超限自动终止、阈值弹窗提醒、后台运行与开机自启动。

## 功能概览

- 监控配置中的游戏进程（扫描间隔 5 秒）
- 检测到游戏进程时累计游戏时长
- 达到阈值后弹窗提醒（首次/最后警告）
- 超出每日时长后弹窗并尝试终止游戏进程
- 每日按 `resetTime` 自动重置配额
- 单实例保护，避免重复启动

## 构建

直接构建可执行文件：

```bash
go build -o game-control.exe ./cmd/game-control
```

构建 Windows 分发目录（包含脚本与示例配置）：

```bash
./build-windows.sh
```

输出目录：`dist/windows-amd64/`

## 命令

```bash
game-control <command> [config]
```

- `start [config]`：启动控制器
- `status [config]`：查看当前状态
- `validate [config]`：校验配置
- `help`：查看帮助

说明：

- `config` 可选，默认 `config.yaml`
- 若配置文件不存在，会使用内置默认配置启动

## 配置项

示例见 `config.yaml.tmpl`。

- `dailyLimit`：每日游戏时长上限（分钟）
- `resetTime`：每日重置时间，格式 `HH:MM`
- `games`：要监控的进程名列表（含 `.exe`）
- `firstThreshold`：首次提醒阈值（分钟）
- `finalThreshold`：最后提醒阈值（分钟，必须小于等于 `firstThreshold`）
- `stateFile`：状态文件路径
- `logFile`：日志文件路径

## 后台运行

PowerShell 示例：

```powershell
Start-Process -FilePath ".\game-control.exe" -ArgumentList 'start','config.yaml' -WindowStyle Hidden
```

检查是否已运行：

```powershell
Get-Process game-control -ErrorAction SilentlyContinue
```

## 自启动

从分发目录运行脚本（推荐，开箱即用）：

```powershell
.\add-autostart.bat
.\remove-autostart.bat
```

说明：

- 自启动任务名为 `GameControlAutostart`
- 任务会调用 `start-background.bat` 后台启动 `game-control.exe start config.yaml`
- `scripts/windows/*.bat` 默认按“脚本同目录”查找 `game-control.exe` 和 `config.yaml`，因此更适合由 `build-windows.sh` 复制到分发目录后使用

## 运行行为

- 告警通过弹窗发送，不仅写日志
- 每个警告阈值每天最多弹窗一次（每日重置后恢复）
- 超限通知每天最多弹窗一次（每日重置后恢复）
- 状态默认每 1 分钟保存一次，并在退出时再次保存

## 注意事项

- 仅支持 Windows
- 终止进程通常需要管理员权限
- 若出现“拒绝访问/无权限结束进程”，请用管理员终端重新执行 `add-autostart.bat` 以创建 `RL HIGHEST` 的计划任务
