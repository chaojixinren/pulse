# Pulse 前端 Phase 2：设备与 AI 开发文档

> 目标：在 Phase 1 闭环之上，支持硬件设备绑定与状态管理，并把 AI 分析结果（身份识别、待办/承诺/笔记）以结构化方式呈现给用户。
> 依赖后端 Phase 2 已提供的设备接口与 AI 分析能力（adk-go 身份识别 + 信息提取）。

## 目标

- 实现设备管理：设备创建/绑定（一次性 token）、列表、详情、解绑、心跳状态展示、指令下发。
- 实现 AI 结果展示：时间线标注 AI 识别的身份，报告/会话中呈现提取的待办、承诺、笔记。

## 完成标志（验收清单）

- [ ] 用户可创建/绑定设备并一次性拿到 device_token 手抄到硬件，绑定后设备出现在列表。
- [ ] 设备列表展示在线状态、电量、固件版本、最后活跃时间。
- [ ] 用户可解绑设备、向设备下发指令（先落库）。
- [ ] 时间线每条会话标注 AI 识别的身份（低置信度时显示「未识别」）。
- [ ] 报告页展示 AI 提取的待办 / 承诺 / 笔记。

## 模块依赖关系

```
Phase 1（骨架 + 认证 + 身份 + 时间线 + 日报）
   ├─→ 模块1 设备管理（较独立，仅依赖认证）
   └─→ 模块2 AI 结果展示（依赖身份 + 时间线 + 报告）
```

---

## 模块 1：设备管理

### 职责

提供设备的绑定、解绑、列表、详情、状态展示与指令下发。

### 目录 / 文件

```
src/
├── pages/Device/DeviceList.tsx        # 设备列表（在线状态/电量/固件）
├── pages/Device/DeviceDetail.tsx      # 设备详情 + 指令下发
├── pages/Device/BindDevice.tsx        # 设备 ID + 名称表单，成功后展示一次性 token
├── services/device.service.ts
├── types/device.types.ts
└── components/business/DeviceCard.tsx # 设备卡片
```

### 类型（types/device.types.ts）

```typescript
export interface Device {
  id: string;
  user_id: string;
  device_id: string;        // 硬件侧唯一标识
  name: string;
  device_type: string;
  firmware_version?: string;
  battery_level?: number;   // 0-100
  last_seen_at?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateDeviceInput {
  device_id: string;        // 硬件侧唯一标识，需与 config.json 中 cloud.device_id 一致
  name?: string;
}

export interface CreateDeviceResult {
  device: Device;
  device_token: string;     // 一次性明文 token，仅此响应返回，服务端只存 hash
}

export interface DeviceCommand {
  id: string;
  device_id: string;
  user_id: string;
  command: string;
  status: string;           // pending / delivered / …
  created_at: string;
  updated_at: string;
}
```

### 服务层（services/device.service.ts）

```typescript
import { http } from './api';
import { Device, CreateDeviceInput, CreateDeviceResult, DeviceCommand } from '@/types/device.types';

export const deviceService = {
  create(data: CreateDeviceInput): Promise<CreateDeviceResult> {
    return http.post<CreateDeviceResult>('/devices', data);
  },
  list(): Promise<Device[]> {
    return http.get<Device[]>('/devices');
  },
  get(id: string): Promise<Device> {
    return http.get<Device>(`/devices/${id}`);
  },
  unbind(id: string): Promise<void> {
    return http.delete(`/devices/${id}`);
  },
  issueCommand(id: string, command: string): Promise<DeviceCommand> {
    return http.post<DeviceCommand>(`/devices/${id}/command`, { command });
  },
};
```

> 说明：心跳上报（`POST /devices/:id/heartbeat`）由硬件侧调用，前端不直接触发；前端通过轮询/刷新读取 `last_seen_at` / `battery_level` 展示状态。

### 页面 / 路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/devices` | DeviceList | 设备列表 + 绑定入口 |
| `/devices/bind` | BindDevice | 填写设备 ID + 名称，成功后展示一次性 token |
| `/devices/:id` | DeviceDetail | 设备详情 + 指令下发 + 解绑 |

交互约定：

