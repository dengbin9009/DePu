# 任务清单：德州扑克完整牌局模拟器

**输入**: `/specs/001-poker-simulator/` 下的规格、计划、研究记录、数据模型和 API 契约。

**前置文档**: `plan.md`、`spec.md`、`research.md`、`data-model.md`、`contracts/openapi.yaml`、`quickstart.md`。

**测试要求**: 规则类任务必须先写失败测试，再实现。短牌 `ante + buttonBlind`、短筹码强制投入、创建校验、只读回放边界、调试牌锁定与随机补足、下注顺序属于高风险规则路径。

## Phase 1: 项目初始化与文档基线

**目的**: 确认现有 Vue + Go + SQLite 工程可承载新版计划。

- [x] T001 确认并保留 `backend/`、`frontend/`、`data/` 目录结构。
- [x] T002 校验 Go 模块和入口命令在 `backend/go.mod` 与 `backend/cmd/depu-server/main.go`。
- [x] T003 校验 Vue + TypeScript + Vite 脚本在 `frontend/package.json` 与 `frontend/vite.config.ts`。
- [x] T004 [P] 校验本地产物忽略规则覆盖 SQLite、Go、Node 输出在 `.gitignore`。
- [x] T005 [P] 确认 Spec Kit 当前计划引用在 `AGENTS.md`。

---

## Phase 2: 基础领域模型、契约与存储

**目的**: 完成所有用户故事共享的规则集、下注结构、牌局状态、错误响应和持久化模型。

**关键约束**: 本阶段阻塞所有用户故事；任何故事实现前必须能表达 `BettingStructure`、`debugLocked`、`isReplay`、结构化 `stateSummary`、`potAwards` 和短筹码实际投入。

- [x] T006 [P] 在 `backend/internal/rules/rules.go` 定义 `RuleSet`、`BettingStructure`、长牌/短牌牌组和短牌牌型排序配置。
- [x] T007 [P] 在 `backend/internal/game/game.go` 更新 `Game`、`Seat`、`Action`、`DebugCardAssignment`、`ReplayRequest` 领域结构。
- [x] T008 [P] 在 `backend/internal/pot/pot.go` 确认底池模型基于实际 `handCommitted` 拆分主池和边池。
- [x] T009 在 `backend/internal/storage/storage.go` 更新 SQLite schema 和快照序列化，保存 `bettingStructure`、`debugLocked`、`isReplay`、`stateSummary` 和 `potAwards`。
- [x] T010 在 `backend/internal/api/server.go` 更新请求/响应 DTO，使实现匹配 `specs/001-poker-simulator/contracts/openapi.yaml`。
- [x] T011 [P] 在 `frontend/src/types/game.ts` 更新前端类型，加入 `BettingStructure`、`RuleSet.bettingStructures`、`GameSnapshot.isReplay`、`GameSnapshot.debugLocked`、结构化 `ActionLog.stateSummary` 和 `ShowdownResult.potAwards`。
- [x] T012 [P] 在 `frontend/src/api/client.ts` 更新创建牌局、调试牌、回放、历史和规则集 API 类型。
- [x] T013 在 `backend/internal/api/server.go` 统一错误响应格式，支持 `invalid_seat`、`invalid_button`、`invalid_player_name`、`invalid_betting_structure`、`invalid_action`、`version_conflict`、`replay_out_of_range`、`debug_locked`、`invalid_card`、`duplicate_card`、`storage_error`、`not_found` 错误码。

**检查点**: 前后端共享模型能表达下注结构、短筹码实际投入、调试锁定、只读回放和历史字段。

---

## Phase 3: 用户故事 1 - 创建并推进完整牌局 (P1)

**目标**: 操作者可以创建长牌或短始牌局，设置下注结构和对应金额，并按后端合法动作推进。

**独立测试**: 创建 4 人长牌 `blinds` 随机发牌牌局、4 人短牌 `ante` 牌局和包含短筹码的牌局，验证每名玩家 2 张手牌、按钮位、强制下注、当前行动者、当前最高投入、全下状态和合法动作正确。

### 测试

