# DevHub 项目总览

更新时间：2026-09-14

DevHub 是一个只在本机运行的开发服务控制室。它把多个项目的目录、端口、启动命令、进程状态、健康检查、实时日志和网页封面放在一个界面里，让开发者快速回答三个问题：服务属于哪个项目、现在是否真的可用、下一次如何可靠启动。

## 项目目标

### 核心目标

1. **自动发现项目服务**：扫描当前用户的本机 TCP 监听进程，向上识别 `package.json`、`go.mod`、`pyproject.toml`、`Cargo.toml` 等项目标记，并排除 QQ、编辑器和系统安装目录等常见误判来源。
2. **统一管理生命周期**：保存可信的项目目录和启动命令，从 DevHub 启动、查看输出并停止服务。启动前检查端口冲突，停止时清理整个由 DevHub 托管的进程树。
3. **提供可理解的运行反馈**：把进程身份、TCP 端口和 HTTP 可用性分开显示；启动超时、快速退出和健康失败都应该能在页面或日志中找到原因。
4. **减少重复操作**：运行服务自动生成本机网页封面，服务配置、语言和扫描偏好保存在 SQLite，重启后可以继续使用原来的工作台。
5. **保持本机安全**：默认只监听 `127.0.0.1`，写 API 校验 Host、Origin、Fetch Metadata 和会话令牌；截图只允许 loopback URL，不接管日常浏览器登录态。

### 范围边界

DevHub 面向同一台电脑上的可信开发者，不是远程服务编排平台、多人权限系统或公网监控平台。容器、WSL、Docker Compose、系统托盘、开机启动和云端同步属于后续候选能力；在没有真实平台证据前，不把交叉编译结果当作 macOS/Linux 运行验收。

## 开发路线

