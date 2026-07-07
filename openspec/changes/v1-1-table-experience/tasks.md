# 实施任务：v1-1-table-experience

## 1. OpenSpec 与协议

- [x] T001 确认 V1.1 范围：行动倒计时、断线重连增强、动作日志、完整回放、房间战绩榜、聊天表情。
- [x] T002 更新 socket 事件契约，定义 timer、presence、log、leaderboard、chat 事件与错误 code。
- [x] T003 明确正式多人回放接口与规则测试页回放接口的边界。

## 2. 后端行动倒计时与超时

- [x] T004 为正式手牌快照增加 `actionStartedAt`、`actionDeadlineAt`、`actionTimeoutSeconds`、`serverTime`。
- [x] T005 在每次进入等待玩家行动状态时生成服务端权威 deadline。
- [x] T006 实现房间级计时调度，到期后向房间串行 command 队列注入 timeout command。
- [x] T007 实现超时自动动作规则：可 `check` 则 `check`，否则 `fold`。
- [x] T008 覆盖玩家动作与超时动作竞争场景，确保只持久化一个有效动作。
- [x] T009 编写后端测试覆盖 deadline 快照、自动 `check`、自动 `fold` 和超时日志。

## 3. 后端断线重连与在线状态

- [x] T010 在 socket hub 中维护房间成员连接计数与在线状态。
- [x] T011 断线后广播 `player.presence.updated`。
- [x] T012 重连订阅时返回包含 timer、presence、recentActionLog、recentChatMessages、leaderboard 的 `room.snapshot`。
- [x] T013 确认断线不自动行动，仍由统一超时规则处理。
- [x] T014 编写断线、重连、多连接同用户的后端测试。

## 4. 后端动作日志与完整回放

- [x] T015 设计并迁移正式房间动作日志与 replay step 存储结构。
- [x] T016 在入座、离座、开局、玩家动作、超时动作、结算时写入公开动作日志。
- [x] T017 广播 `hand.log.appended`，并在 `room.snapshot` 中返回最近日志。
- [x] T018 在正式手牌关键节点保存公开 replay step。
- [x] T019 新增 `GET /api/rooms/{roomId}/hands/{handId}/replay` 只读接口。
- [x] T020 编写回放权限测试，确认非成员不能读取。
- [x] T021 编写回放隐私测试，确认未公开底牌不出现在 replay payload 中。

## 5. 后端房间战绩榜

- [x] T022 新增 `GET /api/rooms/{roomId}/leaderboard`。
- [x] T023 按房间归档结果聚合参与手数、胜利手数、净输赢、最大赢得底池、最近结算时间。
- [x] T024 结算成功后广播 `room.leaderboard.updated`。
- [x] T025 编写战绩榜聚合与权限测试。

## 6. 后端聊天与表情

- [x] T026 定义服务端预设 emoji code 列表。
- [x] T027 实现 `chat.send` socket command，支持短文本与预设表情。
- [x] T028 校验房间成员身份、消息长度、emoji 白名单和发送频率。
- [x] T029 广播 `chat.message`，并在重连快照中返回最近聊天消息。
- [x] T030 编写聊天权限、长度、emoji 白名单和限流测试。

## 7. 前端牌桌体验

- [x] T031 在 socket client 中支持新增事件与 `chat.send` command。
- [x] T032 在 `useAppState` 中保存 timer、presence、actionLog、chatMessages、leaderboard 状态。
- [x] T033 牌桌页展示行动倒计时，并使用服务端 deadline 计算剩余时间。
- [x] T034 牌桌页展示成员在线/离线状态和 socket 重连状态。
- [x] T035 牌桌页展示实时动作日志。
- [x] T036 房间页展示房间战绩榜，并在结算后更新。
- [x] T037 房间页增加聊天与预设表情发送入口。
- [x] T038 房间历史增加正式手牌回放入口。
- [x] T039 新增正式手牌回放视图，支持上一步、下一步、播放/暂停和回到结算。
- [x] T040 编写前端状态、socket 事件、聊天错误和回放控制测试。

## 8. 验证

- [x] T041 运行后端测试：`cd backend && go test ./... -count=1`。
- [x] T042 运行前端测试：`cd frontend && npm test -- --run`。
- [x] T043 手动用两个账号验证倒计时、超时自动动作和断线重连。
- [x] T044 手动完成一手牌并验证动作日志、回放、战绩榜、聊天与表情。
- [x] T045 手动验证规则测试页仍可独立创建测试牌局、调试设牌、历史和只读回放。
