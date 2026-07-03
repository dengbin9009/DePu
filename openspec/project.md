# DePu OpenSpec 项目说明

## 项目目标

DePu 是一个 Vue 前端 + Go 后端的多人德州扑克项目。当前产品方向是从已经可用的规则引擎测试页演进为多人德州扑克对战 v1：保留独立规则测试/回放能力，同时提供账号登录、虚拟金币、房主建房、邀请码加入、多人轮流操作、每手牌结果保存和展示。

本阶段的主要变更是将正式多人对战中依赖 HTTP 轮询和 HTTP 动作提交的部分升级为 socket 通信。账号、钱包、房间创建、邀请码加入和历史查询仍可继续通过 HTTP JSON API 提供。

## 技术栈

- 后端：Go，标准 `net/http` 路由，现有规则引擎位于 `backend/internal/game`、`backend/internal/rules`、`backend/internal/pot`、`backend/internal/handeval`。
- 前端：Vue + TypeScript + Vite，正式多人状态集中在 `frontend/src/composables/useAppState.ts`，API 封装位于 `frontend/src/api/client.ts`。
- 存储：生产默认 MySQL；开发和测试允许 SQLite。业务语义必须在两种数据库配置下保持一致。
- 当前多人 HTTP 路径：`/api/auth/*`、`/api/me/*`、`/api/recharge*`、`/api/rooms*`。
- 独立规则测试页路径：`/api/rulesets`、`/api/games`、`/api/games/{id}/history`、`/api/games/{id}/replay`、`/api/games/{id}/debug/cards`。

## 开发约定

- 正式产品文档使用中文；代码标识、API 路径、目录名、协议事件名和必要技术名词可以保留英文。
- 后端规则引擎是牌局推进、合法动作、牌型评估、分池和结算的唯一权威来源。
- 前端不得自行推断会影响牌局结果的规则，只能展示后端返回的房间状态、手牌状态、合法动作和错误。
- 每个改变牌局状态的动作必须可复盘；正式多人手牌结算必须与钱包流水和历史记录保持原子一致。
- 独立规则测试页必须继续保留，测试页调试设牌和只读回放能力不得进入正式多人主流程。

## OpenSpec 工作流

- 当前行为写入 `openspec/specs/`。
- 新变更写入 `openspec/changes/<change-id>/`。
- 本阶段 change id：`migrate-http-to-socket`。
- 变更实现前先审阅：
  - `openspec/changes/migrate-http-to-socket/proposal.md`
  - `openspec/changes/migrate-http-to-socket/design.md`
  - `openspec/changes/migrate-http-to-socket/tasks.md`
  - `openspec/changes/migrate-http-to-socket/specs/multiplayer-poker/spec.md`

## 测试与验收

实现 socket 升级时至少需要验证：

- 后端规则测试和多人 API 测试仍通过。
- socket 鉴权失败、房间订阅失败、非当前玩家行动、非法动作、断线重连都返回明确错误或恢复状态。
- 多浏览器/多账号同房间时，入座、离座、开局、行动推进、结算、钱包余额和历史展示能够在不依赖 2 秒轮询的情况下更新。
- 规则测试页仍能独立创建测试牌局、调试设牌、查询历史和只读回放。

