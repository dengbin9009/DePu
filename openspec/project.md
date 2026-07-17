# DePu OpenSpec 项目说明

## 项目目标

DePu 是一个 Vue 前端 + Go 后端的多人德州扑克项目。当前产品方向是从已经可用的规则引擎测试页演进为多人德州扑克对战 v1：保留独立规则测试/回放能力，同时提供账号登录、虚拟金币、房主建房、邀请码加入、多人轮流操作、每手牌结果保存和展示。

本阶段的主要变更是在既有 socket 正式多人对战上加固牌桌可玩性、房间生命周期、事件顺序、牌谱完整性和生产级质量门。账号、钱包、房间创建、邀请码加入和历史查询继续通过 HTTP JSON API 提供，正式实时状态继续通过 socket 同步。

## 技术栈

- 后端：Go，标准 `net/http` 路由，现有规则引擎位于 `backend/internal/game`、`backend/internal/rules`、`backend/internal/pot`、`backend/internal/handeval`。
- 前端：Vue + TypeScript + Vite，正式多人状态集中在 `frontend/src/composables/useAppState.ts`，API 封装位于 `frontend/src/api/client.ts`。
- 存储：运行、开发和测试统一使用 MySQL。测试可通过临时 MySQL database 隔离数据，但不得切换到其他存储引擎。
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
- 本阶段 change id：`table-playability-hardening`。
- 变更实现前先审阅：
  - `openspec/changes/table-playability-hardening/proposal.md`
  - `openspec/changes/table-playability-hardening/design.md`
  - `openspec/changes/table-playability-hardening/tasks.md`
  - `openspec/changes/table-playability-hardening/specs/multiplayer-poker/spec.md`

## 测试与验收

实现牌桌生产加固时至少需要验证：

- 后端规则测试和多人 API 测试仍通过。
- socket 鉴权失败、房间订阅失败、事件乱序、非当前玩家行动、非法动作和断线重连都返回明确错误或恢复状态。
- 多浏览器/多账号同房间时，入座、离座、开局、连续多手、房主移交、结算、钱包余额和牌谱能够在不依赖固定轮询的情况下更新。
- 代表性移动端、平板和桌面视口中，牌桌核心区域无阻断遮挡且所有可用按钮可命中。
- 规则测试页仍能独立创建测试牌局、调试设牌、查询历史和只读回放。
