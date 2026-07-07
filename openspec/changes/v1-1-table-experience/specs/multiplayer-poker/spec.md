# multiplayer-poker 增量规格：v1-1-table-experience

## ADDED Requirements

### Requirement: 服务端权威行动倒计时

正式多人手牌 MUST 在每次等待玩家行动时提供服务端权威行动截止时间。前端 MAY 展示倒计时，但 MUST NOT 使用本地计时器自行判定牌局超时或改变手牌状态。

#### Scenario: 当前玩家行动倒计时进入快照

- **GIVEN** 正式多人手牌正在等待某个座位行动
- **WHEN** 服务端发送 `room.snapshot`、`hand.started` 或 `hand.updated`
- **THEN** payload 包含 `actingSeatNo`、`actionStartedAt`、`actionDeadlineAt`、`actionTimeoutSeconds` 和 `serverTime`
- **AND** 房间成员看到同一个服务端截止时间

#### Scenario: 超时后自动过牌

- **GIVEN** 当前行动玩家在截止时间前未提交动作
- **AND** 当前规则状态允许 `check`
- **WHEN** 服务端处理超时
- **THEN** 服务端以 `timeout` 来源应用 `check`
- **AND** 广播 `hand.timeout_applied`
- **AND** 广播推进后的 `hand.updated` 或 `hand.settled`
- **AND** 动作日志记录该次超时自动过牌

#### Scenario: 超时后自动弃牌

- **GIVEN** 当前行动玩家在截止时间前未提交动作
- **AND** 当前规则状态不允许 `check` 但允许 `fold`
- **WHEN** 服务端处理超时
- **THEN** 服务端以 `timeout` 来源应用 `fold`
- **AND** 广播 `hand.timeout_applied`
- **AND** 不允许前端本地直接改变手牌结果

#### Scenario: 玩家动作与超时动作竞争

- **GIVEN** 玩家动作和超时 command 几乎同时到达服务端
- **WHEN** 服务端串行处理同一房间 command
- **THEN** 只有第一个基于最新状态仍合法的 command 会改变手牌状态
- **AND** 后到 command 必须基于最新状态重新校验
- **AND** 非法或过期 command 返回明确错误且不重复推进牌局

### Requirement: 断线重连与房间在线状态

正式多人房间 MUST 展示成员在线状态，并在用户断线重连后通过 `room.snapshot` 恢复权威状态。

#### Scenario: 成员断线后广播离线

- **GIVEN** 房间成员已通过 socket 订阅房间
- **WHEN** 该成员所有订阅连接断开
- **THEN** 服务端将该成员标记为 `offline`
- **AND** 向房间订阅者广播 `player.presence.updated`

#### Scenario: 成员重连后恢复快照

- **GIVEN** 房间成员曾经断线
- **WHEN** 该成员重新建立 socket 连接并发送 `room.subscribe`
- **THEN** 服务端将该成员标记为 `online`
- **AND** 返回最新 `room.snapshot`
- **AND** 快照包含房间、当前手牌、倒计时、成员在线状态、最近动作日志、最近聊天消息和房间战绩榜摘要

#### Scenario: 断线期间不自动重放动作

- **GIVEN** 用户发送 `room.action` 后断线且未收到 ACK
- **WHEN** 用户重连成功
- **THEN** 客户端不得自动重放该 command
- **AND** 用户只能基于最新 `room.snapshot` 重新提交仍然合法的动作

### Requirement: 实时动作日志

正式多人房间 MUST 为房间成员展示服务端生成的实时动作日志。动作日志 MUST 只包含当前公开信息，不得提前泄露未展示底牌。

#### Scenario: 玩家动作进入日志

- **GIVEN** 当前行动玩家提交合法动作
- **WHEN** 服务端持久化该动作并广播手牌更新
- **THEN** 服务端写入动作日志
- **AND** 向房间成员广播 `hand.log.appended`
- **AND** 日志包含街道、座位、玩家昵称快照、动作类型、金额和发生时间

#### Scenario: 系统事件进入日志

- **WHEN** 房间发生入座、离座、开局、超时自动动作、摊牌或结算
- **THEN** 服务端写入对应系统日志
- **AND** 房间成员可以在牌桌动作日志中看到该事件

