# Tasks: 多人德州扑克对战 v1

**Input**: Design documents from `/specs/002-multiplayer-poker-v1/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/openapi.yaml`, `quickstart.md`

**Tests**: 本 feature 明确要求测试优先；账号唯一性、鉴权、钱包流水、邀请码加入、当前行动玩家校验、手牌结算与钱包原子一致必须先写失败测试。

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., `US1`, `US2`)
- Include exact file paths in descriptions

## Path Conventions

- **Web app**: `backend/` and `frontend/`
- **Backend tests**: `backend/internal/...`, `backend/tests/...`
- **Frontend app/tests**: `frontend/src/...`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 准备多人 feature 文档映射、分层入口和后续实现约束。

- [x] T001 复核 `specs/002-multiplayer-poker-v1/plan.md` 与 `AGENTS.md` 中的 feature 引用一致
- [x] T002 [P] 盘点现有规则引擎与测试页可复用边界，记录到 `specs/002-multiplayer-poker-v1/research.md`
- [x] T003 [P] 规划 MySQL/SQLite 运行配置入口并映射到 `backend/cmd/depu-server/main.go`
- [x] T004 规划正式多人 API 与测试页 API 的路由分组边界，落地到 `backend/internal/api/server.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 所有用户故事共享的基础设施，必须先完成。

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 [P] 编写数据库模式切换测试于 `backend/internal/storage/storage_test.go`
- [x] T006 [P] 规划认证中间件与会话注入接口于 `backend/internal/api/server.go`
- [x] T007 [P] 规划统一错误响应扩展，覆盖 `unauthorized`、`forbidden`、`duplicate_username`、`duplicate_nickname`、`insufficient_coins`、`room_not_found`、`invalid_invite_code`、`seat_taken`、`not_room_owner`、`not_your_turn` 于 `backend/internal/api/server.go`
- [x] T008 定义多人正式流程基础存储抽象于 `backend/internal/storage/storage.go`
- [x] T009 规划前端正式多人 API client 扩展入口于 `frontend/src/api/client.ts`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - 注册、登录并维护唯一昵称 (Priority: P1) 🎯 MVP

**Goal**: 用户可以注册、登录、查看资料并维护全站唯一昵称。

**Independent Test**: 注册两个账号并设置不同昵称；重复用户名注册失败，重复昵称修改失败；未登录访问资料接口失败。

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T010 [P] [US1] 编写注册与登录契约测试于 `backend/internal/api/auth_contract_test.go`
- [x] T011 [P] [US1] 编写重复用户名与重复昵称测试于 `backend/internal/api/auth_uniqueness_test.go`
- [x] T012 [P] [US1] 编写未登录访问资料接口测试于 `backend/internal/api/auth_guard_test.go`
- [x] T013 [P] [US1] 编写资料页昵称修改前端交互测试于 `frontend/src/profilePage.test.ts`
- [x] T014 [P] [US1] 编写密码最小长度校验与响应不返回密码字段测试于 `backend/internal/api/auth_security_test.go`

### Implementation for User Story 1

- [x] T015 [P] [US1] 定义 `User` 与 `UserProfile` 存储模型于 `backend/internal/storage/storage.go`
- [x] T016 [P] [US1] 实现密码散列、最小长度校验与响应字段过滤逻辑于 `backend/internal/api/server.go`
- [x] T017 [US1] 实现注册、登录和当前用户接口于 `backend/internal/api/server.go`
- [x] T018 [US1] 实现昵称更新接口与唯一性校验于 `backend/internal/api/server.go`
- [x] T019 [P] [US1] 扩展前端认证与资料 API 类型于 `frontend/src/api/client.ts`
- [x] T020 [US1] 实现资料页与昵称修改交互于 `frontend/src/App.vue`
- [x] T021 [US1] 在 `backend/internal/api/server.go` 统一返回 `duplicate_username`、`duplicate_nickname` 与 `unauthorized`

**Checkpoint**: User Story 1 should be fully functional and independently testable

---

## Phase 4: User Story 2 - 虚拟金币充值与钱包流水 (Priority: P1)

**Goal**: 用户可以执行模拟充值，余额累加并可查看钱包流水。

**Independent Test**: 同一用户连续执行两次模拟充值，余额按档位累加，流水写入两条记录；未登录访问钱包或充值接口失败。

### Tests for User Story 2 ⚠️

- [x] T022 [P] [US2] 编写至少 3 个服务端充值档位与模拟充值契约测试于 `backend/internal/api/wallet_contract_test.go`
- [x] T023 [P] [US2] 编写钱包余额累加与流水写入测试于 `backend/internal/storage/storage_test.go`
- [x] T024 [P] [US2] 编写未登录访问钱包接口测试于 `backend/internal/api/wallet_auth_test.go`
- [x] T025 [P] [US2] 编写前端钱包页交互测试于 `frontend/src/walletPage.test.ts`

### Implementation for User Story 2

- [x] T026 [P] [US2] 定义 `Wallet` 与 `WalletTransaction` 存储模型于 `backend/internal/storage/storage.go`
- [x] T027 [US2] 实现充值档位、钱包余额和流水查询接口于 `backend/internal/api/server.go`
- [x] T028 [US2] 实现模拟充值成功与余额累加逻辑于 `backend/internal/storage/storage.go`
- [x] T029 [P] [US2] 扩展前端钱包 API 类型于 `frontend/src/api/client.ts`
- [x] T030 [US2] 实现充值确认、余额展示和流水列表于 `frontend/src/App.vue`
- [x] T031 [US2] 在 `backend/internal/api/server.go` 统一返回钱包相关 `unauthorized` 错误

**Checkpoint**: User Stories 1 and 2 should both work independently

---

## Phase 5: User Story 3 - 房主建房并通过邀请码组织多人入座 (Priority: P1)

**Goal**: 房主可建房并分享邀请码，其他用户可加入、入座与离座。

**Independent Test**: 房主建房成功并获得邀请码；另外两名用户凭邀请码加入并入座；错误邀请码失败；重复占座失败。

### Tests for User Story 3 ⚠️

- [x] T032 [P] [US3] 编写建房默认 6 人桌、最少 2 人开局与邀请码唯一性测试于 `backend/internal/api/room_create_test.go`
- [x] T033 [P] [US3] 编写通过邀请码加入房间测试于 `backend/internal/api/room_join_test.go`
- [x] T034 [P] [US3] 编写入座/离座、房主离开转移权限与空房关闭测试于 `backend/internal/api/room_seat_test.go`
- [x] T035 [P] [US3] 编写房间页前端交互测试于 `frontend/src/roomPage.test.ts`

### Implementation for User Story 3

- [x] T036 [P] [US3] 定义 `Room`、`RoomMember`、`RoomSeat` 存储模型于 `backend/internal/storage/storage.go`
- [x] T037 [US3] 实现创建房间、邀请码加入和房间详情接口于 `backend/internal/api/server.go`
- [x] T038 [US3] 实现入座、离座、房主离开转移与空房关闭逻辑于 `backend/internal/api/server.go`
- [x] T039 [US3] 在 `backend/internal/api/server.go` 统一返回 `room_not_found`、`invalid_invite_code`、`seat_taken`
- [x] T040 [P] [US3] 扩展前端房间 API 类型于 `frontend/src/api/client.ts`
- [x] T041 [US3] 实现建房、加入房间和入座交互于 `frontend/src/App.vue`

**Checkpoint**: User Stories 1-3 should each be independently testable

---

## Phase 6: User Story 4 - 多真实账号轮流完成一手正式牌局 (Priority: P1)

**Goal**: 正式房间可以开局，且只有当前行动玩家能提交动作。

**Independent Test**: 三个真实账号加入同一房间并开始一手牌；当前行动玩家操作成功，非当前行动玩家被拒绝；房主之外的用户不能发起开局。

### Tests for User Story 4 ⚠️

- [x] T042 [P] [US4] 编写房主开局权限测试于 `backend/internal/api/multiplayer_start_test.go`
- [x] T043 [P] [US4] 编写“仅当前行动玩家可操作”测试于 `backend/internal/api/multiplayer_turn_test.go`
- [ ] T044 [P] [US4] 编写余额不足导致建房失败的测试于 `backend/internal/api/multiplayer_balance_test.go`
- [x] T045 [P] [US4] 编写余额不足导致占座买入失败且不改变座位状态的测试于 `backend/internal/api/multiplayer_balance_test.go`
- [x] T046 [P] [US4] 编写余额不足导致开局失败且不创建正式牌桌状态的测试于 `backend/internal/api/multiplayer_balance_test.go`
- [x] T047 [P] [US4] 编写前端正式牌桌交互测试于 `frontend/src/multiplayerTable.test.ts`

### Implementation for User Story 4

- [x] T048 [P] [US4] 定义 `GameTable`、`Hand`、`HandParticipant` 存储模型于 `backend/internal/storage/storage.go`
- [x] T049 [US4] 在 `backend/internal/game/game.go` 复用现有规则引擎创建正式多人手牌状态
- [x] T050 [US4] 实现房主开局、当前牌局读取和动作提交接口于 `backend/internal/api/server.go`
- [x] T051 [US4] 实现“当前用户必须匹配当前行动席位”的权限校验于 `backend/internal/api/server.go`
- [x] T052 [US4] 在 `backend/internal/api/server.go` 统一返回 `not_room_owner`、`not_your_turn`、`insufficient_coins`、`forbidden`
- [x] T053 [P] [US4] 扩展前端正式牌局 API 类型于 `frontend/src/api/client.ts`
- [x] T054 [US4] 实现正式牌桌状态展示与动作提交通路于 `frontend/src/App.vue`

**Checkpoint**: User Story 4 should be fully functional and independently testable with prior foundations

---

## Phase 7: User Story 5 - 查看每手牌结果与个人战绩 (Priority: P2)

**Goal**: 一手牌结束后可查询房间历史、个人战绩和输赢明细。

**Independent Test**: 完成多手牌后，从房间最近牌局列表与个人战绩页能读取相同的手牌结果和输赢金额。

### Tests for User Story 5 ⚠️

- [x] T055 [P] [US5] 编写手牌结果归档与事务一致性测试于 `backend/internal/storage/storage_test.go`
- [x] T056 [P] [US5] 编写房间最近牌局结果接口测试于 `backend/internal/api/hand_history_test.go`
- [x] T057 [P] [US5] 编写个人战绩接口测试，覆盖昵称快照展示于 `backend/internal/api/user_history_test.go`
- [x] T058 [P] [US5] 编写前端战绩页展示测试于 `frontend/src/historyPage.test.ts`

### Implementation for User Story 5

- [x] T059 [P] [US5] 定义包含昵称快照的 `HandResult` 聚合读取结构于 `backend/internal/storage/storage.go`
- [x] T060 [US5] 实现手牌结算归档、钱包更新和流水写入原子事务于 `backend/internal/storage/storage.go`
- [x] T061 [US5] 实现房间最近牌局结果与个人战绩接口于 `backend/internal/api/server.go`
- [x] T062 [US5] 刷新 `handsPlayed`、`totalProfit`、`lastPlayedAt` 等统计于 `backend/internal/storage/storage.go`
- [x] T063 [P] [US5] 扩展前端历史与战绩 API 类型于 `frontend/src/api/client.ts`
- [x] T064 [US5] 在 `frontend/src/App.vue` 实现使用昵称快照的房间历史与个人战绩展示

**Checkpoint**: User Stories 1-5 should all remain independently verifiable

---

## Phase 8: User Story 6 - 保留独立规则引擎测试与回放页面 (Priority: P2)

**Goal**: 在新增正式多人能力后，现有测试页仍然独立可用。

**Independent Test**: 访问测试页仍能创建测试牌局、调试设牌、查询历史并执行只读回放，且这些能力不进入正式多人房间。

### Tests for User Story 6 ⚠️

- [x] T065 [P] [US6] 编写规则测试页回归测试于 `backend/internal/api/replay_history_v03_test.go`
- [x] T066 [P] [US6] 编写测试页与正式多人边界测试于 `backend/internal/api/testpage_boundary_test.go`
- [x] T067 [P] [US6] 编写前端测试页入口回归测试于 `frontend/src/App.visual.test.ts`

### Implementation for User Story 6

- [x] T068 [US6] 在 `backend/internal/api/server.go` 审查并保持测试页 API 分组独立
- [x] T069 [US6] 在 `frontend/src/App.vue` 保留独立规则测试页入口与状态隔离
- [x] T070 [US6] 更新测试页与正式多人流程用途说明于 `README.md`

**Checkpoint**: User Story 6 preserves regression and debugging value independently of formal multiplayer

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: 全局一致性、手动验收和收尾说明。

- [x] T071 [P] 对照 `specs/002-multiplayer-poker-v1/spec.md`、`specs/002-multiplayer-poker-v1/data-model.md`、`specs/002-multiplayer-poker-v1/contracts/openapi.yaml` 与 `specs/002-multiplayer-poker-v1/tasks.md` 校验关键名词一致
- [x] T072 [P] 补充多人模式运行说明与数据库模式说明于 `README.md`
- [x] T073 执行 `specs/002-multiplayer-poker-v1/quickstart.md` 中的主流程手动验收
- [x] T074 审查 MySQL 与 SQLite 模式下的行为一致性说明于 `specs/002-multiplayer-poker-v1/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1-3**: Depend on Foundational completion
- **User Story 4**: Depends on User Story 3 room/join/seat capability plus Foundational
- **User Story 5**: Depends on User Story 4 formal hand completion
- **User Story 6**: Can start after Foundational, but must be validated before final delivery
- **Final Phase**: Depends on all desired user stories being complete

