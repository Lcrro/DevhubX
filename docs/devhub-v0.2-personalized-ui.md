# DevHub v0.2 个性化控制室验证

日期：2026-09-14 16:57（Windows 本机）

覆盖 `personalized-ui` 与 AC-015。同一隔离实例 `http://127.0.0.1:4782`。

## 已实现

- 主题 dark/light、强调色 lime/cyan/violet/amber、密度 comfortable/compact、视图 cards/list，写入 SQLite。
- 项目颜色、图标（folder/code/server/globe/database）和排序。
- 键盘：`/` 搜索、`N` 添加、`R` 扫描、`L` 切换列表、`Escape` 关闭。
- 窄屏已有侧栏/主栏折叠；列表视图在窄屏改回单列。

## AC-015 证据

- `PUT /api/settings` 保存 `theme=light`、`accent=cyan`、`density=compact`、`serviceView=list`、项目 `Portable` 为 violet/server/order=1，随后 `GET` 读回相同值。
- 生产前端资源包含 `themeLight`、`exportConfig`、`switchList`。
- 焦点样式使用强调色 outline；设置面板与卡片列表共用上述偏好。
