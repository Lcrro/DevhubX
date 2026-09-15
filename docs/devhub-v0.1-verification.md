# DevHub v0.1 验证记录

日期：2026-09-14  
环境：Windows，Go 1.27.1，Node.js 25.2.1，Chrome/Edge 可用。  
范围：当前本机开发版本；不代表 macOS、Linux、安装器或公开 Release 已验收。

## 自动化检查

- `go test ./...`：通过。覆盖 SQLite 恢复与日志保留、真实监听进程发现、系统应用目录排除、loopback URL、真实浏览器截图、进程启动/端口冲突/停止、安全 API 和完整启动命令转换。
- `go vet ./...`：通过。
- `cd web && npm test`：4 项前端测试通过。
- `cd web && npm run build`：React/TypeScript 生产构建通过。
- Windows、Linux amd64、macOS arm64 目标均可编译；后两者没有在真实系统执行，因此不计运行验收。

## Windows 本机工作流

在隔离测试目录中登记 Node HTTP 服务，通过 DevHub 完成添加、启动、访问、自动截图、查看增量日志和停止。HTTP 返回 200；停止后端口关闭。随后重启 DevHub，确认服务配置、PNG 封面和 5 条测试日志恢复，再删除测试记录。

实际项目 `UP DB` 使用端口 8765 验证了完整 PowerShell 命令修正后的启动：服务显示为 DevHub 托管并返回 HTTP 200。原始 `cd "..." powershell ...` 因缺少分隔符触发 `Set-Location` 参数错误；保存命令已修正，通用转换器已有单元和 API 集成测试。

QQ NT 9.9.28 的同一 `QQ.exe` 曾因监听 4001、4301、4310、5283、9210 且资源目录含 `qq-chat` package.json 被误识别为五个项目服务。加入系统应用安装目录排除后重新扫描，`qq-chat` 自动发现记录为 0，真实用户项目继续保留。

## 未验证与已知缺口

- macOS、Linux 没有真实运行验证。
- 当前健康状态主要依据进程身份和 TCP 端口，不等于 HTTP 业务健康。
- PowerShell 失败输出可能包含 CLIXML 或乱码，需在 v0.2 改善。
- 尚无安装包、签名、GitHub Release、数据库版本迁移或配置导入导出。
- 中英文、扫描开关/间隔及个性化 UI 尚未实现。
