# Implementation Plan: 多人德州扑克对战 v1

**Branch**: `002-multiplayer-poker-v1` | **Date**: 2026-06-26 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-multiplayer-poker-v1/spec.md`

## Summary

本功能将项目从“本地规则引擎验证工具”扩展为“多人德州扑克对战 v1”的正式产品规划。核心技术路径是：保留现有独立规则测试/回放页面作为规则验证基线，在其旁边新增账号、钱包、房间、正式多人牌局和每手牌结果归档能力。生产默认使用 MySQL，开发/测试允许 SQLite；正式多人牌局复用现有规则引擎，但不将测试页调试特权暴露给正式用户主流程。

## Technical Context

**Language/Version**: Go（后端）、Vue + TypeScript（前端）

**Primary Dependencies**: 现有 Go 规则引擎、现有 HTTP JSON API、Vue 前端、SQLite 驱动、MySQL 驱动、认证/密码散列库

**Storage**: 生产默认 MySQL；开发/测试允许 SQLite

**Testing**: Go `go test`、后端契约/集成测试、前端基础验证与现有视觉/交互测试

**Target Platform**: 本地开发环境与可部署的服务端 Web 应用环境

**Project Type**: 前后端分离 Web application

**Performance Goals**: 优先保证规则正确性、权限正确性、钱包与战绩一致性；正式多人接口应支持房间内多用户轮询式操作，无需实时推送性能目标

**Constraints**: 不接真实支付、不做 WebSocket 实时推送、不做大厅匹配、不破坏现有规则测试页、不得明文存储密码、每手牌结算与钱包更新必须原子一致
**Constraints**: 不接真实支付、不做 WebSocket 实时推送、不做大厅匹配、不破坏现有规则测试页、不得明文存储密码、密码最小长度 8 位、不得在 API 响应中回传密码或散列值、每手牌结算与钱包更新必须原子一致

**Scale/Scope**: v1 面向小规模房间制多人对战；默认 6 人房间，允许显式配置 2-10 人；默认最少 2 人开局；聚焦账号、钱包、建房、轮流操作、每手牌结果与个人战绩

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### 现有宪章审查

当前 `.specify/memory/constitution.md` 明确把“登录、真钱、多房间、多人实时同步”排除在 001 单机版本之外，因此对 002 多人产品规划而言，**单机边界条款不再直接适用为禁止项，而应视为 001 历史基线约束**。本次 feature 通过新建 `specs/002-multiplayer-poker-v1/` 独立规划多人版本，不修改 001 规则引擎基线的目标范围。

### Gate 结果

- **规则正确性优先**: PASS。正式多人牌局继续以后端规则引擎为权威来源。
- **长牌与短牌规则集可配置**: PASS。002 不改变现有规则集设计，正式房间复用已有规则能力。
- **测试优先覆盖高风险规则**: PASS。002 的高风险路径扩展为账号唯一性、鉴权、钱包流水、邀请码加入、当前行动玩家校验和结算原子一致。
- **单机本地边界**: JUSTIFIED EXCEPTION。该条款针对 001 基线；002 明确作为新产品规划引入账号、房间和多人能力，并保留测试页不受破坏。
- **可复盘、可解释、可恢复**: PASS。测试页保留快照与回放；正式多人新增每手牌结果、钱包流水和统计追踪。

### 结论

可继续 Phase 0 与 Phase 1 设计；唯一需要显式记录的是：002 是对产品范围的升级规划，不是对 001 单机边界的隐式篡改。

## Project Structure

### Documentation (this feature)

```text
specs/002-multiplayer-poker-v1/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── depu-server/
├── internal/
│   ├── api/
│   ├── game/
│   ├── handeval/
│   ├── pot/
│   ├── rules/
│   └── storage/
└── tests/
    ├── integration/
    └── ...

