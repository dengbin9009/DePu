# 效果图驱动功能生产验收报告（2026-07-07）

## 结论

本轮按 OpenSpec `mockup-driven-table-experience` 对新增功能重新做了生产级验收。实现计划已完成；本轮验收中发现并修复 4 个问题，最终自动化回归和最终 HTTP/API 验收通过。

## 本轮修复的问题

1. 站起围观误移除房间成员
   - 现象：`DELETE /api/rooms/{roomId}/seats/{seatNo}` 会把用户从 `room_members` 删除；房主站起后失去房主身份，无法开局。
   - 修复：站起围观只释放座位并退还买入，保留成员和房主身份。
   - 回归：`TestStandUpKeepsRoomMembershipAndOwnership`。

2. 非成员可直接占座
   - 现象：未通过邀请码加入房间的用户可以直接调用坐下接口占座。
   - 修复：`TakeSeat` 增加房间成员校验；非成员返回 `403 forbidden`。
   - 回归：`TestTakeSeatRequiresRoomMembership`。

3. 开局中可站起导致座位/手牌状态分叉
   - 现象：房间进入 `playing` 后仍可站起并退还买入，可能破坏当前手牌权威状态。
   - 修复：站起围观仅允许 `waiting` 状态；`playing` 时返回 `409 room_not_waiting`。
   - 回归：`TestStandUpRejectsPlayingRoom`。

4. 移动端空座点击与商城确认不稳定
   - 现象：浏览器实测中点击牌桌空座未稳定打开买入弹窗；商城使用原生 `window.confirm` 导致 in-app Browser 会话卡死。
   - 修复：座位按钮增加稳定 `data-testid`/`aria-label` 并提升座位层级；牌桌中心信息禁用指针抢占；商城改为应用内确认弹层。
   - 回归：`multiplayerTable.test.ts`、生产构建、浏览器空座点击复测。

## 最终通过项

### 自动化回归

- `cd backend && go test ./... -count=1`：通过。
- `cd frontend && npm test -- --run`：13 个测试文件，54 个测试全部通过。
- `cd frontend && npm run build`：通过，产物生成成功。
- `npx --yes @fission-ai/openspec validate mockup-driven-table-experience --strict --no-interactive`：通过。
- `git diff --check`：通过。

### 最终 HTTP/API 验收

结果文件：`docs/qa/mockup-production-http-final-20260707.json`

- 总数：10
- 通过：10
- 失败：0

覆盖内容：

- 注册与初始钱包。
- 创建 8K 短牌训练赛房间。
- 金币不足买入拒绝。
- 模拟充值写入钱包流水。
- 充值后 8K 买入成功。
- 非成员禁止入座和查看战绩榜。
- 等待态站起释放座位但保留成员/房主身份。
- 邀请码大小写/空格归一加入。
- 非房主禁止开局，房主可开局。
- 开局中站起返回 `room_not_waiting`。
- `/api/games/*` 规则测试页接口隔离可用。

### Socket 验证

- `go test ./internal/api -run 'TestSocket|TestHTTPSeatChangesBroadcastRoomUpdateAndActionLog|TestOwnerCanStartRoomHandAndFetchCurrentHand|TestNonOwnerCannotStartAndNonCurrentActorCannotAct' -count=1`：通过。
- 覆盖订阅快照、房间更新广播、座位变化 action log、聊天、开局、行动、非房主/非当前行动者边界。

### in-app Browser UI 验收

已使用 `browser:control-in-app-browser` 执行并产出截图：

- 创建比赛首屏和底部提交。
- 牌桌等待态。
- 空座点击打开补充记分牌（修复后复测通过）。
- 聊天、当前战绩、牌谱回顾、牌桌设置面板打开。
- 牌桌开局后状态可见。
- UI 重叠复测：修复后非父子严重重叠为 0。

主要截图：

- `docs/qa/prod-ui-create-top-mobile-20260707.png`
- `docs/qa/prod-ui-create-bottom-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-room-waiting-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-empty-seat-buyin-retest-mobile-20260707.png`
- `docs/qa/prod-ui-chat-panel-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-score-panel-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-replay-panel-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-settings-panel-fixed-mobile-20260707.png`

## 已知测试环境限制

in-app Browser 在测试商城旧版原生 `window.confirm` 时卡住，之后无法稳定选择新 iab 标签。已将商城确认改为应用内弹层，并用前端测试、构建、最终 HTTP/API 验收覆盖充值链路。由于插件会话层卡死，改动后的应用内确认弹层没有完成新的 in-app Browser 端到端截图，但其源码测试、构建和后端充值/买入链路均已通过。

## 最终判定

本轮需求实现已完成。阻断级问题已修复，最终自动化回归和后端接口验收通过。UI 浏览器验收完成了主要牌桌与面板流程；商城确认弹层的视觉复测受 in-app Browser 插件会话卡死限制，已由代码级测试与接口级生产链路验证补齐。
