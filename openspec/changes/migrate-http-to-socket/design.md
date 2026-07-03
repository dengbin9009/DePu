# 技术设计：migrate-http-to-socket

## 总体策略

采用“HTTP 保留资源型能力，socket 承担实时房间能力”的混合模型。

HTTP 继续负责：

- 注册、登录、当前用户资料、昵称修改
- 钱包余额、充值档位、模拟充值
- 创建房间、邀请码加入
- 房间最近手牌、个人战绩等历史查询
- 独立规则测试页 `/api/games/*`

socket 负责：

- 连接鉴权
- 订阅和取消订阅正式多人房间
- 广播房间成员、座位、房间状态、当前手牌状态
- 房主开局 command + ack
- 当前玩家动作 command + ack
- 手牌结算、钱包变化提示、错误事件
- 重连后的房间快照恢复

## 后端结构

在 `backend/internal/api` 中新增 socket 层，保持规则引擎和存储接口为权威来源。

建议拆分：

- `socketHub`：管理连接、房间订阅、按房间广播。
- `socketClient`：保存单个连接、用户身份、已订阅房间。
- `socketEnvelope`：统一消息包格式。
- `roomEventBuilder`：把现有 `RoomRecord` 和 `game.Game` 转换成前端可消费的 room/hand 快照。

`Server` 继续持有 `Store`。socket command 处理必须调用现有 store 和 game 逻辑，不复制规则判断。

## 连接与鉴权

客户端通过类似 `/api/socket?token=<token>` 或 `Authorization: Bearer <token>` 建立连接。后端必须复用当前 `requireUser` 的会话语义识别用户。

鉴权失败时：

- 拒绝建立 socket 连接，或建立后立即发送 `error` 并关闭。
- 错误 code 使用 `unauthorized`。

连接建立成功后，服务端发送 `connection.ready`，包含当前用户 id、serverTime 和协议版本。

## 消息信封

所有 socket 消息使用 JSON 信封：

```json
{
  "type": "room.action",
  "requestId": "client-generated-id",
  "roomId": "room_123",
  "payload": {}
}
```

字段约定：

- `type`：事件或 command 类型。
- `requestId`：客户端 command 必须携带；服务端 ACK 或错误应回传同一值。
- `roomId`：房间相关消息必须携带。
- `payload`：具体事件数据。
- `sentAt`：服务端事件可携带发送时间。

## Command 与 ACK

客户端发送：

- `room.subscribe`
- `room.unsubscribe`
- `room.start_hand`
- `room.action`
- `room.refresh`

服务端返回：

- `ack`：command 已被接受并处理成功。
- `error`：command 被拒绝，包含 `code`、`message`、`field` 和原始 `requestId`。

服务端在 ACK 后或同一事务完成后广播状态事件。客户端不能只根据“自己发出的 command”本地乐观推进牌局，必须以后端广播或 ACK payload 中的快照为准。

## 广播事件

服务端向房间内已订阅连接广播：

- `room.snapshot`：订阅成功或显式刷新时发送完整房间快照。
- `room.updated`：成员、座位、房主、状态变化。
- `hand.started`：新手牌创建。
- `hand.updated`：动作推进后的手牌状态。
- `hand.settled`：手牌完成结算，包含结果摘要和钱包刷新提示。
- `wallet.updated`：只发送给相关用户，提示重新读取钱包或携带新余额。
- `error`：订阅级别或广播级错误。

## 房间权限

订阅房间前，服务端必须确认：

- 用户已登录。
- 房间存在。
- 用户是房间成员，或是创建/加入流程刚返回的合法成员。

开局 command 必须确认：

- 用户是当前房主。
- 房间未关闭。
- 已入座玩家数量满足 `minPlayersToStart`。
- 开局前金币和买入条件仍满足。

行动 command 必须确认：

- 房间存在且有当前正式手牌。
- 用户所在座位等于 `game.CurrentSeat`。
- 动作在规则引擎返回的合法动作范围内。

## 并发与顺序

每个房间的 command 必须串行处理。实现可以在 hub 中对 roomId 加锁，或通过单房间队列保证同一房间内的开局和行动按接收顺序处理。

关键规则：

- 同一房间同一时间只能有一个 command 修改正式手牌状态。
- 非法 command 不改变 `game.Game` 和数据库状态。
- 一手牌结束时，`ArchiveHandResult`、钱包更新和房间状态更新必须保持原子一致。
- 广播只能发生在持久化成功之后；如果持久化失败，必须向发起连接返回 `storage_error`，不得广播成功状态。

## 前端状态流

前端新增 socket client，建议位于 `frontend/src/api/socketClient.ts` 或相近模块。

进入房间后：

1. 使用现有 token 建立 socket 连接。
2. 发送 `room.subscribe`。
3. 使用收到的 `room.snapshot` 初始化 `room`、`currentRoomHand`、`recentRoomHands` 的实时部分。
4. 用户点击开局时发送 `room.start_hand`。
5. 当前玩家点击动作按钮时发送 `room.action`。
6. 收到 `hand.updated` 或 `hand.settled` 后更新牌桌状态。
7. 收到 `wallet.updated` 后刷新或更新钱包。

离开房间或退出登录时：

- 发送 `room.unsubscribe`。
- 关闭 socket 或取消当前房间订阅。
- 清理本地 room/hand 状态。

## 重连

客户端断线后应按短间隔重试。重连成功后必须重新发送 `room.subscribe`，服务端返回完整 `room.snapshot`，客户端用快照覆盖本地实时状态。

重连期间：

- 前端应显示连接状态，但不得允许用户重复提交未确认动作。
- 如果存在未完成 command，重连后应提示用户根据最新快照重试，而不是自动重放可能已经过期的动作。

## 与历史查询的关系

`hand.settled` 可以携带结算摘要，但房间历史和个人战绩的权威来源仍是 HTTP 历史接口和数据库。前端收到结算事件后可以刷新：

- `/api/rooms/{roomId}/hands/recent`
- `/api/me/hands`
- `/api/me/wallet`

也可以使用事件中的摘要立即展示，但刷新失败时不得修改已持久化结果。

## 测试策略

后端：

- socket 鉴权测试。
- 订阅非成员房间失败测试。
- 房主开局 socket command 测试。
- 非房主开局失败测试。
- 当前玩家 action 成功并广播测试。
- 非当前玩家 action 返回 `not_your_turn` 且状态不变测试。
- 断线重连后订阅返回最新快照测试。
- 手牌结算后广播发生在持久化成功之后测试。

前端：

- socket client 消息解析和 ACK/error 处理测试。
- `useAppState` 不再依赖房间轮询推进正式牌局状态。
- 多事件顺序下 room/hand/wallet 状态更新测试。
- 断线重连后重新订阅测试。

手动验收：

- 两个浏览器或隐身窗口登录不同账号，同房间入座并开局。
- 一个玩家操作后，另一个玩家无须刷新或等待轮询即可看到状态变化。
- 非当前玩家点击动作收到错误且桌面状态不变。
- 手牌结束后两个账号都看到结算变化，钱包和历史刷新一致。

