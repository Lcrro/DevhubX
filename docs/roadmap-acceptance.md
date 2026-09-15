# DevHub 正式验收

开发检查点不等于正式验收。只有在要求的系统、浏览器和升级场景中取得可复现证据后，才能把对应行标为 `[x]`。当前已勾选项限于 2026-09-14 的 Windows 本机验证及自动化测试范围；macOS、Linux、安装器和成熟版功能保持未验收。

| Status | ID / Requirement | Action or setup | Pass criteria | Level |
| --- | --- | --- | --- | --- |
| [x] | AC-001 / 服务管理闭环 | 保存一个测试服务，从 DevHub 启动、访问、查看日志并停止 | 服务可访问，日志增量出现，停止后监听端口关闭 | Critical |
| [x] | AC-002 / 进程树与端口冲突 | 启动带监听端口的子进程，再尝试用相同端口启动另一个服务 | 冲突被拒绝；停止后托管进程树不残留监听 | Critical |
| [x] | AC-003 / 自动发现开发服务 | 从带 package.json 的目录启动当前用户监听进程 | DevHub 显示正确项目名、框架、PID 和端口 | Critical |
| [x] | AC-004 / 排除桌面应用误判 | 保持 QQ NT 等安装目录程序监听本地端口并触发扫描 | Program Files 等系统应用目录不会生成自动发现记录 | Important |
| [x] | AC-005 / 数据恢复 | 保存服务、封面和日志后重启 DevHub | 配置、封面引用与保留范围内日志恢复，临时记录可完整删除 | Critical |
| [x] | AC-006 / 本机截图边界 | 对 loopback 测试页截图，并尝试远端、file、带凭证和非 loopback URL | 本机页产生 PNG；不允许的 URL 被拒绝，远端子资源被拦截 | Critical |
| [x] | AC-007 / 本机 API 安全 | 使用错误 Host、跨站 Origin、缺失/错误令牌调用写 API | 请求被拒绝；服务只绑定 127.0.0.1 | Critical |
| [x] | AC-008 / 完整启动命令转换 | 保存 cd、Set-Location、pushd 及漏分隔符的完整命令 | 已存在绝对目录被拆分；相对、缺失、只有 cd 或条件回退命令不被改写 | Important |
| [x] | AC-009 / 中英文切换 | 在设置中切换中文和英文并重启应用 | 全部可见文案切换且偏好恢复，没有混合语言或布局截断 | Important |
| [x] | AC-010 / 自动扫描设置 | 关闭扫描并分别设置允许的最短、常用和最长间隔 | 定时器动态更新，关闭后不扫描，非法间隔被拒绝且设置持久化 | Critical |
| [x] | AC-011 / 多层健康状态 | 分别制造进程存在但无端口、TCP 可达但 HTTP 失败、HTTP 健康三种情形 | UI 独立、准确且可解释地显示进程、TCP 和 HTTP 状态 | Critical |
| [x] | AC-012 / 日志与启动失败 | 产生 ANSI、中文、PowerShell CLIXML、超时和快速退出输出 | 日志可读可搜索，卡片直接显示失败原因并可重试 | Critical |
| [x] | AC-013 / 发现规则 | 配置包含/排除目录与多端口进程主网页 | 扫描结果遵循设置、解释命中原因且不会删除手动服务 | Important |
| [x] | AC-014 / 配置迁移与导出 | 从旧数据库升级，导出后在空数据目录导入 | 服务与设置无损恢复，敏感值可剔除，失败不会破坏原库 | Critical |
| [x] | AC-015 / 个性化 UI | 切换主题、强调色、密度、卡片/列表视图并测试键盘与窄屏 | 设置恢复；大量服务仍可操作；焦点、对比度和布局通过验收 | Important |
| [ ] | AC-016 / 三平台运行 | 在受支持 Windows、macOS、Linux 上执行发现、启停、日志和截图流程 | 三个平台都有真实运行记录，权限失败和系统差异有清晰反馈 | Critical |
| [ ] | AC-017 / 安装与发布 | 从 GitHub Release 在干净系统安装、升级、验证校验和并卸载 | 制品可验证、升级保留数据、卸载不删除项目文件且签名策略达成 | Critical |
| [ ] | AC-018 / 成熟版本回归 | 对发布候选执行全部必需验收、数据库升级和安全回归 | 所有 Critical 项通过并链接证据，未通过项阻止正式发布 | Critical |

## 已通过项的证据

- AC-009、AC-010：[`docs/devhub-v0.2-settings.md`](devhub-v0.2-settings.md)，包含浏览器操作、动态扫描观察、边界值和重启恢复结果。
- AC-011、AC-012：[`docs/devhub-v0.2-health-logs.md`](devhub-v0.2-health-logs.md)，包含三类健康探针、启动超时、快速退出、ANSI/CLIXML 日志和重试。
- AC-013：[`docs/devhub-v0.2-discovery-controls.md`](devhub-v0.2-discovery-controls.md)，包含双端口合并、排除端口/目录和手动服务保留。
- AC-014：[`docs/devhub-v0.2-portability.md`](devhub-v0.2-portability.md)，含 v1 迁移、密钥剔除、失败导入和备份。
- AC-015：[`docs/devhub-v0.2-personalized-ui.md`](devhub-v0.2-personalized-ui.md)，含主题/密度/列表视图设置恢复。
