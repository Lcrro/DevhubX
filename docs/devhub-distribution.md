# 安装与发布准备

日期：2026-09-14

AC-017 要求从 GitHub Release 在干净系统安装、升级、校验并卸载。仓库目前 **没有 git 提交和远端**，因此没有真实 GitHub Release。下面是本机已执行的发布准备。

## 已执行

- `scripts/install.ps1` 把 `dist/devhub.exe` 装到 `%LOCALAPPDATA%\Programs\DevHub`，并写快捷方式。数据目录 `%APPDATA%\DevHub` 不被安装程序修改。
- `scripts/uninstall.ps1` 删除程序目录，保留数据目录。实测：安装后 exe/lnk 存在；卸载后程序目录消失，`%APPDATA%\DevHub` 仍在。
- `node scripts/sbom.mjs` 生成 `dist/sbom.json`（218 个组件）。
- `scripts/checksums.ps1` 生成 `dist/SHA256SUMS.txt`。
- `.github/workflows/release.yml` 在 `v*` 标签上构建 linux/windows/macos 制品、SBOM、校验和并创建 GitHub Release。
- 签名策略见 `SECURITY.md`：当前执行 SHA-256 校验和；没有 Authenticode 证书，不声称已代码签名。

## 未完成

- 没有 GitHub 远端，无法从 Release 安装。
- 没有发布者证书，未做 Authenticode / notarization。
- 没有第二台干净机器上的升级路径实测。
