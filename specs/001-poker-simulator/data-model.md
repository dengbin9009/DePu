# 数据模型：德州扑克完整牌局模拟器

## RuleSet

规则集定义一桌牌局使用的扑克牌和比较规则。

字段：

- `id`: `long-holdem` 或 `short-deck`
- `name`: 中文显示名
- `ranks`: 可用点数列表
- `deckSize`: 52 或 36
- `straightRules`: A 低顺子的定义
- `handRanking`: 牌型从大到小排序
- `blindStructure`: 默认小盲、大盲或短牌扩展结构

关系：

- 一个 `Game` 必须引用一个 `RuleSet`。

## Game

一手牌局的权威状态。

字段：

- `id`: 牌局标识
- `rulesetId`: 规则集标识
- `stage`: `waiting`、`preflop`、`flop`、`turn`、`river`、`showdown`、`finished`
- `buttonSeat`: 按钮位
- `smallBlind`: 小盲金额
- `bigBlind`: 大盲金额
- `deck`: 剩余牌堆，调试模式可为空或锁定
- `board`: 公共牌
- `currentSeat`: 当前行动座位
- `minRaise`: 当前最小加注增量
- `currentBet`: 当前街道最高投入
- `pots`: 底池列表
- `version`: 状态版本，用于防止旧快照误提交
- `createdAt` / `updatedAt`: 时间戳

关系：

- 一个 `Game` 有多个 `Seat`、多个 `Action`、多个 `Pot`。

## Seat

座位和玩家在本手牌中的状态。

字段：

- `seatNo`: 座位号
- `name`: 玩家显示名
- `stack`: 剩余筹码
- `holeCards`: 两张手牌
- `status`: `active`、`folded`、`all_in`、`out`
- `streetCommitted`: 当前街道已投入
- `handCommitted`: 本手总投入
- `hasActed`: 当前轮是否已经行动

关系：

- 每个 `Seat` 属于一个 `Game`。

## Action

对牌局状态产生影响的命令和日志。

字段：

- `seq`: 动作序号
- `gameId`: 牌局标识
- `stage`: 动作发生街道
- `seatNo`: 动作玩家；系统发牌动作可以为空
- `type`: `fold`、`check`、`call`、`bet`、`raise`、`all_in`、`deal`、`debug_set_cards`、`settle`
- `amount`: 动作金额
- `payload`: 动作额外数据，例如指定牌或发出的公共牌
- `stateSummary`: 动作后的状态摘要
- `createdAt`: 时间戳

关系：

- `Action` 按 `seq` 排序形成回放基础。

## Pot

主池或边池。

字段：

- `id`: 底池标识
- `gameId`: 牌局标识
- `amount`: 底池金额
- `eligibleSeats`: 有资格竞争该底池的座位
- `winners`: 赢家分配，未结算前为空
- `settled`: 是否已结算

关系：

- 一个 `Game` 可以有一个主池和多个边池。

## ShowdownResult

摊牌比较和筹码分配结果。

字段：

- `gameId`: 牌局标识
- `seatNo`: 玩家座位
- `bestCards`: 最佳五张牌
- `handClass`: 牌型类别
- `rankVector`: 比较向量，用于同牌型比较
- `potAwards`: 从各底池获得的筹码

关系：

- 每个未弃牌且参与摊牌的 `Seat` 有一个 `ShowdownResult`。

## 状态转换

```text
waiting
  -> preflop
  -> flop
  -> turn
  -> river
  -> showdown
  -> finished
```

特殊转换：

- 任意下注阶段只剩一名未弃牌玩家时，直接进入 `finished`。
- 所有未弃牌玩家全下时，自动发完剩余公共牌后进入 `showdown`。
- 调试模式只能在未结算且未锁定的阶段修改指定牌。
