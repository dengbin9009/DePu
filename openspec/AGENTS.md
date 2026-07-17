# OpenSpec 协作说明

本目录是 DePu 当前版本的 OpenSpec 规格入口。正式产品文档使用中文；代码标识、API 路径、目录名、socket 事件名和必要技术名词可以保留英文。

## 当前变更

当前开发变更：`table-playability-hardening`

目标：在既有 socket 多人能力上加固牌桌响应式安全区、关键操作反馈、连续手牌、房主移交、事件顺序、牌面可读性、牌谱明细和生产级质量门。

## 必读文档

- 项目上下文：`openspec/project.md`
- 当前多人基线：`openspec/specs/multiplayer-poker/spec.md`
- 变更提案：`openspec/changes/table-playability-hardening/proposal.md`
- 技术设计：`openspec/changes/table-playability-hardening/design.md`
- 实施任务：`openspec/changes/table-playability-hardening/tasks.md`
- 增量规格：`openspec/changes/table-playability-hardening/specs/multiplayer-poker/spec.md`

## 边界

- 不把真实支付、好友关系、语音、大厅匹配、机器人、新赛事系统或反作弊系统纳入本次变更。
- 不移除规则测试页。
- 不让前端自行计算会影响牌局结果的规则。
- 不恢复固定 HTTP 轮询掩盖 socket 问题；HTTP 只保留资源查询和显式一致性兜底。