- [x] T014 [P] [US1] 在 `backend/internal/api/server_test.go` 编写 `POST /api/games` 契约测试，覆盖 `bettingStructure`、连续座位、按钮位、唯一非空玩家名和非法金额。
- [x] T015 [P] [US1] 在 `backend/internal/game/blinds_test.go` 编写长牌 `blinds` 初始化、随机洗牌发 2 张手牌、`smallBlind < bigBlind` 和单挑行动顺序测试。
- [x] T016 [P] [US1] 在 `backend/internal/game/ante_test.go` 编写短牌 `ante + buttonBlind` 初始化、`streetCommitted`、`currentBet` 和翻牌前首个行动者测试。
- [x] T017 [P] [US1] 在 `backend/internal/game/forced_bet_allin_test.go` 编写小盲、大盲、`ante`、`buttonBlind` 短筹码强制投入全下测试。
- [x] T018 [P] [US1] 在 `backend/internal/game/action_flow_test.go` 编写跟注、加注、不足额全下不重开加注权、非法动作不改变状态、街道推进，以及 2 人和 10 人从创建到结算的完整生命周期测试。

### 实现

- [x] T019 [US1] 在 `backend/internal/rules/rules.go` 实现下注结构校验：长牌仅 `blinds`，短牌允许 `blinds` 或 `ante`，金额必须符合规格。
- [x] T020 [US1] 在 `backend/internal/game/game.go` 实现创建牌局座位校验：`seatNo` 连续、`buttonSeat` 存在、玩家名非空且唯一、筹码为正整数。
- [x] T021 [US1] 在 `backend/internal/game/game.go` 实现随机模式洗牌、每名在局玩家发 2 张手牌、`blinds` 与短牌 `ante + buttonBlind` 的强制下注初始化。
- [x] T022 [US1] 在 `backend/internal/game/game.go` 实现短筹码强制投入按剩余筹码全下，并记录实际 `streetCommitted`、`handCommitted` 和 `status=all_in`。
- [x] T023 [US1] 在 `backend/internal/game/game.go` 实现短牌 `ante` 结构下的跟注、最小加注、行动权推进和街道切换。
- [x] T024 [US1] 在 `backend/internal/api/server.go` 实现新版 `CreateGameRequest.bettingStructure`、创建校验错误和 `GameSnapshot.bettingStructure`。
- [x] T025 [US1] 在 `frontend/src/App.vue` 实现创建牌局时选择 `blinds` 或短牌 `ante` 结构及金额输入。
- [x] T026 [US1] 在 `frontend/src/App.vue` 实现座位、按钮位、玩家名和下注金额的前端输入约束与错误展示。
- [x] T027 [US1] 在 `frontend/src/App.vue` 展示当前下注结构、`ante`、`buttonBlind`、短筹码全下状态、当前行动者和合法动作。

**检查点**: US1 可独立演示：长牌 `blinds`、短牌 `ante` 和短筹码强制投入都能创建并推进至少一轮下注。

---

## Phase 4: 用户故事 2 - 规则判定与摊牌结算 (P1)

**目标**: 系统能正确评估牌型、边池、平分、短牌牌型排序和包含实际强制投入的底池。

**独立测试**: 构造短牌 `ante` 多人全下局面，包含短筹码强制投入，验证 `ante`、`buttonBlind` 和实际全下金额进入投入额、底池和摊牌分配。

### 测试

- [x] T028 [P] [US2] 在 `backend/internal/handeval/evaluator_test.go` 保持并补充最佳五张牌、长牌 A-2-3-4-5、短牌 A-6-7-8-9、同花大于葫芦测试。
- [x] T029 [P] [US2] 在 `backend/internal/pot/ante_pot_test.go` 编写 `ante + buttonBlind` 与短筹码实际投入参与主池和边池拆分测试。
- [x] T030 [P] [US2] 在 `backend/internal/game/showdown_ante_test.go` 编写短牌 `ante` 摊牌结算、弃牌胜出和平分余数测试。
- [x] T031 [P] [US2] 在 `backend/internal/game/allin_showdown_test.go` 编写所有未弃牌玩家全下后自动发完公共牌并进入摊牌测试。

### 实现

