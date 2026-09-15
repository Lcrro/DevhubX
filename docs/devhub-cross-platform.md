# 跨平台运行记录

日期：2026-09-14

AC-016 要求 Windows、macOS、Linux 都有真实发现、启停、日志和截图记录。本文件只记录已经取得的平台证据，不把交叉编译当成运行验收。

## Windows

此前 v0.1/v0.2 验收均在本机 Windows 完成，包括发现、启停、日志、截图、健康分层和发现规则。本次回归构建 `dist/devhub-verify.exe` 后配置导入导出与个性化设置仍然可用。

## Linux（WSL2 Ubuntu）

在 WSL Ubuntu（kernel 6.6.87.2-microsoft-standard-WSL2）运行交叉编译的 `devhub-linux-amd64`：

- `GET /api/session` 的 `platform=linux`。
- 从带 `package.json` 的目录启动 `python3 -m http.server 8766`，扫描得到 `wsl-demo` / `react` / `python3` / `external`。
- 托管 `sleep 8`：启动 PID 813，日志出现 `[DevHub] 已启动 PID 813`，停止后状态 `stopped`。
- 未安装 Chrome/Edge，本次 **没有** 做 Linux 截图。

WSL、容器和权限不足的进程仍可能扫不到；这些路径需要手动添加。Docker 引擎本次未运行，没有容器内证据。

## macOS

当前开发机没有 macOS。没有真实发现、进程组或截图记录。GitHub Actions 的 `macos-latest` 只跑测试和交叉构建，不计入运行验收。
