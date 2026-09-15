# DevHub repository instructions

`roadmap/state.json` 是交互式路线图的展示状态源，但代码、测试、正式验收表和验证记录才是实现结论的事实来源。开发改变能力、依赖、阻碍或发布范围时，同步更新路线图检查点与活动记录；不得仅凭文件存在、编译成功、Mock 或 HTTP 200 宣称完整产品验收。

把正式验收维护在 `docs/roadmap-acceptance.md`，只在要求的平台和场景取得可复现证据后勾选。状态为 `done` 的节点必须完成全部检查点、完成所有前置并链接真实证据。`future` 节点不计入当前发布完成度。

路线图修改后运行：

```sh
node roadmap/update.mjs --check
node scripts/verify-roadmap.mjs
```

DevHub 的常规改动仍需运行相关 Go/前端测试。不要把交互路线图当成自动代码分析器；页面每两秒读取保存后的项目文件，不会自行判断实现或验收状态。
