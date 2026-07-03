# multiplayer-poker 增量规格：migrate-http-to-socket

## MODIFIED Requirements

### Requirement: 正式多人通信通道

正式多人房间的实时状态同步、开局和玩家动作提交 MUST 通过 socket 通道完成。HTTP 仍 MAY 保留账号、资料、钱包、充值、建房、邀请码加入、历史查询和调试兜底读取能力，但正式牌桌 UI MUST NOT 依赖周期性 HTTP 轮询推进房间或手牌状态。

#### Scenario: 进入房间后订阅实时状态

- **GIVEN** 用户已登录并已加入房间
- **WHEN** 前端进入该房间页面
- **THEN** 前端通过 socket 建立鉴权连接
- **AND** 发送 `room.subscribe`
- **AND** 服务端返回或广播 `room.snapshot`
- **AND** 快照包含房间成员、座位、房间状态和当前手牌状态

#### Scenario: 正式牌桌不依赖周期轮询

- **GIVEN** 用户停留在正式多人牌桌页面
- **WHEN** 其他玩家入座、离座、开局或提交动作
- **THEN** 当前用户通过 socket 事件看到状态更新
- **AND** 前端不得依赖固定间隔调用 `/api/rooms/{roomId}` 或 `/api/rooms/{roomId}/current-hand` 才能看到该变化

### Requirement: socket 开局

房主 MUST 能通过 socket command 发起正式手牌，服务端 MUST 复用现有房主权限、座位人数、金币条件和规则引擎校验。

#### Scenario: 房主通过 socket 开局

- **GIVEN** 房主已登录、已订阅房间且房间满足开局条件
- **WHEN** 房主发送 `room.start_hand`
- **THEN** 服务端创建正式手牌
- **AND** 向发起连接返回同 `requestId` 的 `ack`
- **AND** 向房间订阅者广播 `hand.started`

#### Scenario: 非房主通过 socket 开局

- **GIVEN** 非房主已订阅房间
- **WHEN** 该用户发送 `room.start_hand`
- **THEN** 服务端返回同 `requestId` 的 `error`
- **AND** 错误 code 为 `not_room_owner`
- **AND** 不创建正式手牌

### Requirement: socket 玩家动作

当前行动玩家 MUST 能通过 socket command 提交正式牌局动作。服务端 MUST 在应用动作前校验用户身份、当前行动席位、合法动作和房间状态。

#### Scenario: 当前玩家提交合法动作

- **GIVEN** 当前行动席位属于用户 A
- **AND** 用户 A 已订阅房间
- **WHEN** 用户 A 发送 `room.action`，payload 包含合法 `action` 和 `amount`
- **THEN** 服务端使用后端规则引擎应用动作
- **AND** 持久化推进后的手牌状态
- **AND** 向用户 A 返回 `ack`
- **AND** 向房间订阅者广播 `hand.updated` 或 `hand.settled`

#### Scenario: 非当前玩家提交动作

- **GIVEN** 当前行动席位不属于用户 B
- **WHEN** 用户 B 发送 `room.action`
- **THEN** 服务端返回同 `requestId` 的 `error`
- **AND** 错误 code 为 `not_your_turn`
- **AND** 不改变手牌状态

#### Scenario: 非法动作不改变状态

- **GIVEN** 当前行动玩家已订阅房间
- **WHEN** 该玩家发送规则引擎判定为非法的 `room.action`
- **THEN** 服务端返回 `invalid_action`
- **AND** 不改变手牌状态、钱包余额或历史记录

## ADDED Requirements

### Requirement: socket 鉴权与房间订阅权限

socket 连接和房间订阅 MUST 使用现有登录 token 鉴权。服务端 MUST 拒绝未登录用户或非房间成员访问正式房间实时状态。

#### Scenario: 无效 token 建连

- **WHEN** 客户端使用缺失或无效 token 建立 socket 连接
- **THEN** 服务端拒绝连接或立即发送 `unauthorized` 错误并关闭连接

#### Scenario: 非成员订阅房间

- **GIVEN** 用户已登录但不是房间成员
- **WHEN** 用户发送 `room.subscribe`
- **THEN** 服务端返回 `forbidden` 或 `room_not_found`
- **AND** 不把该连接加入房间广播列表

### Requirement: socket 事件顺序与持久化

服务端 MUST 在同一房间内串行处理会修改手牌状态的 socket command，并且 MUST 在持久化成功后再广播成功事件。

#### Scenario: 同房间并发动作

- **GIVEN** 同一房间几乎同时收到两个 `room.action`
- **WHEN** 服务端处理这些 command
- **THEN** 服务端按单一顺序串行应用
- **AND** 第二个 command 必须基于第一个 command 持久化后的最新状态重新校验

#### Scenario: 持久化失败不广播成功

- **GIVEN** 服务端应用动作后写入数据库失败
- **WHEN** 该 command 处理结束
- **THEN** 服务端向发起连接返回 `storage_error`
- **AND** 不向房间广播 `hand.updated`、`hand.started` 或 `hand.settled`

### Requirement: 断线重连恢复

客户端 socket 断开后 MUST 能重新连接并通过订阅房间获得权威快照。客户端 MUST NOT 自动重放断线前未确认的动作 command。

#### Scenario: 重连后恢复房间快照

- **GIVEN** 用户在正式牌局中断开 socket
- **WHEN** 客户端重新连接并发送 `room.subscribe`
- **THEN** 服务端返回最新 `room.snapshot`
- **AND** 前端使用该快照覆盖本地 room/hand 实时状态

#### Scenario: 未确认动作不自动重放

- **GIVEN** 用户发送 `room.action` 后连接断开且未收到 ACK
- **WHEN** 客户端重连成功
- **THEN** 客户端不自动重放该 command
- **AND** 用户只能基于最新快照重新提交仍然合法的动作

### Requirement: 结算广播与钱包提示

当 socket 动作导致一手牌结束时，服务端 MUST 在手牌结果、参与者输赢、钱包更新和流水写入成功后，向房间成员广播结算事件，并向相关用户发送钱包更新提示。

#### Scenario: 手牌结束广播结算

- **GIVEN** 当前玩家提交的动作导致一手牌结束
- **WHEN** 服务端完成结算事务
- **THEN** 服务端广播 `hand.settled`
- **AND** 事件包含手牌编号、赢家摘要、底池摘要、公共牌和参与者输赢摘要

#### Scenario: 钱包更新提示

- **GIVEN** 手牌结算改变了用户钱包余额
- **WHEN** 结算事务成功
- **THEN** 服务端向相关用户发送 `wallet.updated`
- **AND** 前端刷新或更新钱包余额、个人战绩和房间历史

### Requirement: 独立规则测试页不迁移到正式 socket

独立规则测试页 MUST 保留既有 HTTP 调试与回放语义，且 MUST NOT 使用正式多人 socket 协议暴露调试设牌、测试历史或只读回放能力。

#### Scenario: 测试页继续使用 HTTP

- **WHEN** 测试者进入独立规则测试页并创建测试牌局、设置调试牌或执行回放
- **THEN** 前端继续调用 `/api/games/*` HTTP 接口
- **AND** 正式多人 socket 房间不会收到这些测试页事件