#### Scenario: 日志不泄露隐藏底牌

- **GIVEN** 某玩家弃牌且底牌未公开
- **WHEN** 房间成员查看动作日志
- **THEN** 日志不得包含该玩家未公开底牌

### Requirement: 正式手牌完整回放

已结算的正式多人手牌 MUST 支持按公开步骤回放。正式多人回放 MUST 与独立规则测试页回放保持入口和语义隔离。

#### Scenario: 从房间历史进入回放

- **GIVEN** 房间内存在已结算正式手牌
- **WHEN** 房间成员从房间历史打开该手牌回放
- **THEN** 系统返回按顺序排列的 replay steps
- **AND** 每个 step 包含该时点可公开展示的座位、公共牌、底池、行动和结算摘要

#### Scenario: 非成员不能读取回放

- **GIVEN** 用户不是该房间成员
- **WHEN** 用户请求正式手牌回放
- **THEN** 系统返回 `forbidden` 或 `room_not_found`

#### Scenario: 回放不复用测试页调试能力

- **WHEN** 用户查看正式多人手牌回放
- **THEN** 系统不得允许从回放节点继续提交正式牌局动作
- **AND** 系统不得暴露规则测试页的调试设牌能力

### Requirement: 房间战绩榜

正式多人房间 MUST 提供房间维度战绩榜，用于展示该房间内玩家的聚合表现。战绩榜 MUST 以已归档正式手牌结果为权威来源。

#### Scenario: 查询房间战绩榜

- **GIVEN** 房间内已经完成至少一手正式手牌
- **WHEN** 房间成员查看房间战绩榜
- **THEN** 系统展示玩家昵称、参与手数、胜利手数、净输赢、最大赢得底池和最近结算时间

#### Scenario: 结算后更新战绩榜

- **GIVEN** 一手正式手牌结算成功
- **WHEN** 服务端完成手牌归档和钱包更新
- **THEN** 服务端广播 `room.leaderboard.updated` 或返回最新战绩榜摘要
- **AND** 前端展示的房间战绩榜随之更新

#### Scenario: 非成员不能读取房间战绩榜

- **GIVEN** 用户不是该房间成员
- **WHEN** 用户请求房间战绩榜
- **THEN** 系统返回 `forbidden` 或 `room_not_found`

### Requirement: 房间聊天与预设表情

正式多人房间 MUST 支持房间成员发送短文本聊天和服务端允许的预设表情。聊天和表情 MUST NOT 改变牌局状态。

#### Scenario: 房间成员发送文本聊天

- **GIVEN** 用户已登录且是房间成员
- **WHEN** 用户发送 `chat.send`，payload 为合法文本消息
- **THEN** 服务端返回同 `requestId` 的 `ack`
- **AND** 向房间成员广播 `chat.message`
- **AND** 消息包含发送者昵称快照、消息内容和发送时间

#### Scenario: 房间成员发送预设表情

- **GIVEN** 用户已登录且是房间成员
- **WHEN** 用户发送 `chat.send`，payload 为服务端允许的 `emojiCode`
- **THEN** 服务端广播对应 `chat.message`
- **AND** 前端展示该预设表情

#### Scenario: 非成员不能发送聊天

- **GIVEN** 用户不是房间成员
- **WHEN** 用户向该房间发送 `chat.send`
- **THEN** 服务端返回 `forbidden` 或 `room_not_found`
- **AND** 不向房间成员广播消息

#### Scenario: 非法聊天被拒绝

- **WHEN** 用户发送空文本、过长文本、未知 `emojiCode` 或超过发送频率限制
- **THEN** 服务端返回明确错误 code
- **AND** 不改变手牌状态
- **AND** 不写入房间聊天记录

### Requirement: 独立规则测试页继续隔离

V1.1 新增的倒计时、正式手牌回放、动作日志、房间战绩榜、聊天和表情 MUST NOT 破坏独立规则测试页既有 HTTP 调试与回放能力。

#### Scenario: 规则测试页仍按原路径工作

- **WHEN** 测试者进入独立规则测试页并创建测试牌局、设置调试牌或执行只读回放
- **THEN** 前端继续调用 `/api/games/*` HTTP 接口
- **AND** 正式多人 socket 房间不会收到这些测试页事件
