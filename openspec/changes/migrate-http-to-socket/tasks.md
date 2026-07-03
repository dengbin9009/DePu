# 实施任务：migrate-http-to-socket

## 1. 后端 socket 基础设施

- [x] T001 在 `backend/internal/api` 新增 socket 消息信封类型，覆盖 `type`、`requestId`、`roomId`、`payload`、`sentAt`。
- [x] T002 在 `backend/internal/api` 新增 socket hub，管理连接、用户身份、房间订阅和按房间广播。
- [x] T003 为 `Server.Routes()` 增加 socket endpoint，并复用现有 token 会话鉴权。
- [x] T004 编写 socket 鉴权失败测试，验证无 token 或无效 token 返回 `unauthorized`。
- [x] T005 编写连接成功测试，验证服务端发送 `connection.ready`。

## 2. 房间订阅与快照

- [x] T006 实现 `room.subscribe` command，订阅成功后返回 `room.snapshot`。
- [x] T007 实现 `room.unsubscribe` command，释放连接的房间订阅。
- [x] T008 校验订阅权限：非房间成员不得订阅房间。
- [x] T009 编写订阅非成员房间失败测试。
- [x] T010 编写订阅成功后返回 room/hand 快照测试。

## 3. socket 开局与动作提交

- [x] T011 实现 `room.start_hand` command，复用现有 HTTP 开局逻辑中的房主、座位、金币和规则引擎校验。
- [x] T012 实现 `room.action` command，复用现有 HTTP 动作提交逻辑中的当前行动玩家和合法动作校验。
- [x] T013 为同一房间 command 增加串行处理保护，避免并发动作破坏手牌状态。
- [x] T014 编写房主通过 socket 开局成功并广播 `hand.started` 测试。
- [x] T015 编写非房主通过 socket 开局返回 `not_room_owner` 测试。
- [x] T016 编写当前玩家通过 socket 提交动作成功并广播 `hand.updated` 测试。
- [x] T017 编写非当前玩家通过 socket 提交动作返回 `not_your_turn` 且状态不变测试。
- [x] T018 编写非法动作返回 `invalid_action` 且状态不变测试。

## 4. 结算、钱包与历史事件

- [x] T019 在 socket 动作导致手牌结束时，确保手牌结果归档、钱包更新、房间状态更新成功后再广播。
- [x] T020 广播 `hand.settled` 给房间订阅者。
- [x] T021 向相关用户发送 `wallet.updated`，提示前端刷新钱包。
- [x] T022 编写结算后 `hand.settled` 与 `wallet.updated` 事件测试。
- [x] T023 编写持久化失败时不广播成功事件的测试。

## 5. 前端 socket client

- [x] T024 新增前端 socket client 模块，封装连接、重连、发送 command、ACK/error 匹配和事件订阅。
- [x] T025 为 socket client 编写消息解析、ACK、error 和断线重连测试。
- [x] T026 在 `useAppState` 中接入 socket：进入房间后连接并发送 `room.subscribe`。
- [x] T027 将 `doStartRoomHand` 改为发送 `room.start_hand`。
- [x] T028 将 `doRoomAction` 改为发送 `room.action`。
- [x] T029 移除正式牌局推进对 `startRoomPolling()` 的依赖，保留必要的手动刷新能力。
- [x] T030 收到 `room.snapshot`、`room.updated`、`hand.started`、`hand.updated`、`hand.settled` 后更新本地 room/hand 状态。
- [x] T031 收到 `wallet.updated` 后刷新或更新钱包余额和个人战绩。
- [x] T032 退出登录、离开房间或切换房间时取消订阅并关闭连接。

## 6. HTTP 边界与兼容

- [ ] T033 保留账号、资料、钱包、充值、建房、邀请码加入、历史查询 HTTP API。
- [ ] T034 明确 `/api/rooms/{roomId}/current-hand` 可作为重连或调试兜底读取接口，但正式 UI 不再周期轮询它。
- [ ] T035 规则测试页 `/api/games/*` 保持原行为，不接入正式多人 socket 协议。
- [ ] T036 更新 README，说明正式多人实时通道、HTTP 保留边界和本地验收步骤。

## 7. 验证

- [ ] T037 运行后端规则和多人相关测试：`cd backend && go test ./internal/api ./internal/storage ./internal/game -count=1`。
- [ ] T038 运行前端测试：`cd frontend && npm test -- --run` 或项目等效命令。
- [ ] T039 手动使用两个账号验证同房间 socket 同步：入座、开局、行动、结算、钱包和历史刷新。
- [ ] T040 手动验证规则测试页仍可创建测试牌局、调试设牌、查看历史和只读回放。
