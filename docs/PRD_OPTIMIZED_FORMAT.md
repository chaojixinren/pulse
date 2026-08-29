# Pulse · 拾笺 — 产品需求文档（PRD）

一句话定位：拾笺（Pulse）是一个以「身份」为组织单位的个人管理系统。通过可穿戴设备采集语音，云端 AI 自动识别当前身份，并提取待办、承诺、笔记，再按身份生成时间线与日报。

---

## 一、说明

- 架构：为「云端服务 + 明文存储 + 服务端 AI」（E2E / 端侧处理未实现）
- 中枢实体：为「音频会话 AudioSession」
- 身份识别：为「LLM 两阶段 + 置信度阈值 0.6」
- 待办 / 承诺：为「提取结果 JSON 存于音频会话」
- 认证：为「JWT + bcrypt + refresh token」
- 新增：设备管理（绑定码 / 心跳 / 指令）已在 Phase 2 落地

## 二、产品概述

产品做三件事：感知（判断你现在是谁）→ 承接（恢复该身份的上下文）→ 回照（告诉你时间与承诺的去向）。

- 语音采集：可穿戴硬件采集音频，经 HTTPS 上传
- 语音转写：服务端调用 StepFun StepAudio-2.5-ASR
- AI 身份识别：adk-go + OpenAI 兼容模型，从转写文本识别所属身份（返回 identity_id + confidence）
- 信息提取：从对话抽取待办（todos）、承诺（commitments）、笔记（notes）
- 身份管理：身份的增删改查、默认身份
- 时间线：按身份 / 时间查看语音会话
- 日报：按身份聚合会话时长与数量
- 设备管理：设备绑定、心跳、远程指令

## 三、系统架构（基于代码）

```
pulse/
├── backend/     Go 云端服务（Gin）
├── frontend/    React Web（当前仅脚手架）
├── hardware/    ESP32-S3 可穿戴设备
└── docs/        设计文档
```

- 后端：Go 1.26.5 / Gin
- AI SDK：adk-go v2（openaimodel，OpenAI 兼容接口）
- STT：StepFun StepAudio-2.5-ASR（SSE）
- 数据库：MySQL 8.0+（音频二进制直接存 LONGBLOB）
- 缓存：Redis 7+
- 认证：JWT（access 1h）+ refresh token（7 天，仅存哈希）
- 硬件：ESP32-S3 / FreeRTOS / LVGL

### 分层约定

- api（handler）：参数解析、校验、调用 service、统一响应
- service：业务逻辑、事务边界、调用外部服务（stt / ai）
- repository：纯 SQL 访问
- model：struct 定义，对应数据库表
- worker：后台「转写 → AI 分析」流水线（5 秒轮询）
- middleware：auth / logger / cors / error_handler

## 四、数据模型（基于 migrations）

**users**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | CHAR(36) | 主键，UUID |
| email | VARCHAR(255) | 邮箱，唯一 |
| password_hash | VARCHAR(255) | bcrypt 哈希 |
| name | VARCHAR(100) | 用户昵称 |
| settings | JSON | 用户配置 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

**refresh_tokens**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | CHAR(36) | 主键 |
| user_id | CHAR(36) | 关联用户 |
| token_hash | VARCHAR(255) | 哈希存储 |
| expires_at | DATETIME | 过期时间 |
| revoked_at | DATETIME | 吊销时间（可选） |

**identities**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | CHAR(36) | 主键 |
| user_id | CHAR(36) | 所属用户 |
| name | VARCHAR(100) | 身份名称（如"工作"、"家庭"） |
| description | TEXT | 身份描述 |
| color | VARCHAR(7) | 标识颜色（如 #3B82F6） |
| icon | VARCHAR(50) | 图标标识 |
| is_default | BOOLEAN | 是否默认身份 |
| created_at | DATETIME | 创建时间 |

**audio_sessions**（中枢实体）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | CHAR(36) | 主键 |
| user_id | CHAR(36) | 所属用户 |
| identity_id | CHAR(36) | 关联身份（AI 识别后填充） |
| device_id | CHAR(36) | 来源设备 |
| audio_data | LONGBLOB | 音频二进制数据 |
| transcript | TEXT | 转写文本 |
| duration | INT | 时长（秒） |
| status | ENUM | pending/processing/completed/failed |
| extracted_data | JSON | 提取的待办/承诺/笔记 |
| ai_confidence | FLOAT | AI 置信度 [0, 1] |
| created_at | DATETIME | 创建时间 |

**devices**

| 字段 | 类型 | 说明 |
|------|------|------|
| device_id | VARCHAR(100) | 设备唯一标识 |
| user_id | CHAR(36) | 所属用户 |
| name | VARCHAR(100) | 设备名称 |
| device_type | VARCHAR(50) | 设备类型 |
| firmware_version | VARCHAR(20) | 固件版本 |
| battery_level | INT | 电量百分比 |
| last_seen_at | DATETIME | 最后在线时间 |

**device_bind_codes**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | VARCHAR(10) | 6 位数字绑定码 |
| user_id | CHAR(36) | 所属用户 |
| expires_at | DATETIME | 过期时间（10 分钟） |
| used_at | DATETIME | 使用时间（一次性） |

**device_commands**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | CHAR(36) | 主键 |
| device_id | VARCHAR(100) | 目标设备 |
| command | VARCHAR(50) | 指令类型（start/stop_recording） |
| status | ENUM | pending/sent/executed |
| created_at | DATETIME | 创建时间 |

