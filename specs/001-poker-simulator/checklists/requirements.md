# 需求质量检查清单：德州扑克完整牌局模拟器

**Purpose**: 检查当前规格、计划、数据模型、API 契约和任务是否足以指导实现；本清单检查需求写作质量，不检查代码行为。
**Created**: 2026-06-25
**Feature**: [spec.md](/Users/dengbin/Code/github/DePu/specs/001-poker-simulator/spec.md)

**Note**: 本清单由 `/speckit-checklist` 基于当前 feature 文档生成，适合作为 PR 审阅和实现前需求门禁使用。

## Requirement Completeness

- [x] CHK001 是否分别定义了长牌 `blinds`、短牌 `blinds`、短牌 `ante + buttonBlind` 三类创建参数、强制投入和翻牌前行动顺序？ [Completeness, Spec §FR-002, Data Model §BettingStructure]
- [x] CHK002 是否完整定义了短牌 `ante` 如何进入 `streetCommitted`、`handCommitted`、`currentBet`、主池和边池？ [Completeness, Spec §FR-002, Data Model §Seat, Data Model §Pot]
- [x] CHK003 是否为只读回放定义了读取入口、返回快照、禁止继续提交动作和不改变权威状态的完整需求？ [Completeness, Spec §FR-016, Contract §/api/games/{gameId}/replay]
- [x] CHK004 是否为调试指定牌定义了可编辑窗口、锁定条件、重复牌、非法牌、阶段约束和已结算牌局约束？ [Completeness, Spec §FR-004, Spec §边界情况]
- [x] CHK005 是否定义了规则集接口需要暴露可选下注结构、默认下注结构和长短牌关键差异？ [Completeness, Spec §用户故事6, Contract §RuleSet]
- [x] CHK006 是否定义了牌桌视觉体验需要包含牌桌面、玩家座位、手牌、公共牌、筹码/底池、按钮位或当前行动者标识和动作控制？ [Completeness, Spec §用户故事5, Spec §FR-018]
- [x] CHK007 是否定义了发牌、公共牌出现、行动者切换和底池/筹码变化的动画反馈，以及动画不得阻塞动作或遮挡关键信息？ [Completeness, Spec §FR-019, Data Model §TableVisualState]
- [x] CHK008 是否定义了行动历史中系统动作、强制下注、发牌、调试设牌和结算动作的记录要求？ [Completeness, Data Model §Action, Contract §ActionLog]

## Requirement Clarity

- [x] CHK009 `ante` 计入 live bet 的含义是否被明确映射为可实现字段和补齐金额规则，而不依赖口头解释？ [Clarity, Spec §Clarifications, Data Model §BettingStructure]
- [x] CHK010 `buttonBlind` 与 `bigBlind`、`smallBlind` 的适用范围是否清楚区分，避免短牌 ante 与 blinds 结构混用？ [Clarity, Spec §FR-002, Contract §BettingStructure]
- [x] CHK011 “首个玩家动作前”是否用动作日志、阶段和玩家动作类型清楚界定，避免系统 `forced_bet` 或 `deal` 动作误触发调试锁定？ [Clarity, Spec §FR-004, Data Model §Action]
- [x] CHK012 “只读快照”“权威状态”“旧版本提交”的关系是否清楚定义到足以指导 API 错误处理？ [Clarity, Spec §FR-016, Contract §SubmitActionRequest]
- [x] CHK013 牌型排序、A 低顺子和短牌特殊顺子是否分别以规则集维度表达，避免被下注结构或前端展示逻辑影响？ [Clarity, Spec §FR-010, Spec §FR-011, Spec §FR-012]

## Requirement Consistency

- [x] CHK014 规格、数据模型和 OpenAPI 是否都要求 `GameSnapshot` 包含当前 `bettingStructure` 与 `isReplay`？ [Consistency, Spec §关键实体, Data Model §Game, Contract §GameSnapshot]
- [x] CHK015 `RuleSet.bettingStructures` 在规格、数据模型、契约和任务中是否保持一致：长牌仅 `blinds`，短牌包含 `blinds` 与 `ante`？ [Consistency, Data Model §RuleSet, Contract §RuleSet, Tasks §T006]
- [x] CHK016 调试牌锁定规则在规格、OpenAPI 描述和任务拆分中是否没有冲突或遗漏？ [Consistency, Spec §用户故事3, Contract §/api/games/{gameId}/debug/cards, Tasks §T037-T044]
- [x] CHK017 只读回放规则在规格、数据模型、OpenAPI 和任务中是否都禁止从历史节点分叉继续行动？ [Consistency, Spec §FR-016, Data Model §状态转换, Tasks §T045-T053]
- [x] CHK018 前端职责是否在计划和任务中保持为展示后端权威规则与合法动作，而不是推断下注结构或规则结果？ [Consistency, Plan §项目结构, Tasks §T025-T027, Tasks §T054-T062]
- [x] CHK019 牌桌视觉状态是否在规格、计划、数据模型和任务中一致声明为前端派生展示状态，而不是规则权威？ [Consistency, Spec §关键实体, Plan §项目结构, Data Model §TableVisualState, Tasks §T057-T062]

