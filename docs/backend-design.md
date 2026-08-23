# Pulse 后端设计文档

## 1. 系统架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      可穿戴硬件设备                            │
│                   (语音采集 → WiFi 上传)                       │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTPS
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                      负载均衡 (Nginx/ALB)                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                      Gin Web Server (Go)                      │
│  ┌──────────────┬──────────────┬──────────────────────────┐ │
│  │ 认证中间件    │  日志中间件   │  限流中间件               │ │
│  └──────────────┴──────────────┴──────────────────────────┘ │
│  ┌──────────────┬──────────────┬──────────────────────────┐ │
│  │  Auth API    │  Audio API   │  Identity API            │ │
│  ├──────────────┼──────────────┼──────────────────────────┤ │
│  │ Report API   │  Device API  │  Timeline API            │ │
│  └──────────────┴──────────────┴──────────────────────────┘ │
└──────────────────────┬──────────────────────────────────────┘
                       │
       ┌───────────────┼───────────────┬─────────────────┐
       │               │               │                 │
       ↓               ↓               ↓                 ↓
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌──────────────┐
│  PostgreSQL │ │   Redis     │ │    S3/OSS   │ │  AI Service  │
│  (主数据库)  │ │   (缓存)    │ │  (语音存储)  │ │(adk-go+eino) │
└─────────────┘ └─────────────┘ └─────────────┘ └──────────────┘
                                                       │
                                                       ↓
                                              ┌──────────────────┐
                                              │  StepFun STT API │
                                              │ (StepAudio-2.5)  │
                                              └──────────────────┘
```

### 1.2 分层架构

```
┌─────────────────────────────────────────┐
│          API Layer (Gin Router)         │  HTTP 请求处理、参数验证
├─────────────────────────────────────────┤
│        Middleware Layer                 │  认证、日志、CORS、限流
├─────────────────────────────────────────┤
│         Service Layer                   │  业务逻辑处理
│  - AuthService                          │
│  - AudioService                         │
│  - AIService (adk-go + eino)            │
│  - IdentityService                      │
│  - ReportService                        │
├─────────────────────────────────────────┤
│         Repository Layer                │  数据访问层
│  - UserRepository                       │
│  - AudioSessionRepository               │
│  - IdentityRepository                   │
├─────────────────────────────────────────┤
│        Infrastructure Layer             │  基础设施
│  - Database (PostgreSQL)                │
│  - Cache (Redis)                        │
│  - Storage (S3/OSS)                     │
│  - AI Client (adk-go + eino)            │
└─────────────────────────────────────────┘
```

### 1.3 核心模块

#### 1.3.1 认证模块 (Auth)
- JWT Token 认证
- 用户注册/登录
- Token 刷新机制

#### 1.3.2 语音处理模块 (Audio)
- 语音文件上传
- S3/OSS 存储
- STT 转换（StepFun API）
- 语音会话管理

#### 1.3.3 AI 分析模块 (AI Service)
- 使用 adk-go + eino SDK
- 身份识别
- 信息提取（待办、笔记、承诺）
- 对话上下文理解

#### 1.3.4 身份管理模块 (Identity)
- 身份 CRUD 操作
- 身份切换检测
- 身份统计分析

#### 1.3.5 报告生成模块 (Report)
- 日报/周报生成
- 时间统计
- 数据可视化

## 2. API 设计

### 2.1 认证 API

#### 用户注册
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123",
  "name": "张三"
}

Response 201:
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "张三",
    "created_at": "2024-08-23T10:00:00Z"
  },
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 3600
}
```

#### 用户登录
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123"
}

Response 200:
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 3600
}
```

#### 刷新 Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGc..."
}

Response 200:
{
  "access_token": "eyJhbGc...",
  "expires_in": 3600
}
```

### 2.2 语音处理 API

#### 上传语音文件
```http
POST /api/v1/audio/upload
Authorization: Bearer <access_token>
Content-Type: multipart/form-data

file: <audio_file.wav>
device_id: "device-uuid"
duration: 120
recorded_at: "2024-08-23T10:00:00Z"

Response 201:
{
  "session_id": "uuid",
  "audio_url": "https://s3.../audio.wav",
  "status": "processing",
  "created_at": "2024-08-23T10:00:00Z"
}
```