- [x] T032 [US2] 在 `backend/internal/pot/pot.go` 确保盲注、`ante`、`buttonBlind` 和短筹码全下通过实际 `handCommitted` 正确进入主池/边池。
- [x] T033 [US2] 在 `backend/internal/game/game.go` 确保短牌 `ante` 局面摊牌结算、弃牌胜出和余数分配正确。
- [x] T034 [US2] 在 `backend/internal/handeval/evaluator.go` 确保短牌牌型排序和 A-6-7-8-9 顺子不被下注结构影响。
- [x] T035 [US2] 在 `backend/internal/game/game.go` 实现所有未弃牌玩家全下时自动发完剩余公共牌并结算。
- [x] T036 [US2] 在 `frontend/src/App.vue` 展示摊牌结果、最佳牌型、各底池 `potAwards`、短牌规则提示和余数分配。

**检查点**: US2 可独立验证：短牌 `ante` 与短筹码局面的底池和摊牌结果与规则一致。

---

## Phase 5: 用户故事 3 - 调试模式指定牌 (P2)

**目标**: 操作者只能在首个玩家动作前指定全部或部分牌；系统拒绝重复牌、非法牌和锁定后的改牌，并随机补足未指定牌。

**独立测试**: 创建短牌调试牌局，首个玩家动作前指定部分手牌和公共牌；已指定牌固定，未指定牌由后端从剩余合法牌堆随机补足；提交任一玩家动作后再次改牌被拒绝。

### 测试

- [x] T037 [P] [US3] 在 `backend/internal/api/debug_cards_test.go` 编写重复牌、短牌非法点数、超过公共牌数量和合法调试牌设置测试。
- [x] T038 [P] [US3] 在 `backend/internal/game/debug_fill_test.go` 编写部分指定手牌和公共牌后从剩余合法牌堆随机补足测试。
- [x] T039 [P] [US3] 在 `backend/internal/api/debug_lock_test.go` 编写首个玩家动作后 `POST /api/games/{gameId}/debug/cards` 被拒绝测试。

### 实现

- [x] T040 [US3] 在 `backend/internal/game/game.go` 实现调试指定牌校验，拒绝重复牌、非法牌和超过阶段数量的公共牌。
- [x] T041 [US3] 在 `backend/internal/game/game.go` 实现已指定牌从牌堆移除，并让未指定手牌和后续公共牌从剩余合法牌堆随机补足。
- [x] T042 [US3] 在 `backend/internal/game/game.go` 实现首个玩家动作后和后续街道的 `debugLocked` 状态判断。
- [x] T043 [US3] 在 `backend/internal/api/server.go` 实现调试牌锁定和非法调试牌错误响应，包含错误码和字段信息。
- [x] T044 [US3] 在 `frontend/src/App.vue` 实现部分指定手牌/公共牌的调试牌面板、锁定禁用状态和后端错误展示。

**检查点**: US3 可独立验证：调试模式可固定关键牌并随机补足剩余牌，但不会破坏已开始牌局的行动历史。

---

## Phase 6: 用户故事 4 - 保存行动历史并只读回放 (P2)

**目标**: 牌局可恢复、可查看历史，并回放到任意有效动作后的只读快照。

**独立测试**: 完成一手牌后读取历史并回放到 `toSeq=0` 和指定动作节点，返回 `isReplay=true`，且不改变当前权威状态；越界 `toSeq` 返回错误。

### 测试

- [x] T045 [P] [US4] 在 `backend/internal/storage/storage_test.go` 编写包含 `bettingStructure`、`debugLocked`、`isReplay`、结构化 `stateSummary` 和 `potAwards` 的快照保存/读取测试，并覆盖创建牌局、提交动作、调试设牌、结算写入的事务失败不报告成功且不留下部分成功状态。
- [x] T046 [P] [US4] 在 `backend/internal/api/replay_readonly_test.go` 编写回放返回只读快照且不改变权威状态测试。
- [x] T047 [P] [US4] 在 `backend/internal/api/replay_bounds_test.go` 编写 `toSeq=0` 返回初始快照、`toSeq` 越界返回 `replay_out_of_range` 测试。
- [x] T048 [P] [US4] 在 `backend/internal/api/history_test.go` 编写行动历史包含 `forced_bet`、`debug_set_cards`、结构化 `stateSummary` 和系统动作测试。

### 实现

