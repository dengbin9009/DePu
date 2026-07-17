# table-playability-hardening

加固正式多人牌桌的可玩性、房间生命周期、牌谱完整性、实时恢复与生产级验收

## 多账号测试数据隔离

- Go API 与 storage 集成测试通过 `backend/internal/testmysql` 为每个测试创建独立 MySQL database，并在 `t.Cleanup` 阶段删除；`DEPU_TEST_MYSQL_ADMIN_DSN` 用于提供管理连接，`DEPU_TEST_MYSQL_DSN` 仅作为兼容的管理连接来源，不再表示复用固定业务 database。
- 多账号浏览器或手动验收使用 `scripts/with-test-mysql.sh <command...>` 启动。包装器创建包含 `DEPU_TEST_RUN_ID` 的临时 database，向子进程注入 `DEPU_DB_DRIVER=mysql`、`DEPU_DSN` 和 `DEPU_TEST_DATABASE`，命令退出后删除整个 database。
- 外部测试数据仍需命名时使用 `DEPU_TEST_RUN_ID` 作为用户名、昵称或产物目录后缀；不得连接 `depu_multiplayer` 等开发/生产 database 执行清理 SQL。
- Loop 验证必须真实连接 MySQL；创建、连接或删除临时 database 失败均视为失败，不得以 `SKIP` 作为通过。
