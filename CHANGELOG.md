# Changelog

## 0.2.0 - 2026-09-14

### Added
- Settings for language, auto-scan interval, and include/exclude discovery rules.
- Layered process / TCP / HTTP health, start timeout and exit reasons, log search and CLIXML cleanup.
- Same-process port merging with a selectable primary web port.
- SQLite schema migrations, JSON export/import, database backup, and secret env stripping.
- Theme, accent, density, card/list view, project colors/icons/order, and keyboard shortcuts.
- Windows install/uninstall scripts, SBOM generator, SHA-256 checksums, and a GitHub Release workflow.

### Security
- Export omits secrets, logs, and covers unless explicitly requested.
- Import backs up the current database first and refuses to run while DevHub-managed processes are running.
- Executables are verified by SHA-256; Authenticode/notarization requires a publisher certificate and is documented in SECURITY.md.