- 设备列表展示：名称、类型、在线状态（依据 `is_active` 与 `last_seen_at` 判定）、电量、固件版本。
- 绑定页填写设备 ID + 名称，创建成功后展示一次性 `device_token`（4 字符分组 + 复制），提示手抄到硬件 config.json（仅此一次）。
- 指令下发：预设指令（开始录音 / 停止录音 / 上报状态）或自定义，下发后展示落库结果与状态。
- 解绑需二次确认。

### 验收标准

- [ ] 可创建设备（设备 ID + 名称），一次性拿到 `device_token` 手抄到硬件。
- [ ] 绑定成功后设备出现在列表，`device_token` 只展示一次。
- [ ] 设备列表正确展示在线状态、电量、固件、最后活跃时间。
- [ ] 可解绑设备，解绑后列表更新。
- [ ] 可下发指令并看到指令落库结果。

---

## 模块 2：AI 结果展示

### 职责

把后端 AI 分析结果（身份识别、待办 / 承诺 / 笔记提取）结构化呈现，让用户「看到 AI 做了什么」。

### 数据来源

| AI 结果 | 后端载体 | 前端展示位置 |
|---------|----------|--------------|
| 身份识别 | `TimelineItem.identity_id` | 时间线条目的身份徽标 |
| 待办 / 笔记 | `DailyReport.todos / notes` | 日报页 |
| 待办（周） | `WeeklyReport.top_todos` | 周报页（Phase 3） |
| 承诺完成 | `WeeklyReport.commitments_done` | 周报页（Phase 3） |

### 目录 / 文件

```
src/
├── components/business/IdentityBadge.tsx     # 身份徽标（颜色 + 图标 + 名称）
├── components/business/ExtractedList.tsx     # 待办/承诺/笔记列表
├── pages/Timeline/TimelineList.tsx           # 复用 Phase 1，增加身份徽标
├── pages/Report/DailyReport.tsx              # 复用 Phase 1，增强待办/笔记展示
└── hooks/useIdentityMap.ts                   # identity_id → Identity 的映射缓存
```

### 关键实现（hooks/useIdentityMap.ts）

```typescript
import { useMemo } from 'react';
import { Identity } from '@/types/identity.types';

// 将身份列表转为 id → Identity 映射，供时间线/报告快速取名称、颜色、图标
export function useIdentityMap(identities: Identity[] | undefined) {
  return useMemo(() => {
    const map = new Map<string, Identity>();
    identities?.forEach((it) => map.set(it.id, it));
    return map;
  }, [identities]);
}
```

### 页面 / 交互约定

时间线（`/timeline`）：

- 每条会话根据 `identity_id` 显示对应身份徽标（名称 + 颜色 + 图标）。
- `identity_id` 缺失时显示「未识别」灰标（对应后端低置信度不绑定身份的情况）。
- 转写文本折叠展开，长文本不撑破布局。

日报（`/reports/daily`）：

- 待办 / 笔记用 `ExtractedList` 结构化展示，支持逐条展示与空态。
- 待办项可手动标记完成（本地状态，Phase 3 可选持久化）。

### 预留项（依赖后端补充接口）

> 以下能力当前后端尚未提供对应接口，前端先预留设计，待后端补齐后实现：

1. **单会话结构化详情**：当前 `TimelineItem` 仅含转写文本，不含 `extracted_data`（待办/承诺/笔记的逐条原文与置信度）。待后端提供单会话详情接口（如 `GET /audio/:id` 或 `GET /timeline/:session_id`）后，实现「会话详情抽屉」展示提取明细。
2. **低置信度身份手动标注**：当前无「手动修正身份」接口。待后端提供标注接口后，在会话详情提供「重新归类身份」操作。

### 验收标准

- [ ] 时间线每条会话显示 AI 识别的身份徽标；未识别会话显示「未识别」。
- [ ] 身份徽标颜色/图标与身份定义一致。
- [ ] 日报待办 / 笔记结构化展示，空态正常。

---

## 阶段验收清单

- [ ] 设备管理全流程（创建 → 手抄 token → 列表 → 详情 → 指令 → 解绑）可用。
- [ ] 设备状态（在线/电量/固件/最后活跃）正确展示。
- [ ] 时间线身份徽标正确，低置信度场景展示「未识别」。
- [ ] 日报 AI 待办 / 笔记结构化展示。
- [ ] 与后端 Phase 2 联调通过。
