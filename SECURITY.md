# Security policy

## Supported versions

当前维护 `0.1.x` 开发线。此项目用于可信用户自己的本机开发环境，不是多租户服务管理平台。

## Reporting a vulnerability

请不要在公开 Issue 中发布可利用的漏洞细节、会话令牌或用户日志。

仓库公开后，请通过 GitHub 的 **Security → Advisories → Report a vulnerability** 私密报告。仓库维护者需先启用 Private vulnerability reporting。若入口不可用，请创建不包含细节的 Issue，请求维护者提供私密联系渠道；在得到渠道前不要上传利用代码或敏感文件。

报告请说明版本、操作系统、复现步骤、影响及建议修复方式。维护者确认修复之前，请协调披露时间。本仓库尚未配置具体维护者邮箱，不提供虚构联系方式。

## Threat model and safeguards

- 后端只绑定 `127.0.0.1`，拒绝任意网卡和远端地址配置。
- 精确检查 Host 与 Origin，拒绝 cross-site Fetch Metadata，防止 DNS rebinding / 浏览器跨站操控。
- 状态变更要求随机、每次启动重新生成的会话令牌及 JSON Content-Type。不开放 CORS。
- 前端具有 CSP、frame-ancestors、nosniff 和 no-referrer；API 不接受任意文件读取。
- 外部进程停止要求确认、同用户和 PID 创建时间核对。托管进程使用 OS 进程组/Job Object 清理。
- 网页地址只能是 loopback HTTP/HTTPS，截图浏览器使用独立临时配置，拦截非本机网络请求和重定向。不关闭浏览器沙箱。
- 限制请求体和日志保留量。SQLite 使用参数化语句。

## Boundaries

可信本机程序能读取会话令牌并调用 API，这不在浏览器跨站攻击防护范围内。启动命令是显式的本机代码执行，拥有 DevHub 用户的权限；DevHub 不隔离恶意代码。不应以管理员/root 身份运行不可信项目。

本地恶意网页、被攻陷的浏览器或同用户恶意进程不属于完整隔离保证。截图虽限制远端 HTTP 请求，但不是对恶意本机网站的通用安全沙箱；页面可访问其他本机服务。不要截图不可信项目。

数据目录包含命令、路径、日志和网页截图，不加密。请保持系统用户目录权限，分享日志前手动检查和脱敏。不要把管理接口或 Vite 代理到公网，不要共享数据目录给不可信用户。

## 制品校验与签名

发布目录必须附带 `SHA256SUMS.txt`。安装前核对该文件中的哈希。当前执行的签名策略是 **SHA-256 校验和**；Windows Authenticode 与 macOS notarization 需要发布者证书，未配置证书时不得声称二进制已代码签名。GitHub Release 工作流生成三平台制品、SBOM 和校验和。