## Acceptance Criteria Quality

- [x] CHK020 每个用户故事的独立测试是否包含可客观判定的输入、动作和结果，且覆盖该故事最高风险规则？ [Acceptance Criteria, Spec §用户场景与测试]
- [x] CHK021 成功标准中的时间目标、恢复目标和动画完成目标是否有明确测量对象、测量边界和通过条件？ [Measurability, Spec §成功标准, Plan §性能目标]
- [x] CHK022 “后端规则测试覆盖核心场景”是否具体列出必须覆盖的规则集合和失败边界，而不只描述覆盖存在？ [Measurability, Spec §SC-003, Tasks §测试要求]
- [x] CHK023 API 错误响应是否为主要失败场景定义了足够稳定的错误码、字段和消息语义？ [Acceptance Criteria, Contract §ErrorResponse, Gap]
- [x] CHK024 牌桌视觉验收是否包含桌面 1280px、移动窄屏、行动者高亮、公共牌动画完成时间和关键文字可读性？ [Acceptance Criteria, Spec §用户故事5, Spec §SC-006, Spec §SC-007, Quickstart §验证牌桌视觉和扑克动画]

## Scenario Coverage

- [x] CHK025 是否覆盖长牌多人、长牌单挑、短牌 blinds、短牌 ante 四类初始化和行动顺序场景？ [Coverage, Spec §边界情况, Tasks §T014-T018]
- [x] CHK026 是否覆盖多人全下、边池、平分余数、弃牌胜出和所有未弃牌玩家全下自动发牌这些结算路径？ [Coverage, Spec §用户故事2, Spec §边界情况]
- [x] CHK027 是否覆盖随机发牌与调试指定牌两种发牌模式在同一规则集校验体系下的需求？ [Coverage, Spec §FR-003, Spec §FR-004]
- [x] CHK028 是否覆盖存储失败、状态版本冲突、只读回放提交动作和调试锁定失败这些异常路径？ [Coverage, Spec §用户故事4, Contract §ErrorResponse]
- [x] CHK029 是否覆盖动画失败、减少动态效果偏好和窄屏布局降级后仍可读的展示路径？ [Coverage, Spec §边界情况, Data Model §TableVisualState]
- [x] CHK030 是否明确 v1 排除账户、真钱、多人同步、排行榜、AI 建议和 GTO 求解后，对数据模型与 API 不产生隐含需求？ [Coverage, Spec §FR-021, Research §单机本地边界]

## Edge Case Coverage

- [x] CHK031 是否定义玩家初始筹码不足以支付 `smallBlind`、`bigBlind`、`ante` 或 `buttonBlind` 时的创建失败或全下规则？ [Gap, Spec §FR-002]
- [x] CHK032 是否定义 `ante` 或盲注金额非法组合的校验边界，例如 0、负数、`buttonBlind` 小于等于 `ante` 时是否允许？ [Gap, Contract §BettingStructure]
- [x] CHK033 是否定义非连续座位、按钮位不在座位列表、重复座位号和重复玩家名的处理要求？ [Gap, Contract §CreateGameRequest]
- [x] CHK034 是否定义回放目标动作序号不存在、超过历史长度或为结算后节点时的响应语义？ [Gap, Contract §/api/games/{gameId}/replay]
- [x] CHK035 是否定义行动历史摘要需要包含哪些状态字段，才能支撑可解释回放而不依赖前端猜测？ [Gap, Data Model §Action, Contract §ActionLog]

## Non-Functional Requirements

- [x] CHK036 本地 SQLite 存储位置、数据保留策略、失败原子性和恢复边界是否已明确到可实现？ [Non-Functional, Spec §假设, Spec §用户故事4]
- [x] CHK037 性能目标是否覆盖创建牌局、提交动作、回放快照和读取历史这些关键路径，而不只覆盖部分操作？ [Non-Functional, Plan §性能目标]
- [x] CHK038 前端可视化是否定义最小可用展示要求，包括牌桌、玩家状态、下注结构、错误反馈、只读回放、调试锁定状态、牌桌配件和扑克动画？ [Non-Functional, Spec §FR-017, Spec §FR-018, Spec §FR-019, Tasks §T054-T062]
- [x] CHK039 中文正式文档和英文代码/API 标识的边界是否在规格、契约和协作说明中保持一致？ [Consistency, AGENTS.md, Contract §info.description]

## Dependencies & Assumptions

- [x] CHK040 Go 1.22+、Node.js 20+、SQLite 本地文件这些环境假设是否与快速开始、计划和任务验证命令一致？ [Assumption, Plan §技术上下文, Quickstart §前置条件]
- [x] CHK041 是否明确规则权威在后端，前端只消费快照、合法动作和错误信息的架构假设？ [Assumption, Plan §约束, Plan §项目结构]
- [x] CHK042 是否明确 v1 只处理整数筹码后，对底池余数、全下金额和金额输入校验的需求影响？ [Assumption, Spec §假设, Spec §FR-014]

## Ambiguities & Conflicts

