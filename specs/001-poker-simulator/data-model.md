# 数据模型：德州扑克完整牌局模拟器

## RuleSet

规则集定义一桌牌局使用的扑克牌、比较规则和可用下注结构。

字段：

- `id`: `long-holdem` 或 `short-deck`
- `name`: 中文显示名
- `ranks`: 可用点数列表
- `deckSize`: 52 或 36
- `straightRules`: A 低顺子的定义
- `handRanking`: 牌型从大到小排序
- `bettingStructures`: 可选下注结构列表
- `defaultBettingStructure`: 默认下注结构

关系：

- 一个 `Game` 必须引用一个 `RuleSet`。
- `long-holdem` 必须支持 `blinds`。
- `short-deck` 必须支持 `blinds` 和 `ante`。

## BettingStructure

下注结构定义开局强制投入和翻牌前行动顺序。

字段：

- `type`: `blinds` 或 `ante`
- `smallBlind`: 小盲金额，仅 `blinds` 结构需要
- `bigBlind`: 大盲金额，仅 `blinds` 结构需要
- `ante`: 全员前注金额，仅 `ante` 结构需要
- `buttonBlind`: 按钮位额外盲注，仅 `ante` 结构需要
- `preflopFirstActorRule`: 翻牌前首个行动者规则
- `postflopFirstActorRule`: 翻牌后首个行动者规则

接口边界：

- `preflopFirstActorRule` 和 `postflopFirstActorRule` 是后端领域规则描述，由 `type`、规则集、玩家数量和按钮位推导；v1 OpenAPI 不要求客户端提交或接收这两个字段。客户端必须以 `GameSnapshot.currentSeat` 和 `legalActions` 为准，不得自行推断行动权。

校验：

- `blinds` 结构需要正整数 `smallBlind` 和 `bigBlind`，且 `smallBlind < bigBlind`。
- `ante` 结构需要正整数 `ante` 和 `buttonBlind`，二者不要求固定倍数或大小关系。
- `long-holdem` 只允许 `blinds`。
- `short-deck` 允许 `blinds` 或 `ante`。

规则：

- `blinds` 结构下，长牌和短牌都使用小盲/大盲。单挑时按钮位同时是小盲；翻牌前按钮位先行动，翻牌后按钮位后行动。
- 短牌 `ante` 结构下，所有在局玩家支付 `ante`，按钮位额外支付 `buttonBlind`。
- 短牌 `ante` 结构下，`ante` 计入每名玩家翻牌前本轮已投入额，按钮位额外 `buttonBlind` 形成 `ante + buttonBlind` 的当前最高投入。
- 短牌 `ante` 结构下，翻牌前由按钮左侧最近的可行动玩家先行动，翻牌后仍由按钮位后行动。
- 任意强制投入如果超过玩家剩余筹码，则玩家以剩余筹码支付并进入全下状态。

## Game

一手牌局的权威状态。

字段：

- `id`: 牌局标识
- `rulesetId`: 规则集标识
- `bettingStructure`: 当前牌局下注结构和金额
- `dealMode`: `random` 或 `debug`
- `stage`: `waiting`、`preflop`、`flop`、`turn`、`river`、`showdown`、`finished`
- `buttonSeat`: 按钮位
- `deck`: 剩余牌堆，必须排除已发牌和调试指定牌
- `board`: 公共牌
- `currentSeat`: 当前行动座位
- `minRaise`: 当前最小加注增量
- `currentBet`: 当前街道最高投入；短牌 `ante` 结构初始化为实际最高强制投入
- `pots`: 底池列表
- `showdown`: 摊牌结果，未结算前为空
- `isReplay`: 是否为只读回放快照
- `version`: 状态版本，用于防止旧快照误提交
- `debugLocked`: 调试牌是否已锁定
- `createdAt` / `updatedAt`: 时间戳

展示规则：

- `GameSnapshot` 必须向前端暴露 `currentBet` 和 `minRaise`，用于下注/加注滑轨计算最小目标金额、最大可全下金额和不足额全下提示。
- 前端可以用滑轨、数字输入和快捷按钮生成 `SubmitActionRequest.amount`，但不得自行放宽下注、加注、最小加注或不足额全下规则；所有金额仍必须由后端根据权威 `Game` 状态校验。

关系：

- 一个 `Game` 有多个 `Seat`、多个 `Action`、多个 `Pot`。
- 回放接口返回的 `Game` 快照必须为只读，不得作为继续提交动作的权威状态。

校验：

- `buttonSeat` 必须对应一个已存在座位。
- 只读回放快照不得接受玩家动作或调试牌修改。
- 当任一玩家动作提交或牌局进入后续街道后，`debugLocked` 必须为 true。

## Seat

座位和玩家在本手牌中的状态。

字段：

- `seatNo`: 座位号
- `name`: 玩家显示名
- `stack`: 剩余筹码
- `holeCards`: 两张手牌；调试模式可部分预设
- `status`: `active`、`folded`、`all_in`、`out`
- `streetCommitted`: 当前街道已投入
- `handCommitted`: 本手总投入
- `hasActed`: 当前轮是否已经行动
- `currentHand`: 可空展示字段，翻牌圈及之后由当前公共牌和该玩家手牌计算出的当前最佳牌型；翻牌前、弃牌、出局或牌数不足时为空

校验：

- 创建牌局时 `seatNo` 必须是从 1 到玩家人数的连续整数。
- `name` 必须非空，且在本局内唯一。
- `stack` 必须为正整数；若强制投入超过 `stack`，则按 `stack` 实际投入并设为 `all_in`。

关系：

