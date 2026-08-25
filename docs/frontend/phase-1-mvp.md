# Pulse 前端 Phase 1：MVP 开发文档

> 目标：跑通「登录 → 身份管理 → 时间线 → 日报」的最小可用 Web 应用，让用户能通过浏览器完成核心闭环。
> 本阶段不涉及设备管理与 AI 结果的结构化展示（见 Phase 2）。

## 目标

- 搭建可运行的 React + TypeScript + Vite 工程骨架。
- 实现认证（注册 / 登录 / 登出 / 会话保持 / 路由守卫）。
- 实现身份管理（列表 / 创建 / 编辑 / 删除 / 设默认）。
- 实现时间线（转写文本列表 + 按身份/日期/状态过滤 + 分页）。
- 实现日报查看（当日汇总 + 按身份拆分 + AI 提取的待办/笔记）。

## 完成标志（验收清单）

- [ ] `npm run dev` 可启动，`npm run build` 与 `tsc --noEmit` 通过。
- [ ] 用户可注册 / 登录 / 登出，受保护页面未登录自动跳转登录页。
- [ ] 用户可创建 / 编辑 / 删除身份，并可设为默认身份。
- [ ] 时间线按身份、日期范围、状态过滤，分页正常，转写文本可见。
- [ ] 日报页展示当日会话数、总时长、按身份拆分与 AI 待办/笔记。

## 模块依赖关系

```
模块1 项目骨架
   └─→ 模块2 认证（登录态、路由守卫、axios 封装）
          └─→ 模块3 身份管理 ─┐
          └─→ 模块4 时间线 ───┼─→ 模块5 日报查看（依赖时间线+身份）
```

> 说明：模块 5 日报查看依赖时间线数据与身份数据，但后端日报接口已聚合好数据，前端只需渲染，故可与模块 3/4 并行。

---

## 模块 1：项目骨架

### 职责

搭建可运行、可扩展的工程骨架，提供统一的路由、布局、API 客户端、类型与样式基础。

### 目录 / 文件

```
frontend/
├── index.html
├── tsconfig.json
├── vite.config.ts            # @ 别名指向 src
├── package.json
├── .env.example              # VITE_API_BASE_URL
└── src/
    ├── main.tsx              # 挂载 AuthProvider + AppProvider + RouterProvider
    ├── App.tsx               # 根组件（渲染 <Outlet/>）
    ├── router/routes.tsx     # 路由表 + 懒加载 + 路由守卫
    ├── components/layout/    # Layout / Header / Sidebar
    ├── components/common/    # Button / Input / Modal / Loading / Empty / Toast
    ├── services/api.ts       # axios 实例 + 请求/响应拦截器 + ApiError
    ├── types/api.types.ts    # ApiResponse / User / TokenPair / Paginated
    ├── styles/variables.css  # CSS 变量（颜色/字号/间距/圆角/阴影）
    └── styles/global.css     # 全局重置 + 基础排版
```

### 核心类型（types/api.types.ts）

```typescript
// 后端统一响应：code=0 表示成功
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  settings?: string;
  created_at: string;
  updated_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface Paginated<T> {
  items: T[];
  total: number;
  page: number;
  size: number;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly code: number,
    public readonly status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
```

### API 客户端（services/api.ts）

```typescript
import axios, { AxiosRequestConfig } from 'axios';
import { ApiResponse, ApiError } from '@/types/api.types';
import { storage } from '@/utils/storage';

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 30000,
});

// 请求拦截器：附加 Bearer token
client.interceptors.request.use((config) => {
  const token = storage.getAccessToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// 响应拦截器：统一解析 {code, message, data}
client.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse;
    if (body.code !== 0) {
      return Promise.reject(new ApiError(body.message || '请求失败', body.code, response.status));
    }
    return body.data; // 直接返回业务数据
  },
  (error) => {
    const status = error.response?.status ?? 0;
    const body = error.response?.data as ApiResponse | undefined;
    const message = body?.message ?? error.message ?? '网络错误';
    return Promise.reject(new ApiError(message, body?.code ?? status, status));
  },
);

// 类型安全的请求封装：响应拦截器已把 {code, message, data} 解包为业务数据，
// 这里用泛型修正 axios 的静态返回类型，使 service 方法返回业务数据本身。
export const http = {
  get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return client.get(url, config) as Promise<T>;
  },
  post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return client.post(url, data, config) as Promise<T>;
  },
  put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return client.put(url, data, config) as Promise<T>;
  },
  delete<T = void>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return client.delete(url, config) as Promise<T>;
  },
};
```

### 路由与守卫（router/routes.tsx）

