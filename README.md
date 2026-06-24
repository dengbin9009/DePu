# DePu

DePu 是一个德州扑克完整牌局模拟器项目，第一版采用 Vue 前端和 Go 后端，面向本地单机练习、规则验证和牌局复盘。

## 第一版范围

- 支持长牌德州扑克和短牌德州扑克
- 支持随机洗牌发牌，也支持调试模式手动指定牌
- 支持座位、按钮位、盲注、行动顺序、下注轮、摊牌和结算
- 支持主池、边池、平分底池和行动历史
- 使用 SQLite 保存牌局快照、行动日志和回放数据
- 不包含登录、真钱、多房间、多人实时同步、AI 建议或 GTO 求解

## Spec Kit 文档

主要规格位于 [specs/001-poker-simulator/spec.md](/Users/dengbin/Code/github/DePu/specs/001-poker-simulator/spec.md)。

```text
specs/001-poker-simulator/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/openapi.yaml
└── tasks.md
```
