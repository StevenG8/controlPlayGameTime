# Repository Guidelines

## 项目结构与模块组织
- `cmd/game-control/`：CLI 入口（`main.go`），包含 `start`、`status`、`validate`、`help`。
- `internal/`：控制器编排逻辑（监控循环、配额检查、通知触发）。
- `pkg/`：可复用模块：
  - `config/`：配置加载与校验
  - `process/`：进程扫描与终止
  - `quota/`：配额状态与持久化
  - `logger/`、`notifier/`、`singleinstance/`
- `scripts/windows/`：后台运行与开机自启动脚本。
- `dist/windows-amd64/`：Windows 分发产物目录。
- `openspec/`：规格文档与变更归档。

## 构建、测试与开发命令
- `go build -o game-control.exe ./cmd/game-control`  
  构建主程序。
- `./build-windows.sh`  
  生成分发目录（输出到 `dist/windows-amd64/`）。
- `go test ./...`  
  运行全部单元测试。
- `GOCACHE=/tmp/go-build go test ./...`  
  当默认缓存目录权限受限时使用。
- `go test ./internal ./pkg/quota -run TestController`  
  迭代时执行定向测试。

## 代码风格与命名规范
- 语言为 Go；提交前执行 `gofmt`（如 `gofmt -w ./...`）。
- 复用能力放 `pkg/`，流程编排放 `internal/`，保持包职责单一。
- 导出标识符使用 `PascalCase`，非导出使用 `camelCase`。
- 测试文件命名为 `*_test.go`，测试函数命名为 `TestXxxBehavior`。
- 错误包装统一使用 `fmt.Errorf("上下文: %w", err)`，不要在 `Sprintf` 中使用 `%w`。

## 测试指南
- 测试框架：Go 原生 `testing`。
- 行为变更需同步补充同模块测试（`internal/` 或 `pkg/*`）。
- 重点覆盖边界：非法配置、陈旧状态文件、告警去重、进程扫描/终止失败。
- 文件相关测试使用 `t.TempDir()`，保证可重复与隔离。

## 提交与 Pull Request 规范
- 当前历史以简短祈使句为主（中英混用），如 `edit readme`、`remove useless cli`。
- 建议提交格式：`<scope>: <摘要>`，例如 `quota: reset warning flags on rollover`。
- PR 至少应包含：
  - 修改内容与原因
  - 测试证据（`go test ./...` 结果摘要）
  - 配置/脚本影响（如涉及 `config.yaml.tmpl`、`scripts/windows/`）
  - 关联的 spec 或 issue（如 `openspec/changes/...`）

## 安全与配置提示
- 项目仅支持 Windows；终止进程通常需要管理员权限。
- 非发布目的不要提交个人运行产物（如 `state.json`、本地日志、临时 `dist/` 内容）。
