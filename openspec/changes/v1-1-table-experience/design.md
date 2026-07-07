# 技术设计：v1-1-table-experience

## 总体策略

V1.1 继续沿用“HTTP 负责资源查询与历史数据，socket 负责房间实时协作”的模型。正式牌局状态仍以 Go 后端规则引擎、存储事务和 socket 串行 command 为权威来源，前端只展示后端返回的快照和事件。

新增能力分为三类：

- 牌局推进类：行动倒计时、超时自动动作、断线重连状态。
- 牌局可解释类：动作日志、完整回放、房间战绩榜。
- 房间互动类：文字聊天、预设表情。

## 后端设计

### 行动计时器

后端在每次正式手牌进入“等待某座位行动”状态时生成：

- `actingSeatNo`
- `actionStartedAt`
- `actionDeadlineAt`
- `actionTimeoutSeconds`
- `serverTime`

这些字段必须进入当前手牌快照，并随 `room.snapshot`、`hand.started`、`hand.updated` 广播给房间订阅者。

计时器到期后，服务端向同一房间 command 队列注入一个内部 command，而不是直接修改游戏对象：

```text
source: timeout
action: check if legal, otherwise fold
```

这样超时动作与普通玩家动作共享现有权限校验、规则引擎推进、持久化、结算和广播路径。若玩家动作与超时动作几乎同时到达，房间串行处理器必须只接受先被持久化的那一个；后到的 command 基于最新状态重新校验。

### 在线状态与重连快照

`socketHub` 维护每个房间内用户的连接计数。用户至少有一个已订阅连接时视为 `online`，全部断开时视为 `offline`。

房间快照增加：

- `presence`: 每个成员的在线状态、最近断开时间。
- `timer`: 当前行动截止时间。
- `recentActionLog`: 最近动作日志。
- `recentChatMessages`: 最近聊天与表情。
- `leaderboard`: 房间战绩榜摘要。

重连后客户端发送 `room.subscribe`，服务端返回完整 `room.snapshot`。客户端不得自动重放断线前未收到 ACK 的动作。

### 动作日志

新增正式房间动作日志模型，建议按手牌维度递增序号：

- `roomId`
- `handId`
- `seq`
- `kind`
- `street`
- `actorUserId`
- `seatNo`
- `source`: `player`、`timeout`、`system`
- `amount`
- `publicPayload`
- `createdAt`

日志写入必须与对应状态变更处于同一事务或同一失败边界。写入成功后通过 `hand.log.appended` 广播。日志内容只保存公开信息；未摊牌且未公开的底牌不得写入面向房间成员的日志 payload。

### 完整手牌回放

正式多人回放不复用规则测试页 `/api/games/{id}/replay`，避免混淆测试页调试语义。建议新增正式多人 HTTP 只读接口：

- `GET /api/rooms/{roomId}/hands/{handId}/replay`

回放数据由服务端在每个关键节点保存的公开 replay step 组成：

- 开局座位和盲注/前注状态。
- 每次玩家动作后的公开状态。
- 每条街公共牌变化。
- 摊牌与结算公开信息。

回放接口必须校验请求者是该房间成员，并返回公开视角数据。未被公开展示的玩家底牌不得出现在 replay step 中。

### 房间战绩榜

房间战绩榜以已归档正式手牌参与者结果为权威来源，按 `roomId` 聚合：

- `handsPlayed`
- `handsWon`
- `netProfit`
- `biggestPotWon`
- `lastSettledAt`

可以先按查询实时聚合，不强制新增物化表。新手牌结算成功后广播 `room.leaderboard.updated`，payload 可携带摘要，也可以提示前端刷新 HTTP 查询：

- `GET /api/rooms/{roomId}/leaderboard`

### 聊天与表情

新增 socket command：

- `chat.send`

payload 支持：

- `kind: "text"`，`text` 长度限制建议 1 到 200 字符。
- `kind: "emoji"`，`emojiCode` 必须来自服务端允许的预设列表。

服务端校验：

- 用户已登录且是房间成员。
- 用户已订阅或有权限访问该房间。
- 消息长度、emoji code 和发送频率合法。

成功后服务端返回 `ack` 并广播 `chat.message`。最近聊天消息可持久化，供重连快照展示最近 N 条；V1.1 不做私聊、撤回、图片或富文本。

## 前端设计

### 牌桌倒计时

前端从快照中的 `serverTime` 和 `actionDeadlineAt` 计算本地显示倒计时。倒计时只用于 UI 展示；按钮是否可点仍依赖后端返回的 `currentSeat`、当前用户座位和 `availableActions`。

超时后前端等待服务端广播，不自行提交自动动作。

### 重连状态

socket client 保留已有重连能力，并在 UI 中展示连接状态：

- connected
- reconnecting
- disconnected

重连成功后重新订阅房间，并用 `room.snapshot` 覆盖本地 room/hand/timer/log/chat/leaderboard 状态。

### 日志、回放和战绩榜

房间页面新增或扩展信息区：

- 动作日志：显示最近公开动作，支持按手牌清空或切换。
- 房间战绩榜：显示当前房间聚合战绩。
- 房间历史中的每手牌提供“回放”入口。

回放页面或弹层使用 HTTP replay 接口读取数据，并提供上一步、下一步、播放/暂停、回到结算等基础控制。

### 聊天与表情

房间页面增加轻量聊天区域：

- 最近消息列表。
- 短文本输入。
- 预设表情按钮。

发送成功以服务端 `chat.message` 广播为准。若服务端返回限流或非法内容错误，前端显示明确错误，不把消息永久写入本地列表。

## socket 事件扩展

在现有 `room.subscribe`、`room.start_hand`、`room.action` 基础上新增：

- `hand.timer.updated`
- `hand.timeout_applied`
- `player.presence.updated`
- `hand.log.appended`
- `room.leaderboard.updated`
- `chat.send`
- `chat.message`

具体字段见 `contracts/socket-events-v1-1.md`。

## 测试策略

后端至少覆盖：

- 行动截止时间进入房间快照。
- 超时自动 `check`。
- 超时自动 `fold`。
- 玩家动作与超时动作并发时只应用一个有效动作。
- 断线后广播离线，重连后广播在线并返回最新快照。
- 动作日志写入与广播。
- 正式手牌 replay 接口不泄露未公开底牌。
- 房间战绩榜按归档结果聚合。
- 聊天成员权限、长度限制、emoji 白名单和限流。

前端至少覆盖：

- 倒计时展示使用服务端 deadline。
- 重连后重新订阅并覆盖状态。
- 日志、聊天、战绩榜事件更新本地状态。
- 回放控制按 step 展示公开状态。
- socket 错误能显示给用户且不乐观污染状态。

手动验收至少覆盖：

- 两个浏览器账号同房间开局，观察倒计时与超时自动动作。
- 关闭一个账号页面后，另一账号看到离线；重新打开后恢复在线与最新牌桌。
- 完成一手牌后查看动作日志、房间战绩榜和回放。
- 房间成员发送聊天与表情，非成员订阅或发送失败。