```typescript
import { Navigate, createBrowserRouter } from 'react-router-dom';
import { RequireAuth } from './RequireAuth';

export const router = createBrowserRouter([
  { path: '/auth/login', lazy: () => import('@/pages/Auth/Login') },
  { path: '/auth/register', lazy: () => import('@/pages/Auth/Register') },
  {
    path: '/',
    element: <RequireAuth />,
    children: [
      { index: true, element: <Navigate to="/identity" replace /> },
      { path: 'identity', lazy: () => import('@/pages/Identity/IdentityList') },
      { path: 'timeline', lazy: () => import('@/pages/Timeline/TimelineList') },
      { path: 'reports/daily', lazy: () => import('@/pages/Report/DailyReport') },
    ],
  },
  { path: '*', element: <NotFound /> },
]);
```

```tsx
// RequireAuth.tsx：未登录跳登录页，登录后放行子路由
export function RequireAuth() {
  const { isAuthenticated, loading } = useAuth();
  if (loading) return <FullScreenLoading />;
  if (!isAuthenticated) return <Navigate to="/auth/login" replace />;
  return <Layout><Outlet /></Layout>;
}
```

### 验收标准

- [ ] `npm run dev` 可启动，访问 `/` 未登录时跳转 `/auth/login`。
- [ ] `@` 路径别名、CSS 变量、axios 拦截器可用。
- [ ] 布局（Header + Sidebar）与通用组件（Button/Input/Modal/Empty）渲染正常。

---

## 模块 2：认证

### 职责

实现注册、登录、登出、会话保持与路由守卫，是所有受保护页面的前置依赖。

### 目录 / 文件

```
src/
├── pages/Auth/Login.tsx
├── pages/Auth/Register.tsx
├── contexts/AuthContext.tsx      # 认证状态 + login/logout/refreshUser
├── services/auth.service.ts
└── utils/storage.ts              # access_token / refresh_token 本地存取
```

### 服务层（services/auth.service.ts）

```typescript
import { http } from './api';
import { User, TokenPair } from '@/types/api.types';

export const authService = {
  register(data: { email: string; password: string; name: string }): Promise<User> {
    return http.post<User>('/auth/register', data);
  },
  login(data: { email: string; password: string }): Promise<TokenPair> {
    return http.post<TokenPair>('/auth/login', data);
  },
  logout(refresh_token: string): Promise<void> {
    return http.post<void>('/auth/logout', { refresh_token });
  },
  refresh(refresh_token: string): Promise<TokenPair> {
    return http.post<TokenPair>('/auth/refresh', { refresh_token });
  },
  me(): Promise<User> {
    return http.get<User>('/auth/me');
  },
};
```

### 状态管理（contexts/AuthContext.tsx）

```tsx
interface AuthContextValue {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}
```

行为约定：

1. 应用启动时若本地存在 access_token，调用 `me()` 恢复用户（失败则清理本地 token）。
2. `login/register` 成功后写入 `access_token` / `refresh_token` 到 localStorage。
3. `logout` 调用后端并清理本地 token。
4. Phase 1 不做 access token 自动续期（401 时直接跳登录页）；自动续期留到 Phase 3。

### 页面 / 路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/auth/login` | Login | 邮箱 + 密码，登录后跳转 `/` |
| `/auth/register` | Register | 邮箱 + 密码（≥8 位）+ 姓名，注册后自动登录 |

### 验收标准

- [ ] 注册成功后可登录，刷新页面会话保持。
- [ ] 登录态访问受保护页面正常；登出后访问受保护页面跳登录页。
- [ ] 密码错误 / 邮箱已注册等错误给出明确提示。
- [ ] 表单校验：邮箱格式、密码长度、姓名非空。

---

## 模块 3：身份管理

### 职责

提供身份的增删改查与默认身份设置，为时间线与报告提供身份维度。

### 目录 / 文件

```
src/
├── pages/Identity/IdentityList.tsx      # 列表 + 创建/编辑弹窗 + 设默认 + 删除
├── services/identity.service.ts
├── types/identity.types.ts
└── components/business/IdentityCard.tsx # 身份卡片（名称/颜色/图标/默认标识）
```

### 类型（types/identity.types.ts）

```typescript
export interface Identity {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  color: string;
  icon: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface IdentityInput {
  name: string;          // 必填
  description?: string;
  color?: string;
  icon?: string;
  is_default?: boolean;
}
```

### 服务层（services/identity.service.ts）

```typescript
import { http } from './api';
import { Identity, IdentityInput } from '@/types/identity.types';

export const identityService = {
  list(): Promise<Identity[]> {
    return http.get<Identity[]>('/identities');
  },
  create(data: IdentityInput): Promise<Identity> {
    return http.post<Identity>('/identities', data);
  },
  update(id: string, data: IdentityInput): Promise<Identity> {
    return http.put<Identity>(`/identities/${id}`, data);
  },
  remove(id: string): Promise<void> {
    return http.delete(`/identities/${id}`);
  },
  setDefault(id: string): Promise<void> {
    return http.put<void>(`/identities/${id}/default`);
  },
};
```

### 页面 / 路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/identity` | IdentityList | 身份卡片网格，支持创建/编辑/删除/设默认 |

