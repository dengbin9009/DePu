# 实施计划：德州扑克完整牌局模拟器

**分支**: `001-poker-simulator` | **日期**: 2026-06-24 | **规格**: [spec.md](/Users/dengbin/Code/github/DePu/specs/001-poker-simulator/spec.md)

**输入**: 来自 `/specs/001-poker-simulator/spec.md` 的功能规格。

## 摘要

构建一个 Vue + Go 的本地单机德州扑克完整牌局模拟器。Go 后端作为规则权威，负责长牌/短牌规则集、牌局状态机、合法动作校验、牌型评估、主池/边池和 SQLite 持久化；Vue 前端负责牌桌可视化、动作控制台、调试指定牌、历史记录和回放。

## 技术上下文

**语言/版本**: Go 1.22+；TypeScript 5+；Vue 3。

**主要依赖**: Go 标准库 HTTP 路由或轻量路由器；SQLite 驱动；Vue 3；Vite；Pinia 或等价轻量状态管理。

**存储**: SQLite，本地数据库文件保存牌局、动作日志、快照和回放数据。

**测试**: Go `testing`；API 集成测试；前端组件测试；端到端冒烟测试。

**目标平台**: 本地开发环境，浏览器访问 Vue 前端，Go 后端提供 HTTP JSON API。

**项目类型**: Web 应用，前后端分离但同仓库管理。

**性能目标**: 单手牌动作提交在本机 p95 小于 200ms；回放 200 个动作以内的牌局在 1 秒内返回快照。

**约束**: v1 不做登录、真钱、多房间、多人实时同步或 AI 决策；所有规则判定必须在后端完成。

**规模/范围**: 2 到 10 人单桌；长牌 52 张和短牌 36 张；本地单用户操作。

## 宪章检查

- 规则正确性优先：通过后端规则引擎和 Go 测试满足。
- 长牌与短牌规则集可配置：通过 `RuleSet` 实体和独立规则包满足。
- 测试优先覆盖高风险规则：任务清单将先写规则测试和 API 测试。
- 单机本地边界：规格明确排除账户、真钱和多人同步。
- 可复盘、可解释、可恢复：通过行动日志、快照和回放 API 满足。

## 项目结构

### 文档结构

```text
specs/001-poker-simulator/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── openapi.yaml
├── tasks.md
└── spec.md
```

### 源码结构

```text
backend/
├── cmd/
│   └── depu-server/
│       └── main.go
├── internal/
│   ├── api/
│   ├── game/
│   ├── rules/
│   ├── handeval/
│   ├── pot/
│   └── storage/
└── tests/
    ├── contract/
    ├── integration/
    └── unit/

frontend/
├── src/
│   ├── api/
│   ├── components/
│   ├── pages/
│   ├── stores/
│   └── types/
└── tests/
    ├── component/
    └── e2e/

data/
└── .gitkeep
```

**结构决策**: 使用同仓库前后端分离结构。后端 `internal/` 按领域拆分，避免 API 层混入规则。前端以页面、组件、状态和 API 客户端分层。

## 复杂度跟踪

当前设计没有违反项目宪章。短牌和回放增加了复杂度，但它们是用户明确选择的 v1 范围，并通过规则集和行动日志隔离。