frontend/
├── src/
│   ├── api/
│   ├── components/
│   ├── pages/
│   ├── stores/
│   └── types/
└── tests/
```

**Structure Decision**: 保持现有前后端分离结构。后续实现时，正式多人新增的认证、用户、钱包、房间和战绩能力应优先在后端 `internal/` 中形成清晰领域边界，而不是把多人逻辑堆叠到现有单机测试控制器中。前端则在现有 `src/` 下新增正式产品页面，同时保留独立规则测试页入口。

## Phase 0: Research Output

研究结论记录在 [research.md](./research.md)，并已对以下关键问题给出决策：

- 生产默认 MySQL、开发/测试允许 SQLite
- 首版多人采取房主建房 + 邀请码加入 + 多账号轮流操作
- 登录使用用户名 + 密码，昵称独立且全站唯一
- 充值采用确认后直接模拟成功、余额累加、写入流水
- 充值档位由服务端返回，至少 3 个固定档位
- 测试页保留调试与回放特权，正式房间不暴露这些能力
- 每手牌结果必须独立落库，并与钱包变更原子一致
- 金币不足的边界拆分到建房、入座买入和开局前校验三个节点
- 历史与战绩展示使用结算当时的昵称快照

## Phase 1: Design & Contracts Output

- 数据模型: [data-model.md](./data-model.md)
- API 契约: [contracts/openapi.yaml](./contracts/openapi.yaml)
- 快速开始: [quickstart.md](./quickstart.md)

### 设计要点

#### 1. 认证与用户域

- 新增 `User` 与 `UserProfile`，分别承担登录身份和展示资料职责。
- 用户名作为稳定登录标识，全站唯一；昵称作为展示名，全站唯一且允许修改。
- 所有正式多人接口都依赖鉴权上下文。
- 密码安全最低要求包括：最小长度 8 位、安全散列存储、任何 API 响应不返回密码或散列值。

#### 2. 钱包与模拟充值域

- 新增 `Wallet` 和 `WalletTransaction`。
- 钱包余额变化必须有流水支撑，模拟充值语义与正式牌局输赢语义统一落在同一个账本抽象中。
- 不引入真实支付订单与回调状态机。

#### 3. 房间与座位域

- 使用 `Room`、`RoomMember`、`RoomSeat` 表达房主建房和邀请码加入模式。
- 房主负责开局控制；玩家只在加入房间与占座后获得正式牌局参与资格。
- 用户离开房间时需释放成员身份与座位；若房主离开但房间仍有成员，则房主权限转移；空房间关闭且不得继续开局。

#### 4. 正式多人牌局域

- 使用 `GameTable`、`Hand`、`HandParticipant`、`HandResult` 对正式手牌做归档建模。
- 复用现有规则引擎进行发牌、行动合法性、摊牌与分池。
- 在规则引擎外围增加“当前玩家身份必须匹配当前行动席位”的权限约束。

#### 5. 测试页边界

- 测试页继续服务规则验证、调试设牌和只读回放。
- 正式多人主流程不允许直接进入调试设牌、测试历史重放和其他测试特权路径。
- 前端入口也必须分离：测试页是独立入口，正式多人页面不默认暴露测试能力。

## Agent Context Update

`AGENTS.md` 已更新 `<!-- SPECKIT START -->` 和 `<!-- SPECKIT END -->` 之间的计划引用，当前指向 `specs/002-multiplayer-poker-v1/plan.md`。

## Post-Design Constitution Check

- **规则正确性优先**: PASS。设计未将规则判断迁移到前端。
- **长牌与短牌规则集可配置**: PASS。设计只新增正式多人外层域模型，不改变已有规则集语义。
- **测试优先覆盖高风险规则**: PASS。高风险列表已在 `tasks.md` 中前置到测试任务。
- **单机本地边界**: JUSTIFIED EXCEPTION。002 为正式新 feature 规划，边界升级被显式记录，且 001 测试资产保留。
- **可复盘、可解释、可恢复**: PASS。每手牌结果、钱包流水和测试页回放共同保证追溯性。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 宪章中的 001 单机边界限制 | 002 需要正式引入账号、房间、虚拟金币和多人能力 | 继续把多人能力塞进 001 会让范围与宪章描述长期冲突 |
| MySQL + SQLite 双模式 | 正式产品与本地开发/测试需要兼顾 | 仅保留 SQLite 不利于多人产品化规划 |
| 正式多人与测试页并存 | 既要保留规则验证资产，也要建立正式用户主流程 | 废弃测试页或混入正式房间都会提高回归和产品语义风险 |
