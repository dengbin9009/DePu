# multiplayer-poker 当前规格

## 目的

多人德州扑克 v1 必须在保留独立规则引擎测试页的前提下，支持真实账号登录、虚拟金币、房主建房、邀请码加入、同房间多人轮流操作、每手牌结果保存与展示。当前实现基线仍以 HTTP JSON API 为主要通信方式，正式多人页面通过周期性拉取房间和当前手牌状态来刷新界面。

## Requirements

### Requirement: 账号登录与唯一昵称

系统 MUST 支持用户名 + 密码注册登录，并为正式多人接口建立鉴权上下文。

#### Scenario: 注册成功

- **WHEN** 新用户提交未被占用的用户名、合法密码和未被占用的昵称
- **THEN** 系统创建用户、用户资料和钱包
- **AND** 返回不包含密码或密码散列的登录响应

#### Scenario: 唯一性冲突

- **WHEN** 用户使用重复用户名注册
- **THEN** 系统返回 `duplicate_username`
- **WHEN** 用户把昵称修改为已存在昵称
- **THEN** 系统返回 `duplicate_nickname`

### Requirement: 钱包与模拟充值

系统 MUST 为每个用户维护虚拟金币余额，提供至少 3 个服务端固定充值档位，并在确认后模拟充值成功。

#### Scenario: 模拟充值

- **WHEN** 已登录用户选择有效充值档位并确认
- **THEN** 系统增加钱包余额
- **AND** 写入 `recharge_simulated` 钱包流水

#### Scenario: 未登录访问钱包

- **WHEN** 未登录用户访问钱包或充值接口
- **THEN** 系统返回 `unauthorized`

### Requirement: 房主建房与邀请码加入

系统 MUST 支持房主创建房间、生成唯一邀请码、其他已登录用户通过邀请码加入，并在房间内入座或离座。

#### Scenario: 创建默认房间

- **WHEN** 房主创建房间且未显式指定座位数
- **THEN** 系统创建默认 6 人房间
- **AND** 默认最少 2 人可开局

#### Scenario: 邀请码加入

- **WHEN** 已登录用户提交有效邀请码
- **THEN** 系统将其加入房间成员列表

#### Scenario: 座位冲突

- **WHEN** 用户尝试占用已被其他用户占据的座位
- **THEN** 系统返回 `seat_taken`

### Requirement: 正式多人 HTTP 牌局基线

当前基线 MUST 允许房主通过 HTTP 发起正式手牌，允许当前行动玩家通过 HTTP 提交动作，并允许前端通过 HTTP 拉取当前房间和手牌状态。

#### Scenario: HTTP 开局

- **WHEN** 房主对 `/api/rooms/{roomId}/start` 发起 `POST`
- **THEN** 系统创建正式手牌状态
- **AND** 返回当前手牌快照

#### Scenario: HTTP 拉取当前手牌

- **WHEN** 已登录房间成员对 `/api/rooms/{roomId}/current-hand` 发起 `GET`
- **THEN** 系统返回当前手牌快照或明确的未找到错误

#### Scenario: HTTP 提交动作

- **WHEN** 当前行动玩家对 `/api/rooms/{roomId}/actions` 发起 `POST`
- **THEN** 系统校验用户与当前行动席位匹配
- **AND** 使用后端规则引擎应用动作
- **AND** 返回推进后的手牌快照

#### Scenario: 非当前玩家提交动作

- **WHEN** 非当前行动玩家尝试提交正式手牌动作
- **THEN** 系统返回 `not_your_turn`
- **AND** 不改变手牌状态

### Requirement: 每手牌结果与钱包一致性

系统 MUST 在一手牌结束时保存手牌结果、参与者输赢、昵称快照、钱包变更和钱包流水，并保持这些写入原子一致。

#### Scenario: 手牌结算归档

- **WHEN** 一手牌结束
- **THEN** 系统写入手牌结果
- **AND** 写入每名参与者的输赢明细
- **AND** 更新相关玩家钱包余额
- **AND** 写入钱包流水

#### Scenario: 历史昵称快照

- **WHEN** 用户在手牌结算后修改昵称
- **THEN** 历史手牌结果和个人战绩继续展示结算当时的昵称快照

### Requirement: 独立规则测试页

系统 MUST 保留独立规则引擎测试页，且测试页调试能力不得进入正式多人主流程。

#### Scenario: 测试页调试牌局

- **WHEN** 测试者通过规则测试页创建调试牌局、设置调试牌或执行只读回放
- **THEN** 系统继续按测试页语义工作
- **AND** 不影响正式多人房间状态