#### 获取语音会话
```http
GET /api/v1/audio/sessions/:session_id
Authorization: Bearer <access_token>

Response 200:
{
  "id": "uuid",
  "user_id": "uuid",
  "audio_url": "https://s3.../audio.wav",
  "transcript": "今天需要完成项目报告...",
  "duration": 120,
  "status": "completed",
  "identity_id": "uuid",
  "identity_name": "员工",
  "extracted_data": {
    "todos": ["完成项目报告", "发送邮件给客户"],
    "notes": ["会议讨论了新功能"],
    "commitments": ["下周一前完成"]
  },
  "recorded_at": "2024-08-23T10:00:00Z",
  "created_at": "2024-08-23T10:00:00Z",
  "updated_at": "2024-08-23T10:02:00Z"
}
```

#### 获取语音会话列表
```http
GET /api/v1/audio/sessions?page=1&limit=20&identity_id=uuid
Authorization: Bearer <access_token>

Response 200:
{
  "sessions": [
    {
      "id": "uuid",
      "duration": 120,
      "transcript": "今天需要完成...",
      "identity_name": "员工",
      "recorded_at": "2024-08-23T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

### 2.3 身份管理 API

#### 获取身份列表
```http
GET /api/v1/identities
Authorization: Bearer <access_token>

Response 200:
{
  "identities": [
    {
      "id": "uuid",
      "name": "员工",
      "description": "工作场景身份",
      "color": "#FF5722",
      "icon": "briefcase",
      "is_default": true,
      "session_count": 45,
      "total_duration": 5400,
      "created_at": "2024-08-23T10:00:00Z"
    }
  ]
}
```

#### 创建身份
```http
POST /api/v1/identities
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "父母",
  "description": "家庭场景身份",
  "color": "#4CAF50",
  "icon": "home",
  "is_default": false
}

Response 201:
{
  "id": "uuid",
  "name": "父母",
  "description": "家庭场景身份",
  "color": "#4CAF50",
  "icon": "home",
  "is_default": false,
  "created_at": "2024-08-23T10:00:00Z"
}
```

#### 更新身份
```http
PUT /api/v1/identities/:id
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "父母（家庭）",
  "description": "更新后的描述"
}

Response 200:
{
  "id": "uuid",
  "name": "父母（家庭）",
  "description": "更新后的描述",
  "updated_at": "2024-08-23T11:00:00Z"
}
```

#### 删除身份
```http
DELETE /api/v1/identities/:id
Authorization: Bearer <access_token>

Response 204 No Content
```

### 2.4 报告 API

#### 获取日报
```http
GET /api/v1/reports/daily?date=2024-08-23
Authorization: Bearer <access_token>

Response 200:
{
  "date": "2024-08-23",
  "summary": {
    "total_sessions": 12,
    "total_duration": 3600,
    "identities_used": ["员工", "父母", "朋友"]
  },
  "by_identity": [
    {
      "identity_id": "uuid",
      "identity_name": "员工",
      "session_count": 8,
      "duration": 2400,
      "todos": ["完成报告", "发送邮件"],
      "notes": ["项目进展顺利"],
      "time_distribution": [
        {"hour": 9, "duration": 300},
        {"hour": 10, "duration": 600}
      ]
    }
  ],
  "timeline": [
    {
      "time": "09:00",
      "identity": "员工",
      "summary": "讨论项目进展"
    }
  ]
}
```

#### 获取周报
```http
GET /api/v1/reports/weekly?week=2024-W34
Authorization: Bearer <access_token>

Response 200:
{
  "week": "2024-W34",
  "date_range": {
    "start": "2024-08-19",
    "end": "2024-08-25"
  },
  "summary": {
    "total_sessions": 67,
    "total_duration": 18000,
    "avg_daily_duration": 2571
  },
  "by_identity": [...],
  "daily_breakdown": [...]
}
```

## 3. 数据库设计

### 3.1 数据库 Schema

#### users 表（用户表）
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    avatar_url TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

#### identities 表（身份表）
```sql
CREATE TABLE identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#000000',
    icon VARCHAR(50) DEFAULT 'person',
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_identities_user_id ON identities(user_id);
CREATE INDEX idx_identities_deleted_at ON identities(deleted_at);
CREATE UNIQUE INDEX idx_identities_user_default ON identities(user_id, is_default) 
    WHERE is_default = TRUE AND deleted_at IS NULL;