- [x] T049 [US4] 在 `backend/internal/storage/storage.go` 用 SQLite 原子事务保存创建牌局、提交动作、调试设牌、结算结果、行动日志、下注结构、调试锁定标记、摊牌结果和回放所需快照摘要；存储失败返回 `storage_error` 且不报告成功。
- [x] T050 [US4] 在 `backend/internal/api/server.go` 实现 `GET /api/games/{gameId}/history` 返回结构化 `ActionLog.stateSummary`、`payload` 和系统动作。
- [x] T051 [US4] 在 `backend/internal/api/server.go` 实现 `POST /api/games/{gameId}/replay` 的 `toSeq=0`、有效动作节点、结算后节点和越界错误处理。
- [x] T052 [US4] 在 `backend/internal/api/server.go` 拒绝基于只读回放快照或旧版本提交动作。
- [x] T053 [US4] 在 `frontend/src/App.vue` 展示行动历史、只读回放标记、手动动作节点切换、节点切换短暂过渡和回放越界错误；避免回放快照覆盖当前权威牌局状态，且不提供整手牌自动播放时间轴入口。

**检查点**: US4 可独立验证：回放能复盘初始和历史节点，但不能产生历史分支或静默修正越界请求。

---

## Phase 7: 用户故事 5 - 可视化牌桌与扑克动画 (P2)

**目标**: 操作者能在第一屏看到手机优先牌桌配件、玩家座位、所有玩家手牌、翻牌圈及之后的当前最佳牌型、公共牌、筹码/底池、当前行动者和动作控制；主视角玩家固定在底部专区，其余座位显示小手牌并避开底部主手牌区；通过短暂动画理解发牌、公共牌、行动切换、底池变化和回放节点切换。

**独立测试**: 创建 4 人牌局并推进到翻牌圈，在 1280px 宽桌面视口中无需滚动即可看到完整牌桌、所有玩家手牌、当前最佳牌型和控制区；翻牌前不显示当前牌型，翻牌圈及之后未弃牌玩家显示由后端快照提供的当前最佳牌型；主视角底部专区的文字、按钮和大手牌不被其他座位遮挡，其余 3 名玩家座位位于上方和左右侧且显示两张小手牌；创建 2 人、6 人、9 人和 10 人牌局时座位布局保持可读，6 到 10 人在移动窄屏允许纵向滚动但不得关键重叠；公共牌出现和回放节点切换动画在 1 秒内完成，行动者高亮明确。

### 测试

- [x] T054 [P] [US5] 在 `frontend/src/pokerVisuals.test.ts` 编写卡牌解析、2 到 10 人座位位置、主视角以外座位避开底部专区、状态标签和 `replayTransition` 的视觉辅助函数测试。
- [x] T055 [P] [US5] 在 `frontend/src/App.visual.test.ts` 编写牌桌关键区域存在性测试，覆盖 2/4/6/9/10 人牌桌、座位小手牌、主玩家底部手牌区、公共牌、底池、行动控制和历史/调试辅助区。
- [x] T056 [P] [US5] 在 `frontend/src/style.visual.test.ts` 编写样式契约测试，断言存在 1280px 桌面、移动窄屏、座位小手牌、主玩家状态条、6 到 10 人紧凑/纵向滚动、`prefers-reduced-motion` 降级规则，以及翻牌/回放节点切换动画时长不超过 1 秒。
- [x] T056a [P] [US5] 在 `backend/internal/api/server_test.go` 编写 `GameSnapshot.seats[].currentHand` 展示字段测试，覆盖翻牌前为空、翻牌圈及之后返回最佳牌型。
- [x] T056b [P] [US5] 在 `backend/internal/api/server_test.go` 和 `frontend/src/api/client.test.ts` 编写下注金额边界与提交金额测试，覆盖 `GameSnapshot.currentBet`、`GameSnapshot.minRaise` 和滑轨选中金额随动作提交。
- [x] T056c [P] [US5] 在 `frontend/src/bettingControls.test.ts` 编写下注滑轨范围测试，覆盖普通加注、不足额全下、手动输入钳制和最小/全下快捷按钮。

### 实现

