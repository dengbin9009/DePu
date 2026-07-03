# socket 事件契约草案

## 连接地址

建议地址：`/api/socket`

鉴权方式二选一，具体实现时选择当前前后端更容易稳定测试的方式：

- 查询参数：`/api/socket?token=<session-token>`
- 请求头：`Authorization: Bearer <session-token>`

## 通用信封

客户端 command：

```json
{
  "type": "room.action",
  "requestId": "req_1700000000000",
  "roomId": "room_123",
  "payload": {
    "action": "call",
    "amount": 0
  }
}
```

服务端事件：

```json
{
  "type": "hand.updated",
  "requestId": "req_1700000000000",
  "roomId": "room_123",
  "payload": {},
  "sentAt": "2026-07-03T12:00:00Z"
}
```

服务端错误：

```json
{
  "type": "error",
  "requestId": "req_1700000000000",
  "roomId": "room_123",
  "payload": {
    "code": "not_your_turn",
    "message": "not your turn",
    "field": ""
  },
  "sentAt": "2026-07-03T12:00:00Z"
}
```

## 客户端发送

### `room.subscribe`

订阅房间实时事件。

```json
{
  "type": "room.subscribe",
  "requestId": "req_subscribe_1",
  "roomId": "room_123",
  "payload": {}
}
```

成功后服务端发送 `ack` 和 `room.snapshot`。

### `room.unsubscribe`

取消房间订阅。

```json
{
  "type": "room.unsubscribe",
  "requestId": "req_unsubscribe_1",
  "roomId": "room_123",
  "payload": {}
}
```

### `room.refresh`

请求服务端重新发送完整房间快照。

```json
{
  "type": "room.refresh",
  "requestId": "req_refresh_1",
  "roomId": "room_123",
  "payload": {}
}
```

### `room.start_hand`

房主发起开局。

```json
{
  "type": "room.start_hand",
  "requestId": "req_start_1",
  "roomId": "room_123",
  "payload": {}
}
```

### `room.action`

当前玩家提交动作。

```json
{
  "type": "room.action",
  "requestId": "req_action_1",
  "roomId": "room_123",
  "payload": {
    "action": "raise",
    "amount": 300
  }
}
```

## 服务端发送

### `connection.ready`

连接已通过鉴权。

```json
{
  "type": "connection.ready",
  "payload": {
    "userId": "user_123",
    "protocolVersion": "1",
    "serverTime": "2026-07-03T12:00:00Z"
  }
}
```

### `ack`

客户端 command 处理成功。

```json
{
  "type": "ack",
  "requestId": "req_action_1",
  "roomId": "room_123",
  "payload": {
    "command": "room.action"
  }
}
```

### `room.snapshot`

完整房间快照。用于订阅成功、刷新和重连恢复。

```json
{
  "type": "room.snapshot",
  "roomId": "room_123",
  "payload": {
    "room": {
      "id": "room_123",
      "inviteCode": "ABCD12",
      "ownerUserId": "user_1",
      "status": "playing",
      "seatCount": 6,
      "minPlayersToStart": 2,
      "members": [],
      "seats": []
    },
    "hand": {
      "roomId": "room_123",
      "handId": "game_123",
      "status": "preflop",
      "currentSeat": 2,
      "pot": 100,
      "boardCards": [],
      "players": [],
      "availableActions": ["fold", "call", "raise"]
    }
  }
}
```

### `room.updated`

房间成员、座位、房主或状态变化。

```json
{
  "type": "room.updated",
  "roomId": "room_123",
  "payload": {
    "room": {}
  }
}
```

### `hand.started`

正式手牌开始。

```json
{
  "type": "hand.started",
  "requestId": "req_start_1",
  "roomId": "room_123",
  "payload": {
    "hand": {}
  }
}
```

### `hand.updated`

动作推进后的手牌状态。

```json
{
  "type": "hand.updated",
  "requestId": "req_action_1",
  "roomId": "room_123",
  "payload": {
    "hand": {}
  }
}
```

### `hand.settled`

一手牌完成结算。

```json
{
  "type": "hand.settled",
  "requestId": "req_action_1",
  "roomId": "room_123",
  "payload": {
    "handId": "hand_123",
    "handNo": 7,
    "winnerSummary": "Alice wins 1200",
    "potSummary": "main pot 1200",
    "boardCards": ["As", "Kd", "7c", "2h", "9s"],
    "participants": [
      {
        "userId": "user_1",
        "nickname": "Alice",
        "seatNo": 1,
        "profit": 600
      }
    ]
  }
}
```

### `wallet.updated`

钱包变化提示，仅发送给相关用户连接。

```json
{
  "type": "wallet.updated",
  "roomId": "room_123",
  "payload": {
    "reason": "hand_settled",
    "handId": "hand_123"
  }
}
```

### `error`

错误响应。

常见 code：

- `unauthorized`
- `forbidden`
- `room_not_found`
- `not_room_owner`
- `not_your_turn`
- `invalid_action`
- `insufficient_coins`
- `storage_error`
- `bad_message`