交互约定：

- 列表进入时加载全部身份；支持空态（引导创建首个身份）。
- 创建/编辑用 Modal 表单（名称、描述、颜色、图标、是否默认）。
- 删除需二次确认；默认身份不可删除（后端会拒绝，前端提示）。
- 设置默认身份后刷新列表，更新默认标识。

### 验收标准

- [ ] 可创建身份并立即出现在列表。
- [ ] 可编辑身份字段并生效。
- [ ] 删除有二次确认，删除后列表更新。
- [ ] 设默认后对应身份显示「默认」标识。
- [ ] 空态展示与错误态重试正常。

---

## 模块 4：时间线

### 职责

展示语音会话的转写文本，支持按身份、日期范围、状态过滤与分页。

### 目录 / 文件

```
src/
├── pages/Timeline/TimelineList.tsx       # 列表 + 过滤栏 + 分页
├── services/timeline.service.ts
├── types/timeline.types.ts
└── components/business/TimelineItem.tsx  # 单条时间线（转写/时长/身份/状态）
```

### 类型（types/timeline.types.ts）

```typescript
export interface TimelineItem {
  session_id: string;
  identity_id?: string;   // 低置信度时后端可能不返回
  transcript: string;
  duration: number;       // 秒
  status: 'pending' | 'processing' | 'completed' | 'failed';
  recorded_at: string;
}

export interface TimelineQuery {
  identity_id?: string;
  from?: string;   // RFC3339
  to?: string;     // RFC3339
  status?: string;
  page?: number;
  size?: number;
}
```

### 服务层（services/timeline.service.ts）

```typescript
import { http } from './api';
import { TimelineItem, TimelineQuery } from '@/types/timeline.types';
import { Paginated } from '@/types/api.types';

export const timelineService = {
  list(query: TimelineQuery): Promise<Paginated<TimelineItem>> {
    return http.get<Paginated<TimelineItem>>('/timeline', { params: query });
  },
};
```

### 页面 / 路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/timeline` | TimelineList | 时间线列表 + 过滤 + 分页 |

交互约定：

- 过滤条件：身份下拉（来自身份列表）、日期范围、状态（pending/processing/completed/failed）。
- 单条展示转写文本（长文本折叠）、录制时间（本地时区）、时长、状态、身份徽标。
- 分页用「上一页/下一页 + 页码」，后端返回 `total/page/size`。
- `processing` 状态支持轮询刷新（Phase 1 可用手动刷新）。

### 验收标准

- [ ] 列表正常加载并分页。
- [ ] 按身份/日期/状态过滤生效。
- [ ] 转写文本、时长、录制时间、状态正确展示。
- [ ] 空态与加载态正常。

---

## 模块 5：日报查看

### 职责

展示某日的汇总数据：会话数、总时长、按身份拆分，以及 AI 提取的待办与笔记。

### 目录 / 文件

```
src/
├── pages/Report/DailyReport.tsx        # 日期选择 + 汇总卡片 + 身份拆分 + 待办/笔记
├── services/report.service.ts
└── types/report.types.ts
```

### 类型（types/report.types.ts）

```typescript
export interface IdentityStat {
  identity_id: string;
  name: string;
  session_count: number;
  total_duration: number;
}

export interface DailyReport {
  date: string;              // YYYY-MM-DD
  session_count: number;
  total_duration: number;    // 秒
  by_identity: IdentityStat[];
  todos: string[];           // AI 提取的待办
  notes: string[];           // AI 提取的笔记
}
```

### 服务层（services/report.service.ts）

```typescript
import { http } from './api';
import { DailyReport } from '@/types/report.types';

export const reportService = {
  daily(date: string): Promise<DailyReport> {
    return http.get<DailyReport>('/reports/daily', { params: { date } });
  },
};
```

### 页面 / 路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/reports/daily` | DailyReport | 日报（默认今天，可切换日期） |

交互约定：

- 默认加载今天，支持前后切换日期（`YYYY-MM-DD`）。
- 汇总卡片：会话数、总时长（分钟/小时格式化）。
- 按身份拆分：每个身份的名称、会话数、时长。
- 待办/笔记列表分别展示 AI 提取内容；无数据时展示空态。

### 验收标准

- [ ] 日报正确展示汇总、按身份拆分、待办、笔记。
- [ ] 切换日期加载对应日报。
- [ ] 无数据日期展示空态。

---

## 阶段验收清单

- [ ] 工程骨架可运行、可构建、类型检查通过。
- [ ] 注册 / 登录 / 登出 / 会话保持 / 路由守卫完整。
- [ ] 身份管理 CRUD + 设默认可用。
- [ ] 时间线列表、过滤、分页可用。
- [ ] 日报查看可用。
- [ ] 统一错误提示、加载态、空态贯穿各页面。
- [ ] 与后端联调通过（对 `backend` 以 `TEST_DATABASE_DSN` 启动本地服务）。