- [x] T057 [US5] 在 `frontend/src/pokerVisuals.ts` 实现牌面解析、花色颜色、2 到 10 人座位位置、非主视角座位上/左右布局、状态标签、动画阶段和手动回放节点切换过渡辅助函数。
- [x] T058 [US5] 在 `frontend/src/App.vue` 将主区域改造为手机优先牌桌布局，突出主玩家底部手牌区、非主玩家座位小手牌、公共牌、底池、当前行动者和行动控制。
- [x] T058a [US5] 在 `backend/internal/api/server.go` 为座位快照派生 `currentHand` 字段，复用后端牌型评估器，翻牌前和弃牌座位返回空值。
- [x] T058b [US5] 在 `frontend/src/App.vue` 和 `frontend/src/style.css` 显示当前牌型标签，主玩家在底部状态条展示，非主玩家在座位卡片内展示。
- [x] T058c [US5] 在 `backend/internal/api/server.go`、`frontend/src/types/game.ts` 和 OpenAPI 契约中暴露 `currentBet` 与 `minRaise`，供前端下注滑轨计算边界。
- [x] T058d [US5] 在 `frontend/src/bettingControls.ts`、`frontend/src/App.vue` 和 `frontend/src/style.css` 实现下注/加注滑轨、数字输入、最小/半池/底池/全下快捷按钮和提交金额展示。
- [x] T059 [US5] 在 `frontend/src/App.vue` 保留历史、调试发牌和规则集说明作为辅助面板，不得抢占牌桌主视觉。
- [x] T060 [US5] 在 `frontend/src/style.css` 实现真实牌桌风格配件：桌面、座位、卡牌、筹码、底池、按钮位/行动者标识和辅助面板。
- [x] T061 [US5] 在 `frontend/src/style.css` 实现发牌/翻牌、行动者切换、底池/筹码变化和手动回放节点切换动画，并支持浏览器减少动态效果偏好时降级为静态状态。
- [x] T062 [US5] 在 `frontend/src/style.css` 实现 2 到 10 人桌面和移动响应式布局，确保 4 人桌在 1280px 第一屏完整可见，非主玩家座位不进入底部主玩家专区，6 到 10 人桌可紧凑或纵向滚动，且玩家名称、筹码、当前投入、牌面、底池金额和行动按钮不被卡牌、座位、底池或控制区遮挡。

**检查点**: US5 可独立验证：当前实现从文字型界面升级为可演示的德州扑克桌面体验，且不改变后端权威规则。

---

## Phase 8: 用户故事 6 - 管理长牌和短牌规则集 (P3)

**目标**: 创建牌局时清楚展示长牌/短牌差异和可用下注结构。

**独立测试**: 调用规则集接口，检查长牌仅 `blinds`，短牌包含 `blinds` 与 `ante`，前端创建页随规则集切换牌组、牌型排序和下注结构选项。

### 测试

- [x] T063 [P] [US6] 在 `backend/internal/api/rulesets_test.go` 编写 `GET /api/rulesets` 返回 `bettingStructures`、默认下注结构、牌组和牌型排序测试。
- [x] T064 [P] [US6] 在 `frontend/src/types/game.ts` 增加类型层面的长牌/短牌下注结构示例并通过前端构建验证。

### 实现

- [x] T065 [US6] 在 `backend/internal/rules/rules.go` 实现长牌 52 张、短牌 36 张、短牌 A-6-7-8-9 和默认牌型排序配置。
- [x] T066 [US6] 在 `backend/internal/api/server.go` 返回规则集的可选下注结构、默认结构、牌组和牌型排序。
- [x] T067 [US6] 在 `frontend/src/App.vue` 展示长牌/短牌牌组、牌型排序、`blinds`/`ante` 差异和默认下注结构。
- [x] T068 [US6] 在 `frontend/src/style.css` 打磨下注结构选择、只读回放标记、短筹码全下状态和调试锁定状态样式。

**检查点**: US6 可独立验证：用户能在创建前理解当前规则集和下注结构。

---

## Phase 9: 打磨与验收

**目的**: 文档、契约、格式化和端到端验证。

