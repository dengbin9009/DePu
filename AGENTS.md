# DePu 项目协作说明

本项目当前版本使用 OpenSpec 进行规格驱动开发；历史 Spec Kit 文档仍保留为已实现基线参考。正式产品文档使用中文；代码标识、API 路径、目录名、socket 事件名和必要技术名词可以保留英文。

当前目标是 Vue 前端 + Go 后端的多人德州扑克对战 v1，保留独立规则引擎测试页，新增账号登录、虚拟金币、房主建房邀请码加入、正式多人轮流操作，以及每手牌结果保存与展示；运行、开发和测试统一使用 MySQL。当前阶段在既有 socket 多人能力上加固牌桌可玩性、房间生命周期、事件顺序、牌谱完整性和生产级验收。

<!-- OPENSPEC START -->
当前 OpenSpec 变更：`openspec/changes/table-playability-hardening/`

必读入口：
- `openspec/project.md`
- `openspec/specs/multiplayer-poker/spec.md`
- `openspec/changes/table-playability-hardening/proposal.md`
- `openspec/changes/table-playability-hardening/design.md`
- `openspec/changes/table-playability-hardening/tasks.md`
- `openspec/changes/table-playability-hardening/specs/multiplayer-poker/spec.md`
<!-- OPENSPEC END -->