```

#### audio_sessions 表（语音会话表）
```sql
CREATE TABLE audio_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    identity_id UUID REFERENCES identities(id) ON DELETE SET NULL,
    device_id VARCHAR(100),
    audio_url TEXT NOT NULL,
    transcript TEXT,
    duration INTEGER NOT NULL, -- 秒
    file_size BIGINT, -- 字节
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed
    error_message TEXT,
    extracted_data JSONB DEFAULT '{}', -- todos, notes, commitments
    ai_confidence DECIMAL(3,2), -- AI 识别置信度 0.00-1.00
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audio_sessions_user_id ON audio_sessions(user_id);
CREATE INDEX idx_audio_sessions_identity_id ON audio_sessions(identity_id);
CREATE INDEX idx_audio_sessions_recorded_at ON audio_sessions(recorded_at DESC);
CREATE INDEX idx_audio_sessions_status ON audio_sessions(status);
```

#### devices 表（设备表）
```sql
CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    device_type VARCHAR(50) DEFAULT 'wearable',
    firmware_version VARCHAR(20),
    battery_level INTEGER, -- 0-100
    last_seen_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE INDEX idx_devices_device_id ON devices(device_id);
```

#### refresh_tokens 表（刷新令牌表）
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

### 3.2 数据库迁移示例

```go
// migrations/001_create_users_table.go
package migrations

import (
    "database/sql"
)

func Up_001(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            email VARCHAR(255) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            name VARCHAR(100) NOT NULL,
            avatar_url TEXT,
            settings JSONB DEFAULT '{}',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP WITH TIME ZONE
        );

        CREATE INDEX idx_users_email ON users(email);
        CREATE INDEX idx_users_deleted_at ON users(deleted_at);
    `)
    return err
}

func Down_001(db *sql.DB) error {
    _, err := db.Exec(`DROP TABLE IF EXISTS users;`)
    return err
}
```

## 4. AI 服务集成方案

### 4.1 adk-go + eino 架构

```
┌──────────────────────────────────────────────────────┐
│              Pulse Backend (Go)                      │
│                                                      │
│  ┌────────────────────────────────────────────┐    │
│  │           AIService                        │    │
│  │                                            │    │
│  │  ┌──────────────────────────────────────┐ │    │
│  │  │       adk-go Client                  │ │    │
│  │  │  (AI Orchestration & Prompt Mgmt)   │ │    │
│  │  └──────────────┬───────────────────────┘ │    │
│  │                 │                          │    │
│  │                 ↓                          │    │
│  │  ┌──────────────────────────────────────┐ │    │
│  │  │        eino Framework                │ │    │
│  │  │  (Multi-step AI Workflow Engine)     │ │    │
│  │  └──────────────┬───────────────────────┘ │    │
│  │                 │                          │    │
│  └─────────────────┼──────────────────────────┘    │
└───────────────────┼───────────────────────────────┘
                    │
                    ↓
        ┌───────────────────────────┐
        │   StepFun API (STT)       │
        │   StepAudio-2.5-ASR       │
        └───────────────────────────┘
```

### 4.2 AI 服务实现示例

```go
// internal/service/ai_service.go
package service

import (
    "context"
    "fmt"
    
    "github.com/chaojixinren/pulse/internal/model"
    "github.com/volcengine/adk-go/adk"
    "github.com/volcengine/eino/flow"
)

type AIService struct {
    adkClient *adk.Client
    einoFlow  *flow.Flow
}

func NewAIService(apiKey, baseURL string) (*AIService, error) {
    // 初始化 adk-go 客户端
    client := adk.NewClient(adk.Config{
        APIKey:  apiKey,
        BaseURL: baseURL,
    })

    // 创建 eino 工作流
    flowConfig := flow.NewFlow(
        flow.WithSteps(
            flow.Step{
                Name: "transcribe",
                Type: "stt",
            },
            flow.Step{
                Name: "analyze",
                Type: "llm",
            },
            flow.Step{
                Name: "extract",
                Type: "llm",
            },
        ),
    )

    return &AIService{
        adkClient: client,
        einoFlow:  flowConfig,
    }, nil
}

// TranscribeAudio 语音转文字
func (s *AIService) TranscribeAudio(ctx context.Context, audioURL string) (string, error) {
    req := adk.STTRequest{
        AudioURL: audioURL,
        Model:    "stepfun/stepaudio-2.5-asr",
        Language: "zh",
    }

    resp, err := s.adkClient.STT(ctx, req)
    if err != nil {
        return "", fmt.Errorf("STT failed: %w", err)
    }

    return resp.Text, nil
}

// IdentifyContext 识别身份和上下文
func (s *AIService) IdentifyContext(ctx context.Context, transcript string, userIdentities []model.Identity) (*ContextAnalysis, error) {
    // 构建身份列表提示词
    identityPrompt := "用户定义的身份:\n"
    for _, identity := range userIdentities {
        identityPrompt += fmt.Sprintf("- %s: %s\n", identity.Name, identity.Description)
    }

    prompt := fmt.Sprintf(`