- [x] T069 [P] 更新实际启动命令和 OpenAPI 0.3.0 创建请求示例在 `README.md`。
- [x] T070 [P] 校验实现响应与 `specs/001-poker-simulator/contracts/openapi.yaml` 一致。
- [x] T071 [P] 回看 `specs/001-poker-simulator/checklists/requirements.md`，勾选已由规格、模型或契约解决的需求质量项。
- [x] T072 [P] 在 `backend/tests/integration/performance_test.go` 编写创建牌局、动作提交和读取历史 p95 小于 200ms、200 动作内回放小于 1 秒的本机性能基准测试。
- [x] T073 [P] 在 `frontend/tests/visual/table.spec.ts` 增加牌桌视觉验收记录，覆盖 2/4/6/9/10 人、1280px 桌面、移动视口、6 到 10 人纵向滚动和手动回放节点切换截图。
- [x] T074 运行后端格式化和测试：`gofmt` 与 `go test ./...` 于 `backend/`。
- [x] T075 运行前端安装、类型检查和构建：`npm install`、`npm test` 与 `npm run build` 于 `frontend/`。
- [x] T076 执行 `specs/001-poker-simulator/quickstart.md` 中的完整手动验收流程。

---

## 依赖顺序

### Phase 依赖

- Phase 1 必须先完成。
- Phase 2 阻塞所有用户故事。
- US1 和 US2 构成 MVP，必须优先完成。
- US3 和 US4 可在基础状态机完成后并行推进。
- US5 可在后端快照、合法动作和前端类型稳定后推进。
- US6 可在规则集模型和创建牌局 API 稳定后推进。
- Phase 9 依赖目标用户故事完成。

### 用户故事依赖

- **US1 (P1)**: 依赖 Phase 2；不依赖其他用户故事。
- **US2 (P1)**: 依赖 Phase 2；与 US1 共享下注结构、全下状态和底池计算。
- **US3 (P2)**: 依赖 Phase 2 和 US1 的动作日志/行动状态。
- **US4 (P2)**: 依赖 Phase 2 和状态快照保存。
- **US5 (P2)**: 依赖 Phase 2 的前端类型、US1 的基础牌局快照和合法动作展示。
- **US6 (P3)**: 依赖 Phase 2 的规则集与下注结构模型。

## 并行执行示例

### US1 并行任务

```text
T014: server_test.go 创建牌局契约测试
T015: blinds_test.go 长牌 blinds 行动顺序测试
T016: ante_test.go 短牌 ante 初始化测试
T017: forced_bet_allin_test.go 短筹码强制投入测试
T018: action_flow_test.go 动作推进与非法动作不改状态测试
```

### US2 并行任务

```text
T028: evaluator_test.go 最佳五张牌与长短牌顺子测试
T029: ante_pot_test.go ante 与短筹码底池拆分测试
T030: showdown_ante_test.go ante 摊牌测试
T031: allin_showdown_test.go 全下自动摊牌测试
```

### US3/US4 并行任务

```text
T037: debug_cards_test.go 调试牌校验测试
T038: debug_fill_test.go 部分指定牌随机补足测试
T039: debug_lock_test.go 调试锁定测试
T045: storage_test.go 快照保存测试
T046: replay_readonly_test.go 只读回放测试
T047: replay_bounds_test.go 回放边界测试
```

### US5 并行任务

```text
T054: pokerVisuals.test.ts 2 到 10 人座位和回放过渡视觉辅助函数测试
T055: App.visual.test.ts 2/4/6/9/10 人牌桌关键区域存在性测试
T056: style.visual.test.ts 响应式、紧凑/滚动和减少动态效果样式契约测试
```

## MVP 策略

1. 完成 Phase 1 和 Phase 2，建立新版下注结构、API DTO、存储快照和前端类型。
2. 完成 US1，确保长牌 `blinds`、短牌 `ante`、创建校验和短筹码全下都可创建并推进。
3. 完成 US2，确保短牌 `ante` 和短筹码不破坏底池和摊牌。
4. 停下验证 MVP：`go test ./...` 通过，前端能创建并推进短牌 `ante` 短筹码牌局。
5. 再加入 US3 调试锁定/随机补足、US4 只读回放/越界错误/手动节点切换、US5 2 到 10 人牌桌视觉与扑克动画、US6 规则集展示。

## 格式校验

- 所有任务使用 `- [ ] T###` 格式。
- 所有用户故事任务包含 `[US#]` 标签。
- 所有任务包含明确文件路径。
- `[P]` 仅用于不同文件且可并行的任务。
