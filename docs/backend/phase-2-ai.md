# Phase 2：AI 增强开发文档

> **目标**：在 Phase 1 转写闭环基础上，用 adk-go + eino 从转写文本中自动识别身份、提取待办/承诺/笔记，并通过设备与提醒体系把价值推给用户。
> **完成标志**：系统能自动识别身份、提取结构化信息并产生提醒。

## 模块依赖关系

```
[1 AI 分析] ← 依赖 Phase1 的 STT + 身份
[2 设备管理] ← 独立，依赖认证
[3 提醒中心] ← 依赖 AI 分析
```

建议顺序：先 1（AI 分析）→ 3（提醒）→ 2（设备）。设备管理较独立可并行。

---

## 模块 1：AI 分析（adk-go + eino）

### 职责
把 STT 得到的转写文本交给 LLM，完成三件事：
1. **身份识别**：判断这段对话发生在用户哪个身份角色下（员工/父母/朋友…），返回 `identity_id` + 置信度。
2. **信息提取**：抽取结构化字段——待办（todos）、承诺（commitments）、重要笔记（notes）、关键事实。
3. **对话理解**：结合上下文，判断是否需要跨会话串联（Phase 2 先做单会话，上下文串联留 Phase 3）。

### 目录 / 文件
```
internal/service/ai.go        # AI 分析服务（adk-go + eino 编排）
internal/model/extraction.go  # 提取结果 struct
pkg/prompt/                   # prompt 模板集中管理（可选）
```

### 数据结构
```go
// model/extraction.go
type ExtractedData struct {
    Todos       []Todo        `json:"todos"`
    Commitments []Commitment  `json:"commitments"`
    Notes       []string      `json:"notes"`
}

type Todo struct {
    Text     string    `json:"text"`
    DueAt    *time.Time `json:"due_at,omitempty"`
}

type Commitment struct {
    Text     string `json:"text"`
    From     string `json:"from"`   // 谁承诺
    To       string `json:"to"`     // 对谁承诺
    DueAt    *time.Time `json:"due_at,omitempty"`
}

type AnalysisResult struct {
    IdentityID   string         `json:"identity_id,omitempty"`
    Confidence   float64        `json:"confidence"`   // 0-1
    Extracted    ExtractedData  `json:"extracted"`
    RawResponse  string         `json:"raw_response"`
}
```

### 接口签名
```go
type AIService struct { ... }

// Analyze 转写文本 + 用户身份列表 → 分析结果
func (s *AIService) Analyze(ctx, transcript string, identities []model.Identity) (*AnalysisResult, error)

// 内部拆分为两个阶段（eino 编排）：
//   1. 身份识别（分类，返回 identity_id + confidence）
//   2. 信息提取（结构化抽取，返回 todos/commitments/notes）
```

### 编排要点（eino）
- 用 eino 的 `Graph` 把「身份识别」和「信息提取」串成 pipeline，输出合并。
- 身份识别把用户已有身份列表作为候选标签传入，避免 LLM 自由发挥。
- 提示词要求 JSON 输出，解析失败时重试一次，再失败则该会话标记 `confidence=0` 降级为未分类。
- **置信度低于阈值（如 0.6）时**：不自动绑定身份，`identity_id` 留空，交给用户手动标注。

### 与 audio_sessions 的关联
分析结果写入 `extracted_data`（JSON 列）与 `identity_id`、`ai_confidence` 字段。

### 验收标准
- [ ] 给定一段「工作会议」转写，能识别到对应身份且置信度合理。
- [ ] 能从转写中抽出明确的待办和承诺，字段格式正确。
- [ ] LLM 返回非法 JSON 时不崩溃，会话不丢失（降级处理）。
- [ ] 低置信度结果不强制绑定身份。

---

## 模块 2：设备管理

### 职责
硬件设备的注册、绑定、状态上报、心跳、远程控制指令下发。

### 数据模型
```sql
CREATE TABLE devices (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(100) UNIQUE NOT NULL,   -- 硬件唯一标识
    name VARCHAR(100) NOT NULL,
    device_type VARCHAR(50) DEFAULT 'wearable',
    firmware_version VARCHAR(20),
    battery_level INTEGER,                     -- 0-100
    last_seen_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 目录 / 文件
```
internal/api/device.go
internal/service/device.go
internal/repository/device.go
internal/model/device.go
```

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/devices/bind | 用户绑定设备（输入 device_id + 绑定码） |
| GET  | /api/v1/devices | 当前用户设备列表 |
| GET  | /api/v1/devices/:id | 设备详情 |
| DELETE | /api/v1/devices/:id | 解绑 |
| POST | /api/v1/devices/:id/heartbeat | 心跳（硬件上报电量/版本） |
| POST | /api/v1/devices/:id/command | 下发指令（开始/暂停录音） |

### 设备认证说明
- 硬件上传音频不走用户名密码，用设备级密钥（`device_token`）签名。
- 设备绑定流程：前端生成绑定码 → 硬件首次连接时携带绑定码换取设备 token。

### 验收标准
- [ ] 设备可绑定/解绑，绑定码一次性有效。
- [ ] 心跳更新 last_seen_at 与电量。
- [ ] 指令可下发（Phase 2 先落库，硬件按需拉取）。

---

## 模块 3：提醒中心

### 职责
基于 AI 提取的待办/承诺，生成提醒；身份切换时提示「上次未完成事项」。

### 数据模型
```sql
CREATE TABLE reminders (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id CHAR(36) REFERENCES audio_sessions(id) ON DELETE CASCADE,
    identity_id CHAR(36) REFERENCES identities(id) ON DELETE SET NULL,
    type VARCHAR(20) NOT NULL,               -- todo / commitment / identity_switch
    content TEXT NOT NULL,
    due_at DATETIME,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending / done / dismissed
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 目录 / 文件
```
internal/api/reminder.go
internal/service/reminder.go
internal/repository/reminder.go
internal/model/reminder.go
```

### HTTP 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | /api/v1/reminders | 待处理提醒列表 |
| PUT  | /api/v1/reminders/:id/done | 标记完成 |
| PUT  | /api/v1/reminders/:id/dismiss | 忽略 |

### 触发逻辑
- AI 分析完成后，若 `extracted_data` 含 todos/commitments，自动生成对应 reminder。
- 身份切换（新的身份识别结果与上一条不同）时生成 `identity_switch` 提醒，内容引用该身份下未完成的待办。

### 验收标准
- [ ] 提取到待办后自动生成 reminder。
- [ ] 提醒可标记完成/忽略。
- [ ] 身份切换提醒引用正确身份下的未完成事项。

---

## Phase 2 整体验收清单

- [ ] AI 能识别身份并提取待办/承诺，低置信度降级不误绑。
- [ ] 设备可绑定、心跳、解绑。
- [ ] 待办/承诺自动生成提醒，身份切换有提醒。