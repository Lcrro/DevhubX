# Contributing to DevHub

源码仓库：[github.com/Lcrro/DevhubX](https://github.com/Lcrro/DevhubX)。欢迎修复问题、改善跨平台兼容性和交互体验。较大的功能调整请先在该仓库创建 Issue 讨论使用场景。

## 本地准备

安装 Go 1.26+、Node.js 22.12+。按 README 构建前端，随后运行 Go 程序。截图集成测试还需要 Chrome / Edge / Chromium。

## 提交前检查

```sh
gofmt -w cmd internal web/embed.go
go test ./...
go vet ./...
cd web
npm ci
npm test
npm run build
```

修改进程管理逻辑时请增加真实子进程生命周期测试；修改 API 时请覆盖输入校验和 Host/Origin/令牌边界。尽量在相关操作系统上验证，明确注明没有验证的平台。

## 约定

- Go 按职责放在 `internal/`，操作系统差异通过平台文件/build tags 隔离。
- TypeScript 开启 strict；界面必须有加载、空状态和失败反馈，使用可访问的原生控件。
- 不得改为监听所有网卡、禁用截图浏览器沙箱或默认提权运行。
- 不提交用户数据、日志、截图、密钥、`node_modules`、本地工具链或编译产物。
- 依赖变更同时提交 `go.sum` / `package-lock.json`。
- 保持 PR 范围清晰，在描述中写明问题、改动和实际运行的验证。

欢迎使用中文或英文提交 Issue 和 PR。尊重贡献者，讨论具体问题，不进行人身攻击。