分析以下对话内容，识别说话者当前的身份角色。

%s

对话内容:
%s

请以 JSON 格式返回分析结果:
{
  "identity": "识别的身份名称",
  "confidence": 0.95,
  "reasoning": "判断理由"
}
`, identityPrompt, transcript)

    req := adk.ChatRequest{
        Model: "doubao-pro",
        Messages: []adk.Message{
            {
                Role:    "system",
                Content: "你是一个专业的对话分析助手，擅长从对话内容识别说话者的身份角色。",
            },
            {
                Role:    "user",
                Content: prompt,
            },
        },
        ResponseFormat: adk.ResponseFormat{Type: "json_object"},
    }

    resp, err := s.adkClient.Chat(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("identity analysis failed: %w", err)
    }

    var analysis ContextAnalysis
    if err := json.Unmarshal([]byte(resp.Content), &analysis); err != nil {
        return nil, fmt.Errorf("failed to parse analysis: %w", err)
    }

    return &analysis, nil
}

// ExtractInformation 提取待办事项、笔记、承诺
func (s *AIService) ExtractInformation(ctx context.Context, transcript string) (*ExtractedData, error) {
    prompt := fmt.Sprintf(`
从以下对话中提取关键信息:

对话内容:
%s

请以 JSON 格式返回:
{
  "todos": ["待办事项1", "待办事项2"],
  "notes": ["重要笔记1", "重要笔记2"],
  "commitments": ["承诺1", "承诺2"]
}

提取规则:
- todos: 需要完成的任务、行动项
- notes: 重要的信息、观察、想法
- commitments: 对他人的承诺、约定
`, transcript)

    req := adk.ChatRequest{
        Model: "doubao-pro",
        Messages: []adk.Message{
            {
                Role:    "system",
                Content: "你是一个专业的信息提取助手。",
            },
            {
                Role:    "user",
                Content: prompt,
            },
        },
        ResponseFormat: adk.ResponseFormat{Type: "json_object"},
    }

    resp, err := s.adkClient.Chat(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("information extraction failed: %w", err)
    }

    var data ExtractedData
    if err := json.Unmarshal([]byte(resp.Content), &data); err != nil {
        return nil, fmt.Errorf("failed to parse extracted data: %w", err)
    }

    return &data, nil
}

type ContextAnalysis struct {
    Identity   string  `json:"identity"`
    Confidence float64 `json:"confidence"`
    Reasoning  string  `json:"reasoning"`
}

type ExtractedData struct {
    Todos       []string `json:"todos"`
    Notes       []string `json:"notes"`
    Commitments []string `json:"commitments"`
}
```

### 4.3 使用 eino 的多步骤工作流

```go
// internal/service/audio_workflow.go
package service

import (
    "context"
    
    "github.com/volcengine/eino/flow"
)

// ProcessAudioWorkflow 完整的语音处理工作流
func (s *AIService) ProcessAudioWorkflow(ctx context.Context, audioURL string, userIdentities []model.Identity) (*WorkflowResult, error) {
    // 定义工作流
    workflow := flow.NewFlow(
        flow.WithContext(ctx),
        flow.WithSteps(
            // Step 1: 语音转文字
            flow.Step{
                Name: "transcribe",
                Func: func(ctx context.Context, input flow.Input) (flow.Output, error) {
                    transcript, err := s.TranscribeAudio(ctx, input.Get("audio_url").(string))
                    return flow.Output{"transcript": transcript}, err
                },
            },
            // Step 2: 识别身份
            flow.Step{
                Name: "identify",
                Func: func(ctx context.Context, input flow.Input) (flow.Output, error) {
                    transcript := input.Get("transcript").(string)
                    analysis, err := s.IdentifyContext(ctx, transcript, userIdentities)
                    return flow.Output{"analysis": analysis}, err
                },
            },
            // Step 3: 提取信息
            flow.Step{
                Name: "extract",
                Func: func(ctx context.Context, input flow.Input) (flow.Output, error) {
                    transcript := input.Get("transcript").(string)
                    data, err := s.ExtractInformation(ctx, transcript)
                    return flow.Output{"extracted_data": data}, err
                },
            },
        ),
    )

    // 执行工作流
    result, err := workflow.Run(flow.Input{
        "audio_url": audioURL,
    })
    if err != nil {
        return nil, err
    }

    return &WorkflowResult{
        Transcript:    result.Get("transcript").(string),
        Analysis:      result.Get("analysis").(*ContextAnalysis),
        ExtractedData: result.Get("extracted_data").(*ExtractedData),
    }, nil
}

type WorkflowResult struct {
    Transcript    string
    Analysis      *ContextAnalysis
    ExtractedData *ExtractedData
}
```

