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

- 当前仓库实现仍以 `001` 规则引擎与测试页能力为主
- `002` 目录中的 Spec Kit 文档用于指导后续多人版本实现
- 在 `002` 正式落地前，README 中的多人描述属于产品规划而非已完成能力

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
