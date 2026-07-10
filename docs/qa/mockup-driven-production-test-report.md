# 效果图驱动牌桌体验优化 - 生产级测试报告

生成时间：2026-07-07T14:16:10.601Z

## 结论

本轮按照 OpenSpec 变更 `mockup-driven-table-experience` 完成生产级验收。HTTP 接口、移动端 UI 主流程、响应式断点、OpenSpec 严格校验、后端全量测试、前端单测、类型检查和生产构建均通过。

## 本轮测试中发现并修复的问题

1. 后端创建房间接口会把显式传入的 `0` 当作“未传”并套默认值。已修复为通过指针字段区分“省略”和“显式 0”，并对 `ante/minBuyIn/maxBuyIn/buyInCap/durationMinutes/seatCount/minPlayersToStart` 返回明确字段错误。
2. 移动端创建页提交按钮曾固定在视口底部，覆盖训练时长选项。已改为表单内自然布局，底部滚动无明显交互重叠。
3. 移动牌桌等待态 `选座位` 与底部工具栏在小屏上发生重叠。已上移等待态 CTA。
4. 移动牌桌仍显示旧版右侧工具面板，导致底部工具栏与旧聊天按钮重叠。已在 `.room-mobile-screen` 下隐藏旧工具面板。
5. 小屏长昵称会撑高座位节点并与 #9 座位重叠。已固定座位节点高度，并对姓名/状态做单行省略。

## 后端接口验收

报告文件：`docs/qa/mockup-api-qa.json`

- 总计：36
- 通过：36
- 失败：0

覆盖接口与边界：

- `GET /health`
- `POST /api/auth/register`
- `POST /api/recharge`
- `POST /api/rooms`：创建带效果图配置的短牌训练赛房间
- `GET /api/rooms/:roomId`：房间元数据持久化
- `POST /api/rooms/join`：邀请码加入与错误邀请码
- `POST /api/rooms/:roomId/seats/:seatNo`：最小/最大买入、余额不足、重复入座
- `POST /api/rooms/:roomId/start`：HTTP 兼容开局
- `GET /api/rooms/:roomId/current-hand`

重点边界：SNG 未开放、奥马哈未开放、座位数过少/过多、最少开局人数大于座位数、负数最小买入、最大买入小于最小买入、带入上限低于最大买入，以及所有关键数值字段显式传 `0`。

## 浏览器 UI 主流程验收

报告文件：`docs/qa/mockup-ui-qa-final.json`

- 总计：26
- 通过：26
- 失败：0

覆盖流程：

- 登录页和注册页展示
- 注册后进入大厅
- 创建比赛页核心控件、SNG/奥马哈禁用态、短牌训练赛恢复可提交
- 创建页底部滚动区域重叠检测
- 创建后进入牌桌等待态
- 补充记分牌弹窗打开与确认
- 金币不足入口跳转商城、商城分类占位、返回原牌桌上下文
- 入座后个人区域展示
- 聊天、战绩、牌谱、设置四个抽屉打开和关闭
- 浏览器控制台应用错误过滤检查

关键截图：

- `docs/qa/final-login-mobile.png`
- `docs/qa/final-create-bottom-mobile.png`
- `docs/qa/final-room-waiting-mobile.png`
- `docs/qa/final-room-seated-mobile.png`

## 响应式与视觉几何验收

创建页/商城页报告：`docs/qa/mockup-responsive-qa.json`

- 总计：6
- 通过：6
- 失败：0

牌桌页报告：`docs/qa/mockup-room-responsive-qa.json`

- 总计：3
- 通过：3
- 失败：0

覆盖断点：

- 390x844 小屏手机
- 430x932 标准手机
- 1280x720 桌面宽屏

检查项：页面非空、核心文案可见、无横向溢出、主要交互元素无明显重叠。

额外牌桌重叠复测：`docs/qa/mockup-ui-overlap-retest.json`，2/2 通过。

## 自动化回归

已执行并通过：

- `cd backend && go test ./... -count=1`
- `cd frontend && vitest --run`：54/54 tests passed
- `cd frontend && vue-tsc --noEmit`
- `cd frontend && vite build`
- `npx --yes @fission-ai/openspec validate mockup-driven-table-experience --strict --no-interactive`
- `git diff --check`

## 非目标确认

本轮未实现真实支付、好友系统、语音、真实装扮/道具交易、SNG 完整赛事、保位离座、降落伞、投诉和收藏后端流程。相关入口按 OpenSpec 要求作为禁用、占位或说明状态处理。

## 剩余风险

- 本轮浏览器验证使用单浏览器单用户主流程；多人实时同步仍由现有 socket/API 自动化和后端集成测试覆盖，没有做双浏览器并排人工观察。
- 商城仍是模拟充值语义，后续接真实支付时需要重新做支付风控、回调幂等和订单状态测试。
- 牌桌视觉当前按效果图优先实现，后续若加入真实头像、装扮或更长昵称，仍需持续做小屏重叠回归。
