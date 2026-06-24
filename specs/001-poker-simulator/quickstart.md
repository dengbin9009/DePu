# 快速开始：德州扑克完整牌局模拟器

> 当前文档描述第一版目标使用方式。代码实现完成后，命令需要与实际项目脚本保持一致。

## 前置条件

- Go 1.22 或更高版本
- Node.js 20 或更高版本
- SQLite 可由后端驱动自动创建本地数据库文件

## 启动后端

```bash
cd backend
go test ./...
go run ./cmd/depu-server
```

默认 API 地址：

```text
http://localhost:8080
```

## 启动前端

```bash
cd frontend
npm install
npm run dev
```

默认前端地址：

```text
http://localhost:5173
```

## 验证核心流程

1. 打开前端页面。
2. 创建一桌 4 人长牌牌局，设置按钮位、小盲和大盲。
3. 使用随机发牌开始牌局。
4. 按页面显示的合法动作推进到翻牌、转牌、河牌和摊牌。
5. 查看结算结果和行动历史。
6. 刷新页面后重新打开同一牌局，确认状态和历史仍可读取。
7. 创建一桌短牌调试牌局，指定 A-6-7-8-9 相关牌面，确认系统识别短牌顺子。

## 验证 API 契约

接口契约位于：

```text
specs/001-poker-simulator/contracts/openapi.yaml
```

实现阶段应确保后端响应结构与该文件一致。