## 5. 核心业务流程

### 5.1 语音上传与处理流程

```
┌─────────┐
│ 硬件设备 │
└────┬────┘
     │ 1. 上传语音文件
     ↓
┌─────────────────────────────────────────┐
│           POST /api/v1/audio/upload     │
│                                         │
│  1. 验证用户认证                         │
│  2. 验证文件格式 (WAV/MP3/M4A)           │
│  3. 生成唯一 session_id                 │
│  4. 上传文件到 S3/OSS                    │
│  5. 创建 audio_session 记录 (pending)   │
│  6. 返回 session_id                     │
└────┬────────────────────────────────────┘
     │
     │ 2. 异步处理（后台任务）
     ↓
┌─────────────────────────────────────────┐
│       Background Audio Processor        │
│                                         │
│  1. 从队列获取待处理任务                 │
│  2. 更新状态为 processing               │
│  3. 调用 StepFun STT API               │
│  4. 获取转录文本                        │
│  5. 调用 AI 分析（adk-go + eino）       │
│     - 识别身份                          │
│     - 提取信息                          │
│  6. 更新 audio_session 记录             │
│     - transcript                        │
│     - identity_id                       │
│     - extracted_data                    │
│     - status = completed                │
│  7. 缓存结果到 Redis                    │
└─────────────────────────────────────────┘
     │
     │ 3. 通知客户端
     ↓
┌─────────────────────────────────────────┐
│    WebSocket / Polling                  │
│    通知客户端处理完成                     │
└─────────────────────────────────────────┘
```

### 5.2 实现示例

#### 5.2.1 语音上传处理器

```go
// internal/api/audio_handler.go
package api

import (
    "net/http"
    "time"

    "github.com/chaojixinren/pulse/internal/service"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type AudioHandler struct {
    audioService   *service.AudioService
    storageService *service.StorageService
}

func (h *AudioHandler) UploadAudio(c *gin.Context) {
    userID := c.GetString("user_id") // 从 JWT 中间件获取

    // 1. 接收文件
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
        return
    }

    // 2. 验证文件类型
    if !isValidAudioFile(file.Filename) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的音频格式"})
        return
    }

    // 3. 读取其他参数
    deviceID := c.PostForm("device_id")
    duration := c.PostForm("duration")
    recordedAt := c.PostForm("recorded_at")

    // 4. 生成 session ID
    sessionID := uuid.New()

    // 5. 上传到 S3/OSS
    audioURL, err := h.storageService.Upload(c.Request.Context(), file, sessionID.String())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "存储失败"})
        return
    }

    // 6. 创建数据库记录
    session := &model.AudioSession{
        ID:         sessionID,
        UserID:     uuid.MustParse(userID),
        DeviceID:   deviceID,
        AudioURL:   audioURL,
        Duration:   parseDuration(duration),
        FileSize:   file.Size,
        Status:     "pending",
        RecordedAt: parseTime(recordedAt),
    }

    if err := h.audioService.CreateSession(c.Request.Context(), session); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
        return
    }

    // 7. 添加到处理队列（异步）
    go h.audioService.ProcessAudioAsync(sessionID)

    // 8. 返回结果
    c.JSON(http.StatusCreated, gin.H{
        "session_id": sessionID,
        "audio_url":  audioURL,
        "status":     "processing",
        "created_at": time.Now(),
    })
}

func isValidAudioFile(filename string) bool {
    validExts := []string{".wav", ".mp3", ".m4a", ".aac"}
    for _, ext := range validExts {
        if strings.HasSuffix(strings.ToLower(filename), ext) {
            return true
        }
    }
    return false
}
```

#### 5.2.2 异步处理服务

