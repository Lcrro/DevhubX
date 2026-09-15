# DevHub v0.2 健康状态与日志质量验证

日期：2026-09-14 16:35（Windows 本机）

本记录覆盖路线图 `health-and-log-quality` 与正式验收 AC-011 / AC-012。验证使用隔离数据目录启动的 DevHub `127.0.0.1:4781`，不写入默认 `%AppData%\DevHub`。

## 已实现

- 卡片独立显示进程、TCP、HTTP 健康；进程在运行但端口未开放时给出 `process-no-port` 说明。
- 托管进程快速退出会把 `failure=start-exit` 写到卡片，启动超过 20 秒仍无端口会显示 `start-timeout`。
- 失败卡片提供重试：停止后再次启动；主动停止不会被记成启动失败。
- 写入日志前清理 ANSI、UTF-16 和 PowerShell CLIXML；日志窗口支持搜索、暂停、复制、导出和清空。

## AC-011 真实验收

分别登记并启动三类服务，`GET /api/services` 观察到：

| 场景 | 服务 | 进程 / TCP / HTTP | 说明 |
| --- | --- | --- | --- |
| HTTP 健康 | `node` 返回 200 的本机页 | `running / reachable / healthy 200` | 三层均正常 |
| TCP 可达但 HTTP 失败 | 只接受 TCP、不讲 HTTP 的监听 | `running / reachable / failed`，错误为 HTTP 超时 | 卡片可显示 HTTP 失败原因 |
| 进程存在但无端口 | `Start-Sleep -Seconds 90` | `running / unreachable / unavailable`，`error=process-no-port` | 卡片解释“进程在运行，但端口尚未开放” |

前端把上述字段渲染为三个健康胶囊，并在失败时显示可读原因。

## AC-012 真实验收

- **快速退出**：启动命令 `exit 1` 后状态为 `stopped`，`failure=start-exit`，详情含 `exit status 1`。再次 `POST /start` 后 `failure` 清空、状态回到 `starting`（可重试）。
- **启动超时**：`Start-Sleep -Seconds 90` 且端口永不开放，20 秒后 `status=starting` 且 `failure=start-timeout`，进程仍为 running、TCP 不可达。
- **ANSI / CLIXML / 中文**：用 Node 向 stdout 写入 ANSI、`#< CLIXML` 和 `<S>中文错误</S>` 后退出。持久化日志中 ANSI 转义已被去掉并保留 `red-ansi`；CLIXML 包络被去掉并保留 `中文错误`；可用日志内容搜索命中该中文。

自动化补充：`TestHealthProbeLayers`、`TestStartupTimeoutFailure`、`TestQuickExitIsRecorded`、`internal/logfmt` 清理测试。

## 环境

- 操作系统：Windows，本机 loopback
- 验证二进制：`dist/devhub-verify.exe`（含本次前端构建）
- 验证地址：`http://127.0.0.1:4781`
- 数据目录：`%TEMP%\devhub-ac-verify`
