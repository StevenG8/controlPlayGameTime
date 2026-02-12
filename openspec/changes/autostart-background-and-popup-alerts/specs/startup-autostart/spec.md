## ADDED Requirements

### Requirement: 开机后自动后台启动控制器
系统 MUST 提供在 Windows 用户登录后自动启动游戏时间控制器的能力，并以后台模式运行。

#### Scenario: 安装自启动任务成功
- **WHEN** 用户执行安装自启动的命令
- **THEN** 系统 SHALL 在任务计划程序中创建指向 `game-control start --background` 的登录触发任务

#### Scenario: 自启动后进入后台运行
- **WHEN** 用户登录系统且自启动任务被触发
- **THEN** 控制器 SHALL 在无交互终端依赖的后台模式运行并持续执行时间控制循环

### Requirement: 控制器单实例运行
系统 MUST 保证同一用户会话下仅有一个控制器实例运行。

#### Scenario: 自启动与手动启动并发
- **WHEN** 自启动实例已经在运行，用户再次执行 `game-control start`
- **THEN** 系统 SHALL 拒绝启动第二个实例并返回清晰提示

### Requirement: 自启动可卸载
系统 MUST 支持移除已安装的自启动任务。

#### Scenario: 卸载自启动任务成功
- **WHEN** 用户执行卸载自启动命令
- **THEN** 系统 SHALL 删除对应任务计划项，后续登录不再自动启动控制器