### User Story Dependencies

- **US1**: Can start after Foundational - no dependency on other stories
- **US2**: Depends on US1 authenticated user context
- **US3**: Depends on US1 authenticated user context and may rely on US2 wallet checks
- **US4**: Depends on US3 room lifecycle and Foundational auth/error infrastructure
- **US5**: Depends on US4 completion and settlement trigger
- **US6**: Depends on Foundational separation of formal API and test-page API, but should remain independently testable

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Storage/model changes before service/API changes
- Backend API before frontend integration
- Story complete before moving to next P1 dependency story

## Parallel Opportunities

- `US1` 的注册/唯一性/未授权测试可并行编写
- `US2` 的充值契约测试与前端钱包测试可并行编写
- `US3` 的建房、邀请码加入、占座测试可并行编写
- `US4` 的开局权限与当前行动玩家测试可并行编写
- `US5` 的房间历史与个人战绩接口测试可并行编写
- `US6` 的测试页后端边界测试与前端回归测试可并行执行

## Parallel Example: User Story 4

```bash
Task: "T042 [US4] 编写房主开局权限测试于 backend/internal/api/multiplayer_start_test.go"
Task: "T043 [US4] 编写仅当前行动玩家可操作测试于 backend/internal/api/multiplayer_turn_test.go"
Task: "T047 [US4] 编写前端正式牌桌交互测试于 frontend/src/multiplayerTable.test.ts"
```

## Implementation Strategy

### MVP First (US1 → US4)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational
3. 完成 Phase 3: US1 账号与昵称
4. 完成 Phase 4: US2 钱包与模拟充值
5. 完成 Phase 5: US3 房间与邀请码加入
6. 完成 Phase 6: US4 正式多人轮流操作
7. **STOP and VALIDATE**: 先验证正式多人最小闭环

### Incremental Delivery

1. 在 MVP 闭环通过后，再完成 US5 历史与战绩
2. 并行保住 US6 规则测试页不回退
3. 最后完成文档和双数据库模式说明收尾

### Quality Gates

- 所有高风险路径必须先有失败测试
- 不允许把测试页调试设牌能力暴露到正式多人主流程
- 不允许牺牲钱包余额、流水和手牌结果一致性换取实现简化
- 不允许前端自行推断影响牌局结算的规则