- [x] CHK043 是否存在 `Action.stateSummary` 与 `ActionLog.summary` 字段命名或内容范围不一致，需要在契约中统一？ [Ambiguity, Data Model §Action, Contract §ActionLog]
- [x] CHK044 是否存在 `ShowdownResult.potAwards` 与 OpenAPI `awards` 字段命名差异，需要在数据模型或契约中统一？ [Ambiguity, Data Model §ShowdownResult, Contract §ShowdownResult]
- [x] CHK045 是否存在 `BettingStructure` 的规则字段 `preflopFirstActorRule`、`postflopFirstActorRule` 未在 OpenAPI 暴露的有意设计或遗漏？ [Ambiguity, Data Model §BettingStructure, Contract §BettingStructure]

## Recent Clarification Additions

- [x] CHK046 是否定义了 2 到 10 人牌桌视觉的座位布局范围、4 人桌 1280px 第一屏边界、6 到 10 人紧凑布局或移动端纵向滚动边界？ [Completeness, Spec §FR-020, Plan §规模/范围]
- [x] CHK047 是否定义了回放交互仅支持手动切换动作节点、允许短暂过渡动画、且 v1 排除整手牌自动播放时间轴？ [Completeness, Spec §FR-016, Research §回放只做手动节点切换]
- [x] CHK048 “不得关键重叠”“关键文字可读”“更紧凑布局”是否已通过人数、视口、滚动边界或验收截图要求转化为可执行的需求，而不是仅保留主观形容词？ [Clarity, Spec §用户故事5, Spec §SC-006, Tasks §T073]
- [x] CHK049 `replayTransition` 与普通发牌/行动动画的职责边界是否清楚，避免前端把回放节点切换实现成自动播放时间轴或新规则状态机？ [Clarity, Data Model §TableVisualState, Plan §约束]
- [x] CHK050 2 到 10 人牌桌布局要求是否在规格、计划、数据模型、快速开始和任务中保持一致，且没有仍只验收 4 人桌的残留表述？ [Consistency, Spec §FR-020, Plan §规模/范围, Data Model §TableVisualState, Quickstart §验证牌桌视觉和扑克动画, Tasks §T054-T073]
- [x] CHK051 手动回放节点切换和“不提供整手牌自动播放时间轴”是否在规格、计划、研究记录、数据模型、快速开始和任务中保持一致？ [Consistency, Spec §FR-016, Plan §约束, Research §回放只做手动节点切换, Data Model §TableVisualState, Quickstart §验证核心流程, Tasks §T053]
- [x] CHK052 牌桌视觉验收是否明确覆盖 2/4/6/9/10 人代表人数、1280px 桌面、移动视口、6 到 10 人纵向滚动和手动回放节点切换截图？ [Acceptance Criteria, Tasks §T073, Quickstart §验证牌桌视觉和扑克动画]
- [x] CHK053 是否覆盖只读回放的主路径、越界错误、手动节点切换、过渡动画降级和自动播放时间轴排除这些场景类别？ [Coverage, Spec §用户故事4, Spec §FR-016, Data Model §TableVisualState]
- [x] CHK054 是否覆盖 2 人、4 人、6 人、9 人和 10 人这些代表性座位数量，以及桌面和移动两类视口要求？ [Coverage, Spec §用户故事5, Quickstart §验证牌桌视觉和扑克动画, Tasks §T055, Tasks §T073]
- [x] CHK055 是否定义 6 到 10 人移动端纵向滚动时玩家名称、筹码、当前投入、牌面、底池金额和行动按钮的最低可读边界，避免“允许滚动”掩盖关键内容不可达？ [Edge Case, Spec §FR-020, Quickstart §验证牌桌视觉和扑克动画]
- [x] CHK056 前端动画性能要求是否同时覆盖翻牌动画和手动回放节点切换过渡，并明确二者不得阻塞合法动作提交或状态阅读？ [Non-Functional, Plan §性能目标, Spec §FR-019, Spec §FR-016]
- [x] CHK057 是否明确 2 到 10 人视觉布局依赖前端派生状态和后端连续座位假设，而不是要求后端新增视觉布局 API？ [Assumption, Data Model §TableVisualState, Plan §项目结构, Spec §FR-002]
- [x] CHK058 `TableVisualState.replayTransition` 是否在规格关键实体、数据模型和任务中保持同名概念，避免回放过渡状态命名不一致？ [Ambiguity, Data Model §TableVisualState, Spec §关键实体, Tasks §T054-T061]
- [x] CHK059 是否明确手机优先牌桌中主视角玩家固定在底部专区、其余玩家座位显示小手牌并避开底部专区，避免玩家手牌遮挡文字或出现只有文字没有牌的座位？ [Completeness, Spec §用户故事5, Spec §FR-018, Spec §FR-020, Quickstart §验证牌桌视觉和扑克动画]

## Notes

- 勾选完成项时使用 `[x]`。
- 发现需求缺口时，优先回写到 `spec.md`、`data-model.md` 或 `contracts/openapi.yaml`，再同步 `tasks.md`。
- 本清单不替代后端规则测试、API 集成测试或前端构建验证。
