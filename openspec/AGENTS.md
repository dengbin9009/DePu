# OpenSpec 协作说明

本目录是 DePu 当前版本的 OpenSpec 规格入口。正式产品文档使用中文；代码标识、API 路径、目录名、socket 事件名和必要技术名词可以保留英文。

## 当前变更

当前开发变更：`migrate-http-to-socket`

目标：将正式多人对战从 HTTP 轮询和 HTTP 动作提交升级为 socket 驱动的房间实时同步与玩家操作，同时保留账号、钱包、建房、邀请码加入、历史查询和独立规则测试页的 HTTP 能力。

## 必读文档

- 项目上下文：`openspec/project.md`
- 当前多人基线：`openspec/specs/multiplayer-poker/spec.md`
- 变更提案：`openspec/changes/migrate-http-to-socket/proposal.md`
- 技术设计：`openspec/changes/migrate-http-to-socket/design.md`
- 实施任务：`openspec/changes/migrate-http-to-socket/tasks.md`
- 增量规格：`openspec/changes/migrate-http-to-socket/specs/multiplayer-poker/spec.md`
- socket 事件契约草案：`openspec/changes/migrate-http-to-socket/contracts/socket-events.md`

## 边界

- 不把真实支付、聊天、大厅匹配、机器人、倒计时托管、排行榜或反作弊系统纳入本次变更。
- 不移除规则测试页。
- 不让前端自行计算会影响牌局结果的规则。
- 不用 socket 替代所有 HTTP 接口；本次只迁移正式多人房间实时同步和正式手牌操作通道。
