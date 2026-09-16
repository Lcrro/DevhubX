# DevHub

[English](README.md) · [中文](README.zh.md)

Repository: [github.com/Lcrro/DevhubX](https://github.com/Lcrro/DevhubX) · Current release **[v0.2.0](https://github.com/Lcrro/DevhubX/releases/tag/v0.2.0)**

DevHub is a **local-only** control room for development services. It keeps project directories, ports, launch commands, process status, logs, and web covers in one place. The UI is **English by default**; switch to Chinese in Settings.

No account, cloud, or database server. The frontend is embedded in a single Go binary. After start, open **http://127.0.0.1:4780**.

![DevHub workbench](docs/screenshots/workbench.png)

Project goals, roadmap, and stack: [`docs/project-overview.md`](docs/project-overview.md). Open issues and pull requests on this repository.

## Features

- **Auto-discovery**: every 10 seconds by default, enumerates TCP listeners for the current user. Settings can disable scanning, change the 5–300s interval, and include/exclude directories, processes, and ports. Project files such as `package.json`, `go.mod`, `pyproject.toml`, and `Cargo.toml` are used for identity. Ports of the same process are merged, with a selectable primary web port and a match reason on each card.
- **Project workbench**: group by project, search name/framework/path/port, filter running or stopped.
- **Start and stop**: save a trusted local command; port conflicts are rejected. Windows uses PowerShell + Job Object; macOS/Linux use `/bin/sh` + a process group.
- **Covers**: running HTTP services enter a screenshot queue using local Chrome / Edge / Chromium. Manual refresh, timeout, and retry are included.
- **Health and logs**: process, TCP, and HTTP are shown separately. stdout/stderr is tailed, with ANSI / PowerShell CLIXML cleanup, search, pause, copy, clear, and export. Last 500 chunks, 16 KiB each.
- **Local persistence**: SQLite WAL for config and logs, PNG covers. Restart restores records and re-checks processes.
- **Loopback security**: bind `127.0.0.1` only, check Host / Origin / Fetch Metadata, require a per-session token for writes. Do not expose the API to the public internet.
- **Backup and import**: export omits logs, covers, and secrets by default. Import backs up the current database first.
- **Personalization**: light/dark theme, accent, density, card or list view, project color/icon/order.

## Quick start

Download a binary from [Releases](https://github.com/Lcrro/DevhubX/releases), or build from source. Building needs **Go 1.26+**, **Node.js 22.12+**, and **npm**. Running a prebuilt binary does not need Node.js or Go; managed projects still need their own runtimes.

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

Windows PowerShell:

```powershell
./scripts/build.ps1
./dist/devhub.exe
```

Linux / macOS: `sh scripts/build.sh`.

Then open [http://127.0.0.1:4780](http://127.0.0.1:4780). Add a service, or start a project in its own terminal and wait for discovery.

### Add a service

1. Click **Add service**, then set name, absolute project path, port, and launch command.
2. Leave the web URL empty to use `http://127.0.0.1:<port>`, or set a local HTTPS URL or path.
3. Click **Start**. Windows commands run in PowerShell; other systems use `/bin/sh`.

You can paste a full terminal command. If it starts with `cd`, `Set-Location`, or `pushd` plus an existing absolute directory, DevHub splits the directory into **Project directory** and keeps the rest as the launch command. `;` and `&&` work, including a missing separator after a quoted path. Relative paths, missing directories, directory-only commands, and `||` are left unchanged.

Example: directory `C:\Projects\shop`, command `npm run dev -- --port 3000`, port `3000`.

Register frontend and backend as two services with the same project name to group them. The port field is only for identity and health. It does **not** rewrite the command or project config.

Discovery records observed listeners and may prefill a command line. It never starts that command by itself.

### Config and data

```sh
devhub -addr 127.0.0.1:4780 -data /absolute/path/to/devhub-data
```

| Flag / env | Default | Meaning |
| --- | --- | --- |
| `-addr` | `127.0.0.1:4780` | IPv4 loopback only; port can change |
| `-data` | `DevHub` under the user config dir | SQLite and covers |
| `-web` | embedded UI | point at `web/dist` while developing |
| `DEVHUB_BROWSER` | auto | absolute path to Chrome / Edge / Chromium |
| `DEVHUB_TEST_BROWSER=1` | off | enable real-browser screenshot tests |

Default data locations:

- Windows: `%AppData%\DevHub`
- macOS: `~/Library/Application Support/DevHub`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/DevHub`

Quit DevHub before copying the data directory. Saved projects are not auto-started. A clean shutdown stops processes DevHub launched in this session; external processes keep running.

## Development

```sh
# terminal 1
go run ./cmd/devhub
# terminal 2
cd web
npm ci
npm run dev
```

Vite listens on `127.0.0.1:5173` and proxies `/api` and `/covers` to `127.0.0.1:4780`. Do not reverse-proxy the dev server onto another network.

```sh
go test ./...
go vet ./...
cd web
npm test
npm run build
```

```sh
DEVHUB_TEST_BROWSER=1 go test ./internal/cover -run TestCaptureIntegration -v
```

PowerShell: `$env:DEVHUB_TEST_BROWSER='1'` then the same `go test` command.

## Layout

```text
cmd/devhub/          entrypoint, flags, shutdown
internal/server/     HTTP API, security, coordination
internal/discovery/  listeners and project identity
internal/runner/     process lifecycle
internal/cover/      isolated browser screenshots
internal/logfmt/     ANSI / CLIXML / encoding cleanup
internal/store/      SQLite config and logs
internal/model/      data types
web/                 React / TypeScript / Vite UI
scripts/             build and install scripts
.github/             CI, issue, and PR templates
```

## Limits

- **Discovery** only sees current-user processes with a readable working directory, up to six parents. System install paths (Program Files, `/Applications`, `/usr`, `/opt`, Snap, Flatpak, …) are skipped. Include/exclude rules apply after that and never delete manual services. Containers and WSL processes may not group. TCP listen is not HTTP ready.
- **Health** chips are process / TCP / HTTP. HTTP healthy means the local URL returned 2xx/3xx, not that every API works.
- **External processes** require confirm, PID, birth time, and owner checks. Only that PID is killed. An external supervisor may restart it. Historical terminal output is not imported.
- **Stop** force-kills the process tree. There is no app-specific graceful shutdown.
- **Screenshots** use a throwaway browser profile, loopback HTTP/HTTPS only. Missing Chrome/Edge does not block service management.
- **Trust model**: one trusted local user, not multi-tenant auth. Commands run as the DevHub user. Do not run untrusted projects as Administrator/root.
- **Platforms**: Windows has real process-tree and screenshot evidence. GitHub Actions covers Windows, macOS, and Linux builds.

## License

[MIT](LICENSE). See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

Main dependencies: [gopsutil](https://github.com/shirou/gopsutil), [modernc SQLite](https://pkg.go.dev/modernc.org/sqlite), [chromedp](https://github.com/chromedp/chromedp), [React](https://react.dev/), [Vite](https://vite.dev/), [Lucide](https://lucide.dev/).

## Interactive roadmap

```sh
node roadmap/server.mjs
```

Opens `http://127.0.0.1:4319`. The page rereads `roadmap/state.json` every two seconds and does not infer completion from code. Rules: [AGENTS.md](AGENTS.md). Acceptance: [docs/roadmap-acceptance.md](docs/roadmap-acceptance.md).