```go
// internal/service/audio_service.go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/chaojixinren/pulse/internal/model"
    "github.com/chaojixinren/pulse/internal/repository"
    "github.com/google/uuid"
)

type AudioService struct {
    repo          *repository.AudioSessionRepository
    identityRepo  *repository.IdentityRepository
    aiService     *AIService
    cache         *redis.Client
}

func (s *AudioService) ProcessAudioAsync(sessionID uuid.UUID) {
    ctx := context.Background()

    // 1. 获取会话信息
    session, err := s.repo.GetByID(ctx, sessionID)
    if err != nil {
        log.Printf("Failed to get session %s: %v", sessionID, err)
        return
    }

    // 2. 更新状态为 processing
    session.Status = "processing"
    if err := s.repo.Update(ctx, session); err != nil {
        log.Printf("Failed to update session status: %v", err)
        return
    }

    // 3. 调用 STT
    transcript, err := s.aiService.TranscribeAudio(ctx, session.AudioURL)
    if err != nil {
        s.handleProcessError(ctx, session, err)
        return
    }
    session.Transcript = transcript

    // 4. 获取用户的身份列表
    identities, err := s.identityRepo.GetByUserID(ctx, session.UserID)
    if err != nil {
        s.handleProcessError(ctx, session, err)
        return
    }

    // 5. 识别身份
    analysis, err := s.aiService.IdentifyContext(ctx, transcript, identities)
    if err != nil {
        s.handleProcessError(ctx, session, err)
        return
    }

    // 查找匹配的身份
    for _, identity := range identities {
        if identity.Name == analysis.Identity {
            session.IdentityID = &identity.ID
            session.AIConfidence = analysis.Confidence
            break
        }
    }

    // 6. 提取信息
    extractedData, err := s.aiService.ExtractInformation(ctx, transcript)
    if err != nil {
        s.handleProcessError(ctx, session, err)
        return
    }

    // 转换为 JSON
    dataJSON, _ := json.Marshal(extractedData)
    session.ExtractedData = dataJSON

    // 7. 更新会话状态
    session.Status = "completed"
    session.ProcessedAt = time.Now()

    if err := s.repo.Update(ctx, session); err != nil {
        log.Printf("Failed to update session: %v", err)
        return
    }

    // 8. 缓存结果
    s.cacheSession(ctx, session)

    log.Printf("Successfully processed audio session %s", sessionID)
}

func (s *AudioService) handleProcessError(ctx context.Context, session *model.AudioSession, err error) {
    session.Status = "failed"
    session.ErrorMessage = err.Error()
    if updateErr := s.repo.Update(ctx, session); updateErr != nil {
        log.Printf("Failed to update error status: %v", updateErr)
    }
}

func (s *AudioService) cacheSession(ctx context.Context, session *model.AudioSession) {
    key := fmt.Sprintf("session:%s", session.ID)
    data, _ := json.Marshal(session)
    s.cache.Set(ctx, key, data, 1*time.Hour)
}
```

## 6. 安全设计

### 6.1 认证机制

#### JWT Token 实现

```go
// internal/service/auth_service.go
package service

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

type AuthService struct {
    jwtSecret     []byte
    tokenExpiry   time.Duration
    refreshExpiry time.Duration
}

func NewAuthService(secret string) *AuthService {
    return &AuthService{
        jwtSecret:     []byte(secret),
        tokenExpiry:   1 * time.Hour,
        refreshExpiry: 7 * 24 * time.Hour,
    }
}

type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

// GenerateTokens 生成访问令牌和刷新令牌
func (s *AuthService) GenerateTokens(userID, email string) (accessToken, refreshToken string, err error) {
    // 生成访问令牌
    accessClaims := Claims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "pulse-api",
        },
    }

    accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessToken, err = accessTokenObj.SignedString(s.jwtSecret)
    if err != nil {
        return "", "", err
    }

    // 生成刷新令牌
    refreshClaims := Claims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "pulse-api",
        },
    }

    refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshToken, err = refreshTokenObj.SignedString(s.jwtSecret)
    if err != nil {
        return "", "", err
    }

    return accessToken, refreshToken, nil
}

// ValidateToken 验证令牌
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return s.jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, jwt.ErrSignatureInvalid
}

// HashPassword 密码加密
func (s *AuthService) HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

// VerifyPassword 验证密码
func (s *AuthService) VerifyPassword(hashedPassword, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
```

### 6.2 认证中间件

```go
// internal/middleware/auth.go
package middleware

import (
    "net/http"
    "strings"

    "github.com/chaojixinren/pulse/internal/service"
    "github.com/gin-gonic/gin"
)

func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取 Authorization 头
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证令牌"})
            c.Abort()
            return
        }

        // 2. 检查格式 "Bearer <token>"
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌格式错误"})
            c.Abort()
            return
        }

        // 3. 验证令牌
        token := parts[1]
        claims, err := authService.ValidateToken(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌无效或已过期"})
            c.Abort()
            return
        }

        // 4. 将用户信息存入上下文
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)

        c.Next()
    }
}
```