路线图文件是 `roadmap/state.json`，交互页面运行在 [http://127.0.0.1:4319](http://127.0.0.1:4319)。路线图只展示保存后的状态，代码、测试和正式验收记录才是实现结论的事实来源。

| 阶段 | 目标 | 当前状态 |
| --- | --- | --- |
| FOUNDATION | 明确范围、安全边界、SQLite 和验收规则 | 已完成 |
| V0.1 | 自动发现、项目识别、启停、日志、网页截图、命令拆分 | 已完成并通过 Windows 本机验证 |
| V0.2 | 设置、中英文、健康分层、日志质量、可控发现、配置迁移、个性化 UI | 已完成并通过 Windows 验证 |
| RELEASE | Windows/macOS/Linux 真实运行证据、安装升级、Release、SBOM、签名 | Windows/Linux WSL 部分证据已有；缺 macOS 与 GitHub Release |
| NEXT | 项目级编排、WSL/Docker Compose、托盘通知等 | 成熟版本发布后再评估 |

当前路线图统计为 **13 / 16 个必需开发节点完成**、**3 个发布节点受阻**、**15 / 18 条正式验收通过**。v0.2 功能已验收。发布门禁卡在 macOS 真机、Linux 截图和 GitHub Release。

### 下一步顺序

1. 在真实 macOS 上跑发现、启停、日志和截图，并补 Linux 截图。
2. 把仓库推到 GitHub，打 `v*` 标签跑 Release 工作流，用 SHA256SUMS 校验后安装/升级。
3. 若有发布者证书，再执行 Authenticode；否则保持校验和策略。
4. 全部 Critical 项通过后再做成熟版本发布门禁。

## 当前进展

### 已完成能力

- Go 后端与 React/TypeScript 前端打包进单个可执行文件。
- 自动发现当前用户的开发监听进程，并识别常见项目框架。
- 保存、启动、停止和删除服务配置；Windows 使用 PowerShell + Job Object，Unix 使用 `/bin/sh` + 进程组。
- 启动命令自动拆分：支持 `cd`、`Set-Location`、`pushd` 加绝对目录的完整命令粘贴。
- 独立浏览器截图队列，自动生成和手动重试 PNG 封面。
- SQLite WAL 配置和日志持久化，最近 500 条日志片段、单条最多 16 KiB。
- 中英文即时切换、自动扫描开关和 5–300 秒扫描间隔，设置在重启后恢复。
- 健康探针：卡片独立显示进程、TCP、HTTP；进程无端口、HTTP 失败和启动超时/快速退出会显示原因并可重试。
- 日志质量：ANSI/UTF-16/CLIXML 清理，以及搜索、暂停、复制、导出和清空。
- 可控发现：包含/排除目录、进程和端口，同一进程多端口合并，卡片显示命中原因，手动服务不会被规则删除。
- 配置迁移：schema 版本迁移、导出/导入/备份，密钥环境变量默认从导出中剔除。
- 个性化 UI：主题、强调色、密度、卡片/列表、项目颜色图标排序和键盘快捷键。

### 尚未完成

- macOS 真实运行与 Linux 截图。
- GitHub Release、Authenticode 签名和成熟版本发布门禁。

### 已执行的验证

在当前 Windows 工作区已经执行并通过：

```text
go test ./...
go vet ./...
cd web && npm test -- --run
cd web && npm run build
cd web && npm run format:check
node roadmap/update.mjs --check
node scripts/verify-roadmap.mjs
git diff --check
```

正式验收明细见 [`docs/roadmap-acceptance.md`](roadmap-acceptance.md)，v0.1 运行证据见 [`docs/devhub-v0.1-verification.md`](devhub-v0.1-verification.md)，v0.2 设置证据见 [`docs/devhub-v0.2-settings.md`](devhub-v0.2-settings.md)，健康与日志开发记录见 [`docs/devhub-v0.2-health-logs.md`](devhub-v0.2-health-logs.md)，可控发现开发记录见 [`docs/devhub-v0.2-discovery-controls.md`](devhub-v0.2-discovery-controls.md)。

## 技术栈

| 层 | 技术 | 用途 |
| --- | --- | --- |
| 后端 | Go 1.26+、`net/http` | 本机 API、安全中间件、服务协调和静态资源服务 |
| 进程与发现 | `gopsutil/v4`、Go `os/exec`、`x/sys` | 监听进程扫描、PID/创建时间核对、跨平台进程树管理 |
| 截图 | `chromedp`、Chrome/Edge/Chromium | 隔离浏览器会话、本机网页封面和资源拦截 |
| 存储 | SQLite（`modernc.org/sqlite`） | 服务配置、设置、日志和 schema 版本 |
| 前端 | React 19、TypeScript 5.9、Vite 7 | 单页工作台、设置、服务卡片和日志界面 |
| UI | 原生 CSS、Lucide React | 深色控制室视觉、响应式布局和图标 |
| 前端测试 | Vitest、Testing Library、jsdom | 翻译、类型和组件行为测试 |
| 路线图 | Node.js ESM、原生 HTML/CSS/JS、JSON | 依赖路线图、验收清单、开发记录和文档浏览 |
| CI | GitHub Actions | Windows、macOS、Linux 的测试、构建、Race detector 和制品上传 |

## 项目环境

### 当前开发机

本记录生成于 Windows 工作区：

```text
操作系统：Microsoft Windows NT 10.0.26200.0
架构：windows/amd64
Go：go1.27.1（项目最低版本 1.26）
Node.js：v25.2.1（项目最低版本 22.12）
npm：11.6.2
git：2.52.0.windows.1
```

GitHub Actions 使用 Node.js 24，并依据 `go.mod` 安装 Go；矩阵包含 `ubuntu-latest`、`windows-latest` 和 `macos-latest`。CI 的跨平台构建不等于三平台运行验收。

### 运行端口和数据

- DevHub 默认地址：`http://127.0.0.1:4780`
- 交互式路线图：`http://127.0.0.1:4319`
- 默认数据目录：Windows `%AppData%\DevHub`、macOS `~/Library/Application Support/DevHub`、Linux `${XDG_CONFIG_HOME:-~/.config}/DevHub`
- 数据目录包含 `devhub.db` 和私有 `covers/`；日志、命令、路径和截图可能含有敏感信息。
- 可用 `-addr` 更换 DevHub 端口，用 `-data` 指定数据目录；启动地址仍必须是 IPv4 loopback。

### 构建和启动

Windows：

```powershell
./scripts/build.ps1
./dist/devhub.exe
```

macOS/Linux：

```sh
sh scripts/build.sh
./dist/devhub
```

开发前端和后端可以分开启动：

```sh
go run ./cmd/devhub
cd web
npm ci
npm run dev
```

Vite 开发服务器默认监听 `127.0.0.1:5173`，把 `/api` 和 `/covers` 代理到 `127.0.0.1:4780`。路线图服务可单独运行：

```sh
node roadmap/server.mjs
```

### 开发约定

- 每次改变能力、依赖、阻碍或发布范围时，同步更新 `roadmap/state.json`、开发记录和正式验收证据。
- `done` 只表示所有检查点、前置条件和真实证据都齐备；构建成功、Mock 或文件存在本身不构成产品验收。
- 提交前至少运行 Go 测试/`vet`、前端测试/构建/格式检查和路线图校验。
- 不提交本机数据库、日志、截图、密钥、`node_modules`、编译产物或用户项目文件。

更多安全边界见 [`SECURITY.md`](../SECURITY.md)，贡献流程见 [`CONTRIBUTING.md`](../CONTRIBUTING.md)，基础使用说明见 [`README.md`](../README.md)。
