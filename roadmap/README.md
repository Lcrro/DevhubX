# DevHub 交互式项目路线图

在 DevHub 项目根目录运行 `node roadmap/server.mjs`，打开输出的本机地址；Windows 也可以运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/start-roadmap.ps1`。默认端口为 4319，避开路线图工具自身的 4318 演示服务。

页面每两秒读取 `roadmap/state.json` 和正式验收表。它不会分析代码、测试、提交或外部系统，只有保存到项目文件的状态才会显示。

Validate data with `node roadmap/update.mjs --check`. Update a node and append an activity record atomically with `node roadmap/update.mjs <node-id> <status> "evidence-based note"`.

Run the HTTP and document-boundary check with `node scripts/verify-roadmap.mjs`. All evidence and document paths must be project-root-relative. A document is served only when referenced by `state.json`.

实现事实以代码、测试和验证记录为准。开发检查点与正式验收保持独立，维护约定见项目根目录 `AGENTS.md`。
