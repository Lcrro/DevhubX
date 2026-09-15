# DevHub v0.2 可控自动发现验证

日期：2026-09-14 16:35（Windows 本机）

本记录覆盖路线图 `discovery-controls` 与正式验收 AC-013。验证使用隔离数据目录启动的 DevHub `127.0.0.1:4781`。

## 已实现

- 设置可配置包含/排除目录、进程和端口；空包含列表不额外限制；系统安装目录始终排除。
- 同一 PID 的多个端口合并为一条服务，主网页优先保留用户已选端口，否则按常见 Web 端口或最小端口选择。
- 卡片显示命中原因、进程名和额外端口；编辑已停止服务时可选择主网页端口。
- 规则不会删除手动服务；被排除的自动发现记录在放宽规则后可再次出现。

## AC-013 真实验收

在带 `package.json`（`vite`）的临时目录启动同一个 `node.exe`，同时监听 `127.0.0.1:57624` 和 `127.0.0.1:57625`。扫描后：

- 只出现 **1** 条 `source=discovered` 服务，名称为 `multi-port`。
- `ports=57624,57625`，主网页端口为 `57624`。
- `discoveryReason=merged-ports`，`processName=node.exe`。

随后把 `57625` 写入排除端口并扫描：

- 该服务只保留 `ports=57624`，不再包含 `57625`。

再把该项目目录写入排除目录并扫描：

- `multi-port` 自动发现记录被移除。
- 事先手动添加的 `manual-keep` 仍在，且 `source=manual`。

设置面板含包含/排除目录、进程和端口字段；生产前端资源包含 `discoveryRules` / `includeDirectories` 文案。相对路径规则仍返回 HTTP 400。

## 环境

- 操作系统：Windows，本机 loopback
- 验证二进制：`dist/devhub-verify.exe`
- 验证地址：`http://127.0.0.1:4781`
- 数据目录：`%TEMP%\devhub-ac-verify`
