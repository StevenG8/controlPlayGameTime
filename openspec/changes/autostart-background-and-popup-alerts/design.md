## Context

当前系统以命令行方式运行，`start` 命令进入循环并通过日志输出告警。现状存在两个问题：
- 程序不会在 Windows 启动后自动运行，容易因忘记手动启动导致当天无控制能力。
- 警告信息仅写日志，用户在不看终端时无法及时感知。

基于 proposal，本次设计目标是：
- 在 Windows 上通过分发脚本安装/卸载自启动任务。
- 在到达警告时间点时弹窗提醒用户。
- 每个警告时间点只弹窗一次，避免每 5 秒轮询时重复打扰。
- 不新增告警配置定制能力，沿用现有阈值含义。

## Goals / Non-Goals

**Goals:**
- 在 `dist` 目录提供可直接执行的 `add-autostart.bat` / `remove-autostart.bat`。
- 脚本通过 Windows Task Scheduler 注册登录触发任务，后台启动 `game-control.exe start config.yaml`。
- 引入桌面弹窗通知通道，用于首个警告点、最终警告点和超限提示。
- 建立“单日单阈值一次性弹窗”机制，并在每日重置后自动恢复可提醒状态。
- 保留现有日志审计行为，弹窗与日志并行。

**Non-Goals:**
- 不在 CLI 中提供自启动安装/卸载命令。
- 不引入新的告警配置字段（如可调频率、可调文案、可关闭某一级告警）。
- 不实现托盘 GUI、通知中心历史、交互式确认等 UI 功能。
- 不在本次变更中扩展 Linux/macOS 自启动方案。

## Decisions

### 1. 自启动管理通过 `dist` 脚本实现

**Decision**
- 提供 `add-autostart.bat` 与 `remove-autostart.bat`，由脚本调用 `schtasks` 创建/删除登录触发任务。
- 任务动作使用 `cmd /c start "" /b ...` 后台启动控制器。

**Rationale**
- 交付更简单，用户无需记忆 CLI 子命令。
- 脚本可与分发包一起发布，降低版本迁移和操作门槛。

**Alternatives Considered**
- CLI 子命令（`install-autostart` / `remove-autostart`）：功能完整，但增加程序入口复杂度。
- 启动目录快捷方式：实现简单，但可观测性与可维护性较弱。

### 2. 控制器启动命令保持精简，仅保留 `start`

**Decision**
- 保留 `start`、`status`、`validate` 等核心命令。
- 不再在程序中维护自启动安装/卸载命令。

**Rationale**
- 与“自启动由脚本负责”的职责边界一致。
- 降低 CLI 维护成本。

### 3. 弹窗通知抽象为 `Notifier` 接口，默认 Windows 实现

**Decision**
- 通知抽象层保留以下能力：
  - `NotifyFirstWarning(remainingMinutes int)`
  - `NotifyFinalWarning(remainingMinutes int)`
  - `NotifyLimitExceeded()`
- Windows 默认实现使用系统命令（PowerShell/原生命令）触发桌面弹窗；失败时记录错误日志。

**Rationale**
- 将通知与控制逻辑解耦，便于测试与后续替换实现。
- 保持依赖最小化，不强制引入重量级 GUI 库。

### 4. 阈值弹窗去重采用“当日阈值标记”并持久化到状态文件

**Decision**
- 在 `QuotaState` 增加当日弹窗状态（如 `FirstWarned`, `FinalWarned`, `LimitWarned` 或等价位图）。
- 触发规则：
  - 首警告条件满足且 `FirstWarned=false` 时弹窗并置位。
  - 末警告条件满足且 `FinalWarned=false` 时弹窗并置位。
  - 超限条件满足且 `LimitWarned=false` 时弹窗并置位。
- 每日重置时清空上述标记。

**Rationale**
- 直接解决“每到警告时间只弹一次”的需求。
- 标记持久化可覆盖进程重启场景，避免重启后重复提醒同一阈值。

## Risks / Trade-offs

- [脚本在无管理员权限或策略受限环境下失败] -> 输出明确错误并保留手工排障路径。
- [任务计划程序创建失败（权限/策略限制）] -> 输出明确错误与修复建议，保留手动 `start` 路径。
- [单实例锁实现不当导致“假死实例”] -> 使用带进程信息的锁文件并在启动时清理陈旧锁。
- [状态文件结构变更引发兼容问题] -> 新增字段采用向后兼容默认值，读取旧状态时自动补齐。
- [弹窗可能打断用户体验] -> 严格执行“每阈值每日一次”，避免轮询重复提醒。

## Migration Plan

1. 调整分发与文档：
- 在分发目录提供 `add-autostart.bat` / `remove-autostart.bat`。
- 更新 README 与构建脚本，确保每次构建都会打包脚本。

2. 精简 CLI：
- 移除程序内自启动安装/卸载命令与对应帮助文案。

3. 保留通知与去重能力：
- 通知模块继续接入阈值判断分支并保留日志。
- 每日重置时清空阈值去重状态。

4. 发布与回滚：
- 发布后用户执行一次 `add-autostart.bat`。
- 若异常，执行 `remove-autostart.bat` 并回退到手动 `start`。
- 回滚不影响既有核心限时逻辑。

## Open Questions

- 弹窗标题与正文是否需要统一文案规范（当前可先使用固定中文文案）？
- 超限弹窗是否也必须“每日仅一次”（设计默认是）？