### 6.3 限流中间件

```go
// internal/middleware/rate_limit.go
package middleware

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "golang.org/x/time/rate"
)

// RateLimitMiddleware 基于 Token Bucket 的限流
func RateLimitMiddleware(r *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Every(window/time.Duration(limit)), limit)

    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "请求过于频繁，请稍后重试",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// UserRateLimitMiddleware 基于用户的限流
func UserRateLimitMiddleware(r *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("user_id")
        if userID == "" {
            c.Next()
            return
        }

        key := "rate_limit:user:" + userID
        ctx := c.Request.Context()

        // 检查当前请求数
        count, err := r.Incr(ctx, key).Result()
        if err != nil {
            c.Next()
            return
        }

        // 首次请求设置过期时间
        if count == 1 {
            r.Expire(ctx, key, window)
        }

        // 超过限制
        if count > int64(maxRequests) {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "超过请求限制",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 6.4 数据加密

#### 语音文件加密存储

```go
// internal/service/storage_service.go
package service

import (
    "context"
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "io"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/s3"
)

type StorageService struct {
    s3Client       *s3.S3
    bucket         string
    encryptionKey  []byte // 32 bytes for AES-256
}

// UploadEncrypted 加密上传文件
func (s *StorageService) UploadEncrypted(ctx context.Context, data []byte, key string) (string, error) {
    // 1. 加密数据
    encryptedData, err := s.encrypt(data)
    if err != nil {
        return "", err
    }

    // 2. 上传到 S3
    _, err = s.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
        Bucket:               aws.String(s.bucket),
        Key:                  aws.String(key),
        Body:                 bytes.NewReader(encryptedData),
        ServerSideEncryption: aws.String("AES256"), // S3 服务端加密
        ContentType:          aws.String("audio/encrypted"),
    })

    if err != nil {
        return "", err
    }

    url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
    return url, nil
}

// DownloadDecrypted 下载并解密文件
func (s *StorageService) DownloadDecrypted(ctx context.Context, key string) ([]byte, error) {
    // 1. 从 S3 下载
    result, err := s.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, err
    }
    defer result.Body.Close()

    encryptedData, err := io.ReadAll(result.Body)
    if err != nil {
        return nil, err
    }

    // 2. 解密数据
    return s.decrypt(encryptedData)
}

