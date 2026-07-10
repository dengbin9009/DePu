# socket 事件契约草案：V1.1 牌桌体验

本文档扩展 `migrate-http-to-socket` 已建立的 socket 信封：

```json
{
  "type": "event.or.command",
  "requestId": "client-generated-id",
  "roomId": "room_123",
  "payload": {},
  "sentAt": "2026-07-06T12:00:00Z"
}
```

## 快照新增字段

`room.snapshot` 的 `payload` SHOULD 增加：

```json
{
  "timer": {
    "actingSeatNo": 2,
    "actionStartedAt": "2026-07-06T12:00:00Z",
    "actionDeadlineAt": "2026-07-06T12:00:30Z",
    "actionTimeoutSeconds": 30,
    "serverTime": "2026-07-06T12:00:10Z"
  },
  "presence": [
    {
      "userId": "user_1",
      "seatNo": 1,
      "status": "online",
      "lastDisconnectedAt": null
    }
  ],
  "recentActionLog": [],
  "recentChatMessages": [],
  "leaderboard": []
}
```

## 计时事件

### `hand.timer.updated`

服务端在当前行动座位或截止时间变化时广播。

```json
{
  "actingSeatNo": 2,
  "actionStartedAt": "2026-07-06T12:00:00Z",
  "actionDeadlineAt": "2026-07-06T12:00:30Z",
  "actionTimeoutSeconds": 30,
  "serverTime": "2026-07-06T12:00:00Z"
}
```

### `hand.timeout_applied`

服务端自动处理超时后广播。

```json
{
  "handId": "hand_1",
  "seatNo": 2,
  "action": "fold",
  "source": "timeout",
  "appliedAt": "2026-07-06T12:00:30Z"
}
```

## 在线状态

### `player.presence.updated`

```json
{
  "userId": "user_1",
  "seatNo": 1,
  "status": "offline",
  "lastDisconnectedAt": "2026-07-06T12:01:00Z"
}
```

`status` 允许值：

- `online`
- `offline`

## 动作日志

### `hand.log.appended`

```json
{
  "entry": {
    "handId": "hand_1",
    "seq": 12,
    "kind": "player_action",
    "street": "flop",
    "seatNo": 2,
    "nickname": "玩家A",
    "action": "check",
    "amount": 0,
    "source": "player",
    "createdAt": "2026-07-06T12:02:00Z"
  }
}
```

`kind` 初始允许值：

- `seat_taken`
- `seat_left`
- `hand_started`
- `player_action`
- `timeout_action`
- `street_dealt`
- `showdown`
- `hand_settled`

## 房间战绩榜

### `room.leaderboard.updated`

```json
{
  "items": [
    {
      "userId": "user_1",
      "nickname": "玩家A",
      "handsPlayed": 8,
      "handsWon": 3,
      "netProfit": 420,
      "biggestPotWon": 900,
      "lastSettledAt": "2026-07-06T12:03:00Z"
    }
  ]
}
```

## 聊天与表情

### `chat.send`

客户端 command。

文本消息：

```json
{
  "kind": "text",
  "text": "这手精彩"
}
```

预设表情：

```json
{
  "kind": "emoji",
  "emojiCode": "nice_hand"
}
```

### `chat.message`

服务端广播。

```json
{
  "message": {
    "id": "chat_1",
    "kind": "emoji",
    "text": "",
    "emojiCode": "nice_hand",
    "userId": "user_1",
    "nickname": "玩家A",
    "createdAt": "2026-07-06T12:04:00Z"
  }
}
```

## 新增错误 code

- `action_timeout_race_lost`：客户端动作到达时当前行动已被超时或其他动作推进。
- `chat_message_empty`：聊天文本为空。
- `chat_message_too_long`：聊天文本超过服务端长度限制。
- `chat_emoji_unknown`：表情 code 不在服务端预设列表。
- `chat_rate_limited`：发送频率超过限制。
- `replay_forbidden`：用户无权读取该正式手牌回放。
