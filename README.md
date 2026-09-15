# DevHub

仓库：[github.com/Lcrro/DevhubX](https://github.com/Lcrro/DevhubX) · 当前版本 **0.2.0**

一个只在本机运行的开发服务工作台。用 **Go + React / TypeScript + SQLite** 把项目目录、端口、启动命令、进程状态、日志和网页封面放在一起。

项目目标、开发路线、进展、技术栈和环境见 [`docs/project-overview.md`](docs/project-overview.md)。 Issue 和 Pull Request 请提到本仓库。

不需要账号、云服务或数据库服务器。前端打包到 Go 可执行文件中，启动后访问 **http://127.0.0.1:4780**。

## 功能

- **自动发现**：默认每 10 秒枚举当前用户的 TCP 监听进程；可在设置中关闭扫描或调整 5–300 秒间隔，并配置包含/排除目录、进程和端口。读取进程目录并识别 package.json、go.mod、pyproject.toml、Cargo.toml 等项目文件。同一进程的多个端口会合并成一条服务，可选择主网页；每条结果会说明命中原因。
- **项目工作台**：按项目分组，搜索项目名称、服务、框架、路径或端口，筛选运行/停止状态。
- **启动与停止**：保存任意可信的本机启动命令；启动前检查端口冲突。Windows 使用 PowerShell + Job Object，macOS/Linux 使用 `/bin/sh` + 进程组。
- **自动截图**：运行中的服务进入独立截图队列，使用本机 Chrome / Edge / Chromium。支持手动更新、超时、失败原因和自动重试。
- **状态与日志**：区分托管运行、启动中、外部运行、已停止，并分别显示进程、TCP 和 HTTP 健康；stdout/stderr 实时增量读取，自动清理 ANSI/常见 PowerShell CLIXML，日志支持搜索、暂停、复制、清空和导出，保留最近 500 条片段，每条最多 16 KiB。
- **本地持久化**：SQLite WAL 保存配置和日志，PNG 保存封面；重启后恢复项目记录并重新核对进程。
- **本机安全边界**：只允许监听 `127.0.0.1`，检查 Host/Origin/Fetch Metadata，写操作要求随机会话令牌；不向公网发布管理接口。
- **中英文设置**：工作台文案支持中文和 English，语言偏好、自动发现开关和扫描间隔保存在本机 SQLite，重启后恢复。
- **备份与导入**：导出服务和设置；默认去掉日志、封面和密钥。导入前自动备份数据库。
- **个性化**：深色/浅色主题、强调色、密度、卡片或列表视图，以及项目颜色、图标和排序。

## 快速开始

构建需要 **Go 1.26+、Node.js 22.12+、npm**。建议使用当前 Go 稳定版与 Node.js LTS。正常使用编译好的程序不需要 Node.js 或 Go；被管理的项目仍需要自己的运行时。

```sh
git clone https://github.com/Lcrro/DevhubX.git
cd DevhubX
cd web
npm ci
npm run build
cd ..
go build -o dist/devhub ./cmd/devhub
./dist/devhub
```

Windows PowerShell：

```powershell
./scripts/build.ps1
./dist/devhub.exe
```

打开 [DevHub 本地工作台](http://127.0.0.1:4780)。可以直接添加服务，也可以先在原来的终端启动项目，等待自动发现。

Linux / macOS 也可使用 `sh scripts/build.sh`。

### 添加一个服务

1. 点击「添加服务」，填写名称、项目绝对路径、端口和启动命令。
2. 网页 URL 留空时使用 `http://127.0.0.1:<端口>`，也可填写本机 HTTPS 地址或子路径。
3. 点击「启动」。Windows 的命令按 PowerShell 语法执行，其他系统按 `/bin/sh` 语法执行。

也可以直接粘贴完整终端命令。若命令以 `cd`、`Set-Location` 或 `pushd` 加一个已存在的绝对目录开头，DevHub 会在保存时自动把路径移到“项目目录”，并保留后面的实际启动命令。支持 `;`、`&&`，也兼容引号路径后漏写分隔符的常见粘贴形式。相对路径、目录不存在、只有切换目录而没有启动动作，以及 `||` 条件命令不会被改写。

例如：项目目录 `C:\Projects\shop`，启动命令 `npm run dev -- --port 3000`，端口 `3000`。

前后端分别登记为两个服务，使用相同的项目名称即可归组。端口字段只用于识别和状态检查，**不会改写启动命令或项目配置**。含有空格的可执行路径请按对应 shell 语法引用。

自动发现只登记观察到的服务，不会猜测或自动执行启动命令。若需要从 DevHub 重启外部服务，先停止，再编辑并填写确认过的启动命令。

### 配置与数据

```sh
devhub -addr 127.0.0.1:4780 -data /absolute/path/to/devhub-data
```

| 参数/变量 | 默认值 | 说明 |
| --- | --- | --- |
| `-addr` | `127.0.0.1:4780` | 只接受 IPv4 loopback，可更换端口 |
| `-data` | 系统用户配置目录下的 `DevHub` | SQLite 与封面目录 |
| `-web` | 内嵌前端 | 开发时可指定 `web/dist` |
| `DEVHUB_BROWSER` | 自动查找 | Chrome / Edge / Chromium 可执行文件绝对路径 |
| `DEVHUB_TEST_BROWSER=1` | 关闭 | 启用需要真实浏览器的截图集成测试 |

默认数据位置：

- Windows：`%AppData%\DevHub`
- macOS：`~/Library/Application Support/DevHub`
- Linux：`${XDG_CONFIG_HOME:-~/.config}/DevHub`

备份时先退出 DevHub，再复制整个数据目录。不会自动启动上次保存的项目。正常关闭 DevHub 会结束由本次会话启动的托管服务；外部服务保持运行。

## 开发

生产构建由单个 Go HTTP 服务提供前后端，开发时可分别运行：

```sh
# 终端 1
go run ./cmd/devhub
# 终端 2
cd web
npm ci
npm run dev
```

Vite 默认运行于 `127.0.0.1:5173`，将 `/api` 和 `/covers` 代理到 `127.0.0.1:4780`。只用于本地开发；代理重写 Host 并去除开发页面的 Origin，仍转发 Fetch Metadata 和会话令牌。请不要把开发服务器反向代理到其他网络。

```sh
go test ./...
go vet ./...
cd web
npm test
npm run build
```

真实截图测试：

```sh
DEVHUB_TEST_BROWSER=1 go test ./internal/cover -run TestCaptureIntegration -v
```

PowerShell 使用 `$env:DEVHUB_TEST_BROWSER='1'` 后运行同一 Go 测试命令。

## 结构

```text
cmd/devhub/          程序入口、参数、退出处理
internal/server/    HTTP API、安全中间件、服务协调
internal/discovery/ 监听进程与项目识别
internal/runner/    跨平台进程生命周期
internal/cover/     隔离浏览器会话、截图
internal/logfmt/    ANSI、CLIXML 与编码清理
internal/store/     SQLite 配置与日志
internal/model/     数据结构
web/                React / TypeScript / Vite 前端与内嵌资源
scripts/            跨平台构建脚本
.github/            CI、Issue 和 PR 模板
```

## 已知边界

- **发现范围**：只枚举当前用户且可读取工作目录的进程，向上最多查找六层。为避免把 QQ、编辑器等桌面应用误判为项目，自动发现会排除 Windows 的 Program Files/ProgramData/Windows、macOS 的 Applications/System/Library，以及 Linux 的 `/usr`、`/opt`、Snap 和 Flatpak 安装目录；这些路径仍可手动添加。设置中的包含/排除规则在系统目录排除之后生效，不会删除手动服务。不可读取的目录、无项目标记的服务、容器和 WSL 内的进程可能无法自动归组。TCP 监听不等于 HTTP 就绪，非网页服务的截图可能失败。
- **进程状态**：卡片分别显示进程、TCP 和 HTTP 探针；HTTP 健康只代表本机 URL 返回 2xx/3xx，不代表业务接口完整可用。启动命令常驻但未监听时会显示进程运行、TCP 不可达，请检查日志。
- **外部进程**：停止前必须确认并核对 PID、创建时间和所属用户，只结束该 PID；外部 supervisor 可能重新启动它。不能接管外部终端的历史 stdout/stderr。
- **进程清理**：托管服务停止使用强制结束进程树，未提供应用特定的优雅关闭协议。Windows Job Object 在管理进程终止时清理子进程；Unix 非正常崩溃可能留下进程，下次会作为外部服务重新发现。不要使用守护化、脱离进程组或提权的启动命令。
- **截图**：使用独立临时浏览器配置，无登录态；只加载本机 HTTP/HTTPS 资源，外部字体/图片被阻止。没有安装浏览器不会影响服务管理。自签名证书不会被自动信任。
- **安全模型**：面向同一台电脑的可信用户，不是多用户权限系统。命令拥有启动 DevHub 的用户权限；不要以管理员/root 身份运行不可信项目。日志和截图可能包含敏感信息。
- **跨平台**：Windows 上实际验证进程树和浏览器截图；GitHub Actions 配置覆盖 Windows、macOS、Linux，提交到 GitHub 后才会产生远端 CI 结果。

## 开源

采用 [MIT License](LICENSE)。提交改动前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)，安全问题请按 [SECURITY.md](SECURITY.md) 私密报告。

主要依赖：[gopsutil](https://github.com/shirou/gopsutil)、[modernc SQLite](https://pkg.go.dev/modernc.org/sqlite)、[chromedp](https://github.com/chromedp/chromedp)、[React](https://react.dev/)、[Vite](https://vite.dev/)、[Lucide](https://lucide.dev/)。依赖遵循各自许可证。

## 交互式路线图

项目内置证据约束的依赖路线图，用于区分已验证的 v0.1 能力、v0.2 成熟化工作、发布门禁和未来选项：

```sh
node roadmap/server.mjs
```

默认打开 `http://127.0.0.1:4319`。路线图每两秒读取保存后的 `roadmap/state.json`，不会自动判断代码完成度。维护规则见 [AGENTS.md](AGENTS.md)，正式验收见 [docs/roadmap-acceptance.md](docs/roadmap-acceptance.md)。
