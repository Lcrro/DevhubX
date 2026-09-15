# DevHub v0.2 配置迁移与备份验证

日期：2026-09-14 16:57（Windows 本机）

覆盖 `configuration-portability` 与 AC-014。验证使用 `dist/devhub-verify.exe` 和隔离数据目录 `http://127.0.0.1:4782`。

## 已实现

- SQLite `user_version` 从 1/2 升到 3；更高版本拒绝打开。失败的迁移在事务中回滚。
- `POST /api/export` 默认剔除运行时字段、封面、日志和密钥环境变量。
- `POST /api/import` 先 `VACUUM INTO` 备份，再事务替换服务与设置；非法包返回 400，原库不变。
- `POST /api/backup` 写入数据目录 `backups/`。
- 服务可保存环境变量；名称含 token/secret/password 的项按密钥处理，界面用密码框，导出默认值为空。

## AC-014 证据

- 自动化：`TestMigrateFromV1AndUnknownVersion`、`TestExportImportRoundTripAndSecrets`、`TestImportFailureLeavesOriginal`、`TestExportImportAndBackupAPI`。
- 运行中的 4782：保存带 `API_TOKEN=s3cret` 的服务，导出 JSON 不含该值；非法导入 HTTP 400；合法导入 `imported=4`，导入后密钥仍为空。
- 备份文件写入 `%TEMP%\devhub-ac-014\backups\devhub-*.db` 且可再次打开。

空数据目录导入由 `ReplaceAll` 在新 Store 上的单元测试覆盖。
