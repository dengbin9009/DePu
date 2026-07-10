# 效果图驱动新功能生产级验收报告 R2

生成时间：2026-07-07 23:40 Asia/Shanghai  
分支：`codex/mockup-driven-feature-optimization`  
测试环境：后端 `127.0.0.1:5190`，前端 `127.0.0.1:5191`，MySQL 隔离库 `depu_prod_qa_20260707_r2`

## 结论

本次 OpenSpec 计划项已实施完成。本轮重新做了真实后端 HTTP API、WebSocket、前端单测、类型检查、生产构建和 OpenSpec 严格校验，核心业务验收通过。

本轮新增的真实接口与 socket 验收脚本共 58 项，结果为 58 通过、0 失败。结果文件：

- `docs/qa/mockup-production-r2-api-socket-20260707.json`
- `docs/qa/run-mockup-production-r2.mjs`

## 本轮发现与处理

1. 验收脚本中的 `variant` 一开始使用了展示名 `short-deck`，真实接口契约为 `short_holdem`。已修正脚本。
2. 效果图中的“带入记分牌上限：无限制”在当前产品实现中会被前端提交为 `60000`，后端也拒绝显式 `buyInCap=0`。本轮按当前已实现契约验收通过，但这属于产品语义风险：UI 展示“无限制”与服务端表达不是同一语义。
3. 验收脚本最初使用了座位字段 `chips`，真实响应字段为 `buyInChips`。已修正脚本。
4. 非房主开局和非当前行动玩家操作返回更精确错误码：`not_room_owner`、`not_your_turn`。验收脚本已对齐后端契约。
5. `/api/games` 兼容性用例最初用了 `long-holdem + no-limit`，当前规则引擎合法样例为 `short-deck + ante`。已修正脚本并通过。
6. Socket payload 在 Node WebSocket 中可能已是对象，不一定是 JSON 字符串。验收脚本已兼容两种格式。

以上修正均为测试脚本或验收契约修正，没有引入额外业务代码改动。

## HTTP API 覆盖

已覆盖并通过：

- `/health`
- `/api/auth/register`：正常注册、短密码、重复用户名、重复昵称
- `/api/auth/login`：正常登录、错误密码
- `/api/me`：未登录拒绝
- `/api/rulesets`
- `/api/rooms`：创建训练赛、SNG 暂不支持、座位数越界、买入范围冲突、配置字段返回
- `/api/rooms/join`：无效邀请码、大小写/空格归一化、重复加入幂等
- `/api/rooms/{id}`：详情
- `/api/rooms/{id}/seats/{seatNo}`：非成员拒绝、低于最小带入、高于上限、余额不足、入座成功、重复入座、等待态站起、游戏中站起拒绝
- `/api/rooms/{id}/start`：非房主拒绝、房主开局
- `/api/rooms/{id}/current-hand`
- `/api/rooms/{id}/actions`：非当前行动玩家拒绝、当前行动玩家合法动作
- `/api/rooms/{id}/leaderboard`：非成员拒绝、成员可访问
- `/api/rooms/{id}/hands/recent`
- `/api/rooms/{id}/hands/{handId}/replay`：不存在回放返回 `replay_not_found`
- `/api/me/hands`
- `/api/me/wallet`：充值和买入流水
- `/api/recharge/options`
- `/api/recharge`：未确认拒绝、模拟充值成功
- `/api/games`：规则测试页创建兼容
- `/api/games/{id}/replay`：越界返回 `replay_out_of_range`

## WebSocket 覆盖

已覆盖并通过：

- 连接后收到 `connection.ready`
- 成员 `room.subscribe` 返回 `ack` 与 `room.snapshot`
- `room.snapshot` 包含房间、战绩、聊天记录和在线状态
- 非成员订阅返回 `forbidden`
- 空聊天返回 `chat_message_empty`
- 文本聊天 `chat.send` 成功广播 `chat.message`
- 频率限制返回 `chat_rate_limited`
- 合法表情广播成功
- 房主通过 `room.start_hand` 开局并广播 `hand.started`
- 非法动作返回 `invalid_action`，不广播成功状态
- `room.unsubscribe` 返回 `ack`

## UI 与内置浏览器验证

按要求读取并使用了 `browser:control-in-app-browser`。本轮内置浏览器的浏览器发现层可用，可列出 `Codex In-app Browser`；但标签页层在多次 fresh browser 后仍然不可用：

- `browser.tabs.new()` 连续超时
- `browser.tabs.selected()` 超时
- `browser.tabs.list()` 超时
- 按文档读取 `bootstrap-troubleshooting` 与 `browser-troubleshooting` 后，重新获取 fresh browser 仍无法获得 fresh tab

因此，本轮无法再用内置浏览器产生新的 UI 截图或完成新的 UI 点击链路。没有切换到 Chrome、外部 Playwright 或其他浏览器控制方式，因为该技能文档明确要求只使用内置浏览器通道。

可用的 UI 证据来自此前同一功能分支已生成的内置浏览器截图与记录，包括：

- `docs/qa/prod-ui-create-top-mobile-20260707.png`
- `docs/qa/prod-ui-create-bottom-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-room-waiting-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-empty-seat-buyin-retest-mobile-20260707.png`
- `docs/qa/prod-ui-chat-panel-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-score-panel-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-replay-panel-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-settings-panel-fixed-mobile-20260707.png`
- `docs/qa/prod-ui-shop-confirm-retest-mobile-20260707.png`

本轮 UI 风险状态：

- 创建比赛、牌桌、补充记分牌、商城确认、聊天、战绩、牌谱、设置面板已有截图证据。
- 本轮未能通过内置浏览器重新执行一次完整 UI 点击流程，原因是工具标签页控制不可用。
- 前端测试、类型检查和生产构建均通过，接口与 socket 权威状态 58/58 通过。

## 自动化回归

全部通过：

- `cd backend && go test ./... -count=1`
- `cd frontend && PATH="/Users/dengbin/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH" npm test -- --run`
- `cd frontend && PATH="/Users/dengbin/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH" npm run typecheck`
- `cd frontend && PATH="/Users/dengbin/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH" npm run build`
- `npx --yes @fission-ai/openspec validate mockup-driven-table-experience --strict --no-interactive`
- `git diff --check`

## 仍需产品确认的风险

1. “带入记分牌上限：无限制”当前前端提交为 `60000`，后端不接受 `0` 作为无限制。建议后续明确：到底是隐藏展示语义，还是服务端需要支持 `0/null` 代表无限制。
2. 本轮内置浏览器标签页控制不可用，导致无法完成新的 UI 点击回归。业务逻辑已经通过接口、socket、前端测试和构建覆盖，但严格意义上的“本轮新截图级 UI 验收”未完成。

## 最终状态

业务功能计划已完成；本轮接口与 socket 生产级验收通过；自动化回归通过；OpenSpec 严格校验通过。内置浏览器工具在标签页层不可用，本报告已按实际证据记录。
