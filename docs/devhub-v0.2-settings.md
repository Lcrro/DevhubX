# DevHub v0.2 设置与自动扫描验证

日期：2026-09-14 15:18（Windows 本机）

本记录覆盖设置与中英文节点的真实验证。测试通过正在运行的 DevHub `127.0.0.1:4780` 完成，最后恢复为中文、自动发现开启、10 秒间隔。

## 已验证

- 设置面板提供中文和 English 选择，选择 English 后保存，工作台导航、统计、按钮、状态、日志和设置文案即时切换为英文。
- 设置接口只接受 `language`、`autoScan` 和 `scanIntervalSeconds` 三个可写字段；读取接口额外返回只读范围字段，前端保存时不会回传它们。
- 关闭自动发现后等待 6 秒，`lastScan` 保持不变；重新开启并设为 5 秒后，扫描时间在 6 秒内更新。
- 5 秒、10 秒和 300 秒边界均可保存；4 秒和 301 秒均被 HTTP 400 拒绝。
- 保存设置后重启 DevHub，`GET /api/settings` 恢复 `language=zh`、`autoScan=true`、`scanIntervalSeconds=10`。
- `internal/store/store_test.go` 覆盖默认值、SQLite 持久化和最大值归一化；`internal/server/server_test.go` 覆盖 API、非法语言与非法间隔。

## 仍未验收

这份记录只覆盖 AC-009（中英文切换）和 AC-010（自动扫描设置）。健康分层、PowerShell 日志清理、可控发现、配置导入导出、主题/密度/卡片列表等其他 v0.2 节点仍保持未完成。
