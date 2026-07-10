# 数据模型：多人德州扑克对战 v1

## 概览

本功能在现有规则引擎和测试页之外，引入正式多人产品所需的账号、钱包、房间和战绩模型。生产默认数据库为 MySQL，开发/测试允许 SQLite，但实体语义保持一致。

## 实体

### User

- **用途**: 登录主体与用户全局主键。
- **关键字段**:
  - `id`
  - `username`：全站唯一，登录标识
  - `passwordHash`
  - `status`：`active | disabled`
  - `createdAt`
  - `updatedAt`
- **约束**:
  - `username` 全站唯一
  - 不保存明文密码

### UserProfile

- **用途**: 用户展示资料与基础统计。
- **关键字段**:
  - `userId`
  - `nickname`：全站唯一，牌桌/资料页主展示名
  - `handsPlayed`
  - `totalProfit`
  - `lastPlayedAt`
  - `updatedAt`
- **约束**:
  - `nickname` 全站唯一
  - 与 `User` 为一对一关系

### Wallet

- **用途**: 记录用户当前金币余额。
- **关键字段**:
  - `userId`
  - `balance`
  - `updatedAt`
- **约束**:
  - 与 `User` 为一对一关系
  - 余额业务上不可为负

### WalletTransaction

- **用途**: 记录钱包变更流水。
- **关键字段**:
  - `id`
  - `userId`
  - `type`：`recharge_simulated | hand_settlement_credit | hand_settlement_debit`
  - `amount`
  - `balanceAfter`
  - `referenceType`：`recharge | hand`
  - `referenceId`
  - `note`
  - `createdAt`
- **约束**:
  - 每次余额变更必须对应至少一条流水
  - `balanceAfter` 应可用于审计该时点余额

### Room

- **用途**: 房主组织对战的房间。
- **关键字段**:
  - `id`
  - `inviteCode`：唯一邀请码
  - `ownerUserId`
  - `status`：`waiting | playing | closed`
  - `ruleSetId`
  - `seatCount`
  - `minPlayersToStart`
  - `createdAt`
  - `updatedAt`
- **约束**:
  - `inviteCode` 唯一
  - `ownerUserId` 必须存在

### RoomMember

- **用途**: 记录加入房间的用户成员关系。
- **关键字段**:
  - `roomId`
  - `userId`
  - `role`：`owner | player`
  - `joinedAt`
- **约束**:
  - 同一房间中 `userId` 唯一

### RoomSeat

- **用途**: 记录房间中的座位占用情况。
- **关键字段**:
  - `roomId`
  - `seatNo`
  - `userId`：可空，空表示该座位无人占用
  - `buyInChips`
  - `seatStatus`：`empty | occupied | sitting_out`
  - `updatedAt`
- **约束**:
  - 同一房间中 `seatNo` 唯一
  - 同一时刻一个用户在同一房间最多占一个座位

### GameTable

- **用途**: 正式多人牌桌实例，连接房间与正在进行的牌局。
- **关键字段**:
  - `id`
  - `roomId`
  - `status`：`idle | in_hand | settling`
  - `currentHandId`
  - `startedAt`
  - `updatedAt`
- **约束**:
  - 一个房间同一时刻最多存在一个活动牌桌上下文

### Hand

- **用途**: 单手牌的执行与结算记录。
- **关键字段**:
  - `id`
  - `tableId`
  - `handNo`
  - `status`：`dealing | acting | showdown | settled`
  - `ruleSetId`
  - `startedAt`
  - `settledAt`
  - `winnerSummary`
- **约束**:
  - 同一 `tableId` 下 `handNo` 递增唯一

### HandParticipant

- **用途**: 某用户在某手牌中的参与快照。
- **关键字段**:
  - `handId`
  - `userId`
  - `seatNo`
  - `nicknameSnapshot`
  - `stackStart`
  - `stackEnd`
  - `profit`
  - `resultType`：`win | lose | split | fold`
- **约束**:
  - 同一 `handId` + `userId` 唯一
  - `nicknameSnapshot` 用于保留当时展示名

### HandResult

- **用途**: 单手牌结算结果的查询视图或聚合对象。
- **关键字段**:
  - `handId`
  - `roomId`
  - `tableId`
  - `completedAt`
  - `boardCards`
  - `potSummary`
  - `winnerSummary`
  - `participants[]`
- **约束**:
  - 一手牌结算完成后必须可从 `Hand` 和 `HandParticipant` 重建或直接读取

## 状态流转

### Room.status

- `waiting`：房间已创建，允许加入、入座、离座、调整准备状态
- `playing`：已有正式牌局在进行
- `closed`：房间关闭，不再允许新操作

### RoomSeat.seatStatus

- `empty`：无人占座
- `occupied`：已有用户在座且参与正式流程
- `sitting_out`：保留座位但当前不参与下一手牌

### GameTable.status

- `idle`：房间可开新手牌
- `in_hand`：当前手牌进行中
- `settling`：当前手牌结束，正在归档结果和更新钱包

### Hand.status

- `dealing`：发牌和初始化阶段
- `acting`：行动轮阶段
- `showdown`：摊牌比较阶段
- `settled`：已完成结算并落库

### WalletTransaction.type

- `recharge_simulated`：模拟充值成功
- `hand_settlement_credit`：用户在某手牌中净赢得金币
- `hand_settlement_debit`：用户在某手牌中净输掉金币

## 关系说明

- `User` 1:1 `UserProfile`
- `User` 1:1 `Wallet`
- `User` 1:N `WalletTransaction`
- `Room` 1:N `RoomMember`
- `Room` 1:N `RoomSeat`
- `Room` 1:1 `GameTable`（逻辑活动上下文）
- `GameTable` 1:N `Hand`
- `Hand` 1:N `HandParticipant`
- `Hand` 1:1 `HandResult`（逻辑查询结果，可为聚合读取模型）

## 一致性与事务要求

- 用户注册成功时，应同时具备 `User`、`UserProfile` 和 `Wallet` 的初始记录。
- 模拟充值成功时，应原子更新 `Wallet.balance` 并写入 `WalletTransaction`。
- 一手牌结算成功时，应原子完成以下动作：
  - 更新 `Hand` 为 `settled`
  - 写入或刷新 `HandParticipant` / `HandResult`
  - 更新涉及用户的钱包余额
  - 追加对应的钱包流水
  - 刷新用户统计字段（如 `handsPlayed`、`totalProfit`、`lastPlayedAt`）
- 不允许出现“手牌结果已写入但钱包未更新”或“钱包已更新但手牌结果缺失”的部分成功状态。

## 测试页与正式流程的数据边界

- 现有规则测试页可继续使用其既有 `Game`、历史和回放存储语义。
- 正式多人房间不得依赖“测试页调试锁定牌”作为常规用户能力。
- 若未来复用底层规则引擎对象，必须在文档和实现中明确区分：
  - 测试页上下文：允许调试设牌、只读回放
  - 正式多人上下文：强调鉴权、房间成员、当前行动玩家约束和钱包结算