**核心实体说明**：

- 中枢实体是 audio_sessions（音频会话）：一段音频一条记录
- 身份识别结果与提取结果都写回 audio_sessions 的 identity_id / ai_confidence / extracted_data 字段
- 没有独立的待办表、没有提醒表、没有端到端加密字段

## 五、核心业务流程

### 流程 1：音频上传与处理

```
硬件录音 → POST /api/v1/audio/upload → audio_sessions (status=pending)
   ↓
worker 每 5 秒轮询 pending
   ↓
认领 (pending→processing) → STT 转写 (StepFun)
   ↓
写 transcript → status=completed
   ↓
AI 分析（若配置）：拉取身份列表 → 身份识别 → 信息提取
   ↓
写回 identity_id / ai_confidence / extracted_data
```

**关键设计**：

- 音频上传即落库（明文 LONGBLOB），状态机 pending→processing→completed / failed（failed 可重试）
- 转写成功即置 completed；AI 分析是转写之后的附加步骤，失败仅记日志、不影响会话
- AI 身份识别把用户已有身份作为候选标签传入，避免自由发挥

### 流程 2：AI 身份识别与信息提取

两阶段编排：先身份识别，再信息提取。

**阶段 1：身份识别**

```
候选身份列表 + 转写文本 → {identity_id, confidence}
```

**阶段 2：信息提取**

```
转写文本 → {todos, commitments, notes}
```

**校验规则**：

- 阈值 AI_CONFIDENCE_THRESHOLD（默认 0.6）：confidence ≥ 阈值且 identity_id 非空才绑定身份，否则留空交用户手动标注
- LLM 返回非法 JSON 时重试一次，再失败降级为 confidence=0、不绑定身份
- 返回的 identity_id 必须在校验候选列表内，否则作废
- 置信度截断到 [0, 1]

**提取字段**：

- todos：text, due_at
- commitments：text, from, to, due_at
- notes：string[]

## 六、API 接口清单（基于 router）

**认证**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/register | 用户注册 |
| POST | /api/v1/auth/login | 用户登录 |
| POST | /api/v1/auth/refresh | 刷新 token |
| POST | /api/v1/auth/logout | 登出 |
| GET | /api/v1/auth/me | 当前用户信息 |

**语音**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/audio/upload | 上传音频 |
| POST | /api/v1/audio/:id/retry | 重试失败的转写 |

**身份**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/identities | 获取身份列表 |
| POST | /api/v1/identities | 创建身份 |
| PUT | /api/v1/identities/:id | 更新身份 |
| DELETE | /api/v1/identities/:id | 删除身份 |
| PUT | /api/v1/identities/:id/default | 设为默认身份 |

**时间线**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/timeline | 按身份/时间查看语音会话 |

**报告**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/reports/daily?date= | 按日期获取日报 |

**设备**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/devices/bind-code | 生成绑定码 |
| POST | /api/v1/devices/bind | 绑定设备 |
| GET | /api/v1/devices | 设备列表 |
| GET | /api/v1/devices/:id | 设备详情 |
| DELETE | /api/v1/devices/:id | 解绑设备 |
| POST | /api/v1/devices/:id/heartbeat | 设备心跳 |
| POST | /api/v1/devices/:id/command | 下发指令 |

## 七、设备管理（Phase 2 已实现）

- 生成绑定码：6 位数字，10 分钟有效，一次性
- 绑定：device_id + 绑定码换取设备 token（token 只返回一次，库中仅存哈希）
- 心跳：更新 last_seen_at、电量、固件版本
- 指令：start_recording / stop_recording（先落库，硬件按需拉取）
- 解绑：删除设备记录

## 八、功能现状与差距

### 已实现

- 认证（注册/登录/刷新/登出）
- 音频上传、存储、转写、重试
- AI 身份识别 + 信息提取
- 身份 CRUD + 默认身份
- 时间线查询
- 日报聚合
- 设备绑定 / 心跳 / 指令

### 规划中

- 端到端加密（当前音频明文存 MySQL）
- 端侧处理（当前全部在服务端）
- 三级识别链 / 置信度分档交互（当前单阈值）
- 独立待办系统、提醒隔离、记忆恢复卡
- 切换过渡、冲突协调、统计周报、数据导出删除
- 前端页面（当前 frontend 仅有脚手架）
- 硬件（ESP32-S3 固件原型）

## 九、安全与工程现状

### 已实现

- 密码 bcrypt 哈希；JWT + refresh token（哈希存库、可吊销）
- 接口鉴权：middleware.Auth 校验 Bearer token
- 音频格式与魔数校验、大小上限（默认 50MB）
- CORS、统一错误响应、结构化日志（zap）
- CI：gofmt + go vet + 单测/竞态 + Phase1/Phase2 e2e 矩阵

### 待补

- 限流（middleware/ratelimit 规划中）
- 音频加密存储（当前明文）
- 数据导出 / 删除接口

## 十、路线图

**Phase 1（已实现）**: 认证、音频上传、MySQL 存储、STT、身份、时间线、日报

**Phase 2（已实现）**: AI 分析（身份识别/信息提取）、设备管理

**Phase 3（规划中）**: 报告增强、加密存储、数据删除/导出、限流配额、可观测性、部署

**前端（待启动）**: React + TypeScript 页面

**硬件（待启动）**: ESP32-S3 固件原型