- 每个 `Seat` 属于一个 `Game`。
- 短牌 `ante` 结构初始化时，每名在局玩家的 `streetCommitted` 和 `handCommitted` 都增加实际支付的 `ante`；按钮位额外增加实际支付的 `buttonBlind`。
- `currentHand` 只存在于对外 `GameSnapshot` 展示中，不作为持久化权威状态；前端不得自行推断牌型，应使用后端快照字段。

## DebugCardAssignment

调试模式中的指定牌信息。

字段：

- `gameId`: 牌局标识
- `holeCards`: 按座位号指定的手牌，允许只指定部分座位或单个座位的完整两张牌
- `board`: 指定公共牌，最多 5 张
- `locked`: 是否已锁定

校验：

- 只能在首个玩家动作前修改。
- 不允许重复牌。
- 不允许不属于当前规则集牌组的牌。
- 不允许超过阶段数量的公共牌或超过 5 张公共牌。
- 已指定牌必须从 `deck` 中移除，后续未指定手牌和公共牌从剩余合法牌堆随机补足。

## Action

对牌局状态产生影响的命令和日志。

字段：

- `seq`: 动作序号
- `gameId`: 牌局标识
- `stage`: 动作发生街道
- `seatNo`: 动作玩家；系统发牌、强制下注或结算动作可以为空
- `type`: `fold`、`check`、`call`、`bet`、`raise`、`all_in`、`deal`、`debug_set_cards`、`forced_bet`、`settle`
- `amount`: 动作金额
- `payload`: 动作额外数据，例如指定牌、发出的公共牌、强制下注结构或短筹码标记
- `stateSummary`: 动作后的状态摘要对象
- `createdAt`: 时间戳

关系：

- `Action` 按 `seq` 排序形成只读回放基础。
- `forced_bet` 记录强制投入的实际支付金额；短筹码玩家的金额可能低于应缴金额。

`stateSummary` 内容：

- `stage`: 动作后的牌局阶段
- `currentSeat`: 动作后的当前行动座位；无行动者时为空
- `currentBet`: 动作后当前街道最高投入
- `potTotal`: 动作后所有底池总额
- `board`: 动作后公共牌列表
- `activeSeats`: 仍可行动座位
- `allInSeats`: 已全下座位
- `foldedSeats`: 已弃牌座位
- `isReplay`: 该摘要是否来自只读回放快照

该摘要用于行动历史解释和回放节点列表展示，不替代完整 `Game` 快照。

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
- `amount` 根据每个座位的实际 `handCommitted` 计算，包含实际支付的盲注、`ante`、`buttonBlind`、下注、跟注、加注和全下金额。

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

## ReplayRequest

回放请求。

字段：

- `gameId`: 牌局标识
- `toSeq`: 目标动作序号

校验：

- `toSeq=0` 返回初始发牌后的只读快照。
- `toSeq` 小于 0 或大于该牌局最新动作序号时必须返回错误。
- `toSeq` 指向已存在的结算或已结束动作节点时返回该节点后的最终只读快照。
- 回放不得改变当前权威 `Game` 状态。

## TableVisualState

前端牌桌视觉状态，由后端 `Game` 快照、合法动作和行动历史派生，不持久化为规则权威。

字段：

- `seatPositions`: 每个座位在牌桌上的视觉位置和朝向；主视角玩家固定在底部专区，其余座位应避开底部主手牌区
- `cardDisplayStates`: 手牌、公共牌和牌背的展示状态；模拟器 v1 可展示所有玩家手牌，主视角用大牌，其余座位用小牌
- `currentHandLabels`: 翻牌圈及之后显示的当前最佳牌型标签；翻牌前不显示
- `activeSeat`: 当前行动者高亮座位
- `dealerButtonSeat`: 按钮位标识
- `potVisual`: 底池和筹码的视觉摘要
- `betAmountControl`: 下注/加注金额输入状态，包含当前动作类型、滑轨最小值、最大值、步进、当前选中金额和快捷按钮选项
- `animationPhase`: `idle`、`deal`、`reveal_board`、`action_shift`、`pot_update`
- `replayTransition`: 手动切换回放动作节点时的短暂过渡状态
- `reducedMotion`: 是否根据浏览器偏好降级为静态展示

规则：

- `TableVisualState` 只能由后端快照、行动历史或前端视口尺寸重新计算。
- `seatPositions` 必须支持 2 到 10 人；4 人桌在 1280px 桌面第一屏完整可见，且非主视角座位必须布局在上方和左右侧，不得压到底部主玩家信息、行动按钮或主手牌。
- 发牌、翻公共牌、行动切换和底池变化动画不得修改后端权威状态。
- 当前最佳牌型标签只能使用后端快照派生结果，不得成为下注、结算或回放的规则权威。
- 下注金额滑轨只能作为输入辅助；它必须使用后端快照中的 `currentBet`、`minRaise`、当前玩家 `streetCommitted` 和 `stack` 计算范围，并在提交前由后端再次校验。
- 回放节点切换可以使用短暂过渡动画，但 v1 不维护整手牌自动播放时间轴。
- 动画失败、浏览器降级或用户偏好减少动态效果时，牌桌信息仍必须完整可读。
- 桌面和移动视口都必须避免玩家座位、卡牌、底池和行动控制发生不可理解的重叠；可读性至少覆盖玩家名称、筹码、当前投入、座位小手牌、主玩家大手牌、底池金额和行动按钮。

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
- 调试模式只能在首个玩家动作前修改指定牌；首个玩家动作后或进入后续街道后锁定。
- 调试模式只指定部分牌时，后续发牌从剩余合法牌堆随机补足。
- 回放到指定动作序号只生成只读快照，不改变当前牌局状态。
- v1 回放只支持手动切换动作节点，不产生自动播放时间轴状态。
