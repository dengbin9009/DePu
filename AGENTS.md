# DePu 项目协作说明

本项目当前版本使用 OpenSpec 进行规格驱动开发；历史 Spec Kit 文档仍保留为已实现基线参考。正式产品文档使用中文；代码标识、API 路径、目录名、socket 事件名和必要技术名词可以保留英文。

当前目标是 Vue 前端 + Go 后端的多人德州扑克对战 v1，保留独立规则引擎测试页，新增账号登录、虚拟金币、房主建房邀请码加入、正式多人轮流操作，以及每手牌结果保存与展示；生产默认 MySQL，开发/测试允许 SQLite。本版本计划将正式多人对战从 HTTP 轮询/HTTP 动作提交升级为 socket 实时同步与 socket command。

<!-- OPENSPEC START -->
当前 OpenSpec 变更：`openspec/changes/migrate-http-to-socket/`

必读入口：
- `openspec/project.md`
- `openspec/specs/multiplayer-poker/spec.md`
- `openspec/changes/migrate-http-to-socket/proposal.md`
- `openspec/changes/migrate-http-to-socket/design.md`
- `openspec/changes/migrate-http-to-socket/tasks.md`
- `openspec/changes/migrate-http-to-socket/specs/multiplayer-poker/spec.md`
<!-- OPENSPEC END -->
