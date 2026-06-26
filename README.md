# DePu

DePu 是一个德州扑克项目，当前目标是基于 Vue 前端和 Go 后端，逐步从规则引擎验证工具演进为**多人德州扑克对战 v1**。

当前规划重点包括：

- 保留独立规则引擎测试/回放页面
- 支持用户名登录、昵称管理与个人资料页
- 支持虚拟金币充值（模拟成功，不接真实支付）
- 支持房主建房、邀请码加入、多人同房间轮流操作
- 支持每手牌结果保存、房间历史和个人战绩展示
- 生产默认使用 MySQL，开发/测试允许 SQLite

## 当前版本范围

### 002：多人德州扑克对战 v1（当前主规划）

- 账号注册、登录与鉴权
- 全站唯一用户名与昵称
- 模拟金币充值与钱包流水
- 房主建房、邀请码加入、入座与离座
- 多真实账号同房间轮流推进正式牌局
- 每手牌结果归档、房间最近对局与个人战绩
- 保留独立规则测试页，不与正式多人流程混用
- 不包含真实支付、WebSocket 实时推送、大厅匹配、托管和机器人

### 001：完整牌局模拟器（历史规则引擎基线）

- 长牌与短牌德州扑克规则支持
- 随机发牌与调试指定牌
- 牌局推进、摊牌结算、主池边池和平分底池
- SQLite 保存牌局快照、行动历史和只读回放
- 面向本地规则验证、回归测试和牌局复盘

## Spec Kit 文档

### 当前主规格

主要规格位于 [specs/002-multiplayer-poker-v1/spec.md](/Users/dengbin/Code/github/DePu/specs/002-multiplayer-poker-v1/spec.md)。

```text
specs/002-multiplayer-poker-v1/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/openapi.yaml
└── tasks.md
```

### 历史规则引擎规格

规则引擎基线规格仍保留在 [specs/001-poker-simulator/spec.md](/Users/dengbin/Code/github/DePu/specs/001-poker-simulator/spec.md)。

## 本地启动

后端：

```bash
cd backend
go test ./...
go run ./cmd/depu-server
```

前端：

```bash
cd frontend
npm install
npm run dev
```

如果需要显式指定后端地址或数据库路径：

```bash
cd backend
DEPU_ADDR=:18080 DEPU_DB_PATH=/tmp/depu.sqlite go run ./cmd/depu-server
```

如果前端需要指向非默认后端地址：

```bash
cd frontend
DEPU_API_TARGET=http://localhost:18080 npm run dev
```

如果本机同时存在旧 Homebrew Node，建议显式使用 Node 20：

```bash
cd frontend
env PATH="$HOME/.nvm/versions/node/v20.19.4/bin:$PATH" npm run dev
```

## 当前实现说明

- 当前仓库同时保留 `001` 规则测试页能力与 `002` 多人 v1 的增量实现
- 规则测试页继续使用 `/api/rulesets`、`/api/games`、`/api/games/{id}/history`、`/api/games/{id}/replay` 等接口
- 正式多人流程使用独立的 `/api/auth/*`、`/api/me/*`、`/api/recharge*`、`/api/rooms*` 路由，不复用测试页调试能力
- 正式多人当前已支持：注册登录、昵称修改、模拟充值、建房/加入/入座/开局、轮流操作、房间最近牌局、个人战绩
- 当前多人版本仍为本地开发态，不包含真实支付、WebSocket 实时推送、大厅匹配、超时托管

### 已实现的多人页面体验

- 登录后可查看资料摘要：昵称、金币余额、总手数、总收益、最近对局时间
- 可查看钱包流水，区分“模拟充值 / 买入冻结 / 离座返还 / 牌局结算”
- 房间区可看到我的座位、买入金额、当前是否轮到我操作
- 房间历史可展示每手牌公共牌、赢家摘要、参与者投入/返奖/净输赢
- 个人战绩区可展示最近正式多人对局记录

## 数据库模式

### 默认开发模式：SQLite

后端默认使用本地 SQLite 文件，适合单机开发和规则测试：

```bash
cd backend
DEPU_DB_PATH=./data/depu.db go run ./cmd/depu-server
```

### 多人模式验证：MySQL

如果要验证 002 多人 v1 的生产目标数据库语义，可切换到 MySQL DSN。当前实现目标是业务语义一致，仅配置不同。

```bash
cd backend
DEPU_DB_DRIVER=mysql \
DEPU_DSN='root@tcp(127.0.0.1:3306)/depu_multiplayer?parseTime=true&multiStatements=true' \
go run ./cmd/depu-server
```

建议预先创建数据库：

```sql
CREATE DATABASE depu_multiplayer CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

## 002 手动验收提要

- 注册两个或以上账号，并保证用户名、昵称都唯一
- 登录后修改昵称，确认牌桌与资料页都展示昵称
- 执行模拟充值，确认余额增加且流水落库
- 房主建房并分享邀请码，其他账号加入并入座
- 房主开局后，按当前行动席位轮流提交动作
- 一手牌结束后检查房间最近牌局、个人战绩、钱包流水是否一致
- 确认规则测试页仍可独立创建测试牌局、设定调试牌与只读回放

## 当前本地验收建议

建议按下面顺序在本地快速验证当前分支能力：

1. 启动后端：`cd backend && go run ./cmd/depu-server`
2. 启动前端：`cd frontend && npm run dev`
3. 注册两个账号，例如 `owner01` / `player02`
4. 分别设置不同昵称，观察资料摘要是否更新
5. 给两个账号各做一次模拟充值，确认钱包流水生成
6. 房主建房，复制邀请码给第二个账号加入
7. 两个账号分别入座，确认会扣减买入冻结金额
8. 房主开局并轮流操作，确认页面提示“现在轮到我操作”只出现在当前玩家侧
9. 牌局结束后检查：
   - 房间最近牌局
   - 个人战绩
   - 钱包流水
   三者数据是否一致
10. 回到规则测试页，验证创建测试牌局、调试设牌、历史回放仍然正常

## 规则引擎 OpenAPI 创建请求示例

以下示例对应现有规则引擎测试路径的数据结构，而不是未来多人房间正式接口：

```json
{
  "rulesetId": "short-deck",
  "buttonSeat": 1,
  "bettingStructure": {
    "type": "ante",
    "ante": 10,
    "buttonBlind": 50
  },
  "dealMode": "random",
  "seats": [
    { "seatNo": 1, "name": "BTN", "stack": 1000 },
    { "seatNo": 2, "name": "A", "stack": 1000 },
    { "seatNo": 3, "name": "B", "stack": 1000 }
  ]
}
```