// encrypt 使用 AES-256-GCM 加密
func (s *StorageService) encrypt(plaintext []byte) ([]byte, error) {
    block, err := aes.NewCipher(s.encryptionKey)
    if err != nil {
        return nil, err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, aesGCM.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

// decrypt 解密
func (s *StorageService) decrypt(ciphertext []byte) ([]byte, error) {
    block, err := aes.NewCipher(s.encryptionKey)
    if err != nil {
        return nil, err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := aesGCM.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    return aesGCM.Open(nil, nonce, ciphertext, nil)
}
```

### 6.5 HTTPS 与 CORS

```go
// internal/middleware/cors.go
package middleware

import (
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
    config := cors.Config{
        AllowOrigins:     []string{"https://app.pulse.example.com"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }

    return cors.New(config)
}
```

## 7. 部署与监控

### 7.1 Docker 部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o pulse-backend ./cmd/server

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/pulse-backend .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./pulse-backend"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DATABASE_URL=postgresql://pulse:password@postgres:5432/pulse
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
      - AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: pulse
      POSTGRES_USER: pulse
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

### 7.2 环境变量配置

```bash
# .env.production
PORT=8080
GIN_MODE=release

# 数据库
DATABASE_URL=postgresql://user:pass@host:5432/pulse?sslmode=require
REDIS_URL=redis://:password@host:6379/0

# 对象存储
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
S3_BUCKET=pulse-production
S3_REGION=us-east-1

# AI 服务
AI_API_KEY=your_adk_api_key
AI_BASE_URL=https://ark.cn-beijing.volces.com/api/v3

# 安全
JWT_SECRET=your-super-secret-jwt-key-change-in-production
ENCRYPTION_KEY=32-byte-aes-256-encryption-key-here

# StepFun API
STEPFUN_API_KEY=your_stepfun_api_key
STEPFUN_BASE_URL=https://api.stepfun.com
```

## 8. 性能优化

### 8.1 数据库连接池

```go
// internal/config/database.go
package config

import (
    "database/sql"
    "time"

    _ "github.com/lib/pq"
)

func InitDB(databaseURL string) (*sql.DB, error) {
    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        return nil, err
    }

    // 连接池配置
    db.SetMaxOpenConns(25)                 // 最大打开连接数
    db.SetMaxIdleConns(5)                  // 最大空闲连接数
    db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生命周期
    db.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接超时

    // 测试连接
    if err := db.Ping(); err != nil {
        return nil, err
    }

    return db, nil
}
```

### 8.2 Redis 缓存策略

```go
// internal/service/cache_service.go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
)

type CacheService struct {
    client *redis.Client
}

// 缓存用户会话列表
func (s *CacheService) CacheUserSessions(ctx context.Context, userID string, sessions []*model.AudioSession) error {
    key := fmt.Sprintf("user:%s:sessions", userID)
    data, err := json.Marshal(sessions)
    if err != nil {
        return err
    }

    return s.client.Set(ctx, key, data, 15*time.Minute).Err()
}

// 缓存身份列表
func (s *CacheService) CacheIdentities(ctx context.Context, userID string, identities []*model.Identity) error {
    key := fmt.Sprintf("user:%s:identities", userID)
    data, err := json.Marshal(identities)
    if err != nil {
        return err
    }

    // 身份变化不频繁，可以缓存更久
    return s.client.Set(ctx, key, data, 1*time.Hour).Err()
}

// 失效缓存
func (s *CacheService) InvalidateUserCache(ctx context.Context, userID string) error {
    pattern := fmt.Sprintf("user:%s:*", userID)
    keys, err := s.client.Keys(ctx, pattern).Result()
    if err != nil {
        return err
    }

    if len(keys) > 0 {
        return s.client.Del(ctx, keys...).Err()
    }

    return nil
}
```

## 9. 错误处理与日志

### 9.1 统一错误响应

```go
// pkg/errors/errors.go
package errors

import "net/http"

type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
    return e.Message
}

var (
    ErrUnauthorized     = &AppError{Code: http.StatusUnauthorized, Message: "未授权"}
    ErrForbidden        = &AppError{Code: http.StatusForbidden, Message: "禁止访问"}
    ErrNotFound         = &AppError{Code: http.StatusNotFound, Message: "资源不存在"}
    ErrBadRequest       = &AppError{Code: http.StatusBadRequest, Message: "请求参数错误"}
    ErrInternalServer   = &AppError{Code: http.StatusInternalServerError, Message: "服务器内部错误"}
    ErrServiceUnavailable = &AppError{Code: http.StatusServiceUnavailable, Message: "服务暂时不可用"}
)

func NewAppError(code int, message string) *AppError {
    return &AppError{Code: code, Message: message}
}
```

### 9.2 错误处理中间件

```go
// internal/middleware/error_handler.go
package middleware

import (
    "net/http"

    "github.com/chaojixinren/pulse/pkg/errors"
    "github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // 检查是否有错误
        if len(c.Errors) > 0 {
            err := c.Errors.Last()

            // 判断错误类型
            if appErr, ok := err.Err.(*errors.AppError); ok {
                c.JSON(appErr.Code, gin.H{
                    "error":   appErr.Message,
                    "details": appErr.Details,
                })
                return
            }

            // 未知错误
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "服务器内部错误",
            })
        }
    }
}
```

### 9.3 结构化日志

```go
// pkg/logger/logger.go
package logger

import (
    "os"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func Init(environment string) {
    var config zap.Config

    if environment == "production" {
        config = zap.NewProductionConfig()
    } else {
        config = zap.NewDevelopmentConfig()
    }

    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

    logger, err := config.Build()
    if err != nil {
        panic(err)
    }

    Log = logger
}

// 使用示例
// logger.Log.Info("Audio uploaded", 
//     zap.String("session_id", sessionID.String()),
//     zap.String("user_id", userID),
//     zap.Int64("file_size", fileSize),
// )
```

## 10. 总结

本文档详细设计了 Pulse 后端系统的各个方面：

1. **系统架构**：采用分层架构，清晰划分了 API、服务、数据访问等层次
2. **API 设计**：提供了完整的 RESTful API 端点，涵盖认证、语音、身份、报告等模块
3. **数据库设计**：设计了合理的表结构，支持用户、身份、语音会话等核心实体
4. **AI 集成**：基于 adk-go + eino 实现语音转文字、身份识别、信息提取的完整工作流
5. **业务流程**：明确了语音上传、处理、分析的端到端流程
6. **安全设计**：实现了 JWT 认证、限流、数据加密等安全机制

下一步可以：
- 实现具体的代码模块
- 编写单元测试和集成测试
- 完善 CI/CD 流程
- 部署到生产环境
