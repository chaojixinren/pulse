# Pulse 前端 Phase 3：生产化开发文档

> 目标：在 Phase 1/2 功能之上，补齐报告分析、账户管理、体验健壮性与部署能力，使前端达到可上线标准。

## 目标

- 实现报告增强：周报、区间统计、趋势图表、身份占比可视化。
- 实现账户管理：个人数据导出、账户注销。
- 补齐体验健壮性：错误边界、路由懒加载、深色模式、可访问性、access token 自动续期。
- 完成部署与 CI/CD。

## 完成标志（验收清单）

- [ ] 周报与统计页可用，含趋势图与身份占比图。
- [ ] 用户可导出个人数据（JSON 下载）并注销账户。
- [ ] 全站错误边界、加载态、空态、深色模式、键盘可访问性就绪。
- [ ] access token 失效可自动续期并重放请求。
- [ ] 通过 Docker/Nginx 部署，CI 全绿。

## 模块依赖关系

```
Phase 2
   ├─→ 模块1 报告增强（依赖报告接口，较独立）
   ├─→ 模块2 账户管理（依赖认证）
   ├─→ 模块3 体验健壮性（横切，贯穿各模块）
   └─→ 模块4 部署与 CI/CD（收尾）
```

> 模块 1/2/3 相互独立，可并行推进；模块 4 在所有功能稳定后收尾。

---

## 模块 1：报告增强

### 职责

在日报基础之上，提供周报与区间统计，并以图表呈现趋势与身份占比。

### 目录 / 文件

```
src/
├── pages/Report/WeeklyReport.tsx      # 周报（趋势 + 身份拆分 + 待办 + 承诺）
├── pages/Report/StatsReport.tsx       # 自定义区间统计
├── components/business/TrendChart.tsx # 每日会话数/时长趋势（SVG）
├── components/business/IdentityPie.tsx# 身份占比（SVG）
├── services/report.service.ts         # 扩展 weekly / stats
└── types/report.types.ts              # 扩展 WeeklyReport / StatsReport
```

### 类型（types/report.types.ts 扩展）

```typescript
export interface DailyPoint {
  date: string;              // YYYY-MM-DD
  session_count: number;
  total_duration: number;    // 秒
}

export interface WeeklyReport {
  week: string;              // 周一起始日 YYYY-MM-DD
  session_count: number;
  total_duration: number;
  by_identity: IdentityStat[];
  top_todos: string[];
  commitments_done: number;  // 已完成承诺数
  daily_trend: DailyPoint[];
}

export interface StatsReport {
  from: string;
  to: string;
  session_count: number;
  total_duration: number;
  by_identity: IdentityStat[];
  daily_trend: DailyPoint[];
}
```

### 服务层（services/report.service.ts 扩展）

```typescript
export const reportService = {
  daily(date: string): Promise<DailyReport> { /* Phase 1 */ },
  weekly(week?: string): Promise<WeeklyReport> {
    return http.get<WeeklyReport>('/reports/weekly', { params: { week } });
  },
  stats(from: string, to: string): Promise<StatsReport> {
    return http.get<StatsReport>('/reports/stats', { params: { from, to } });
  },
};
```

### 图表方案

- 优先使用 **SVG 自建**柱状图（`daily_trend`）与占比环图（`by_identity`），避免引入重依赖。
- 如需更丰富的交互（悬停提示、动画），可引入 `recharts`（纯 React、按需引入）。

### 页面 / 路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/reports/weekly` | WeeklyReport | 周报（默认本周，可切换周） |
| `/reports/stats` | StatsReport | 自定义区间统计 |

交互约定：

- 周报默认取本周（周一为起点），可前后切换；展示趋势柱状图、身份占比、Top 待办、承诺完成数。
- 统计页提供 `from/to` 日期选择，缺省为近 30 天。
- 时长统一格式化为「X 小时 Y 分钟」。

### 验收标准

- [ ] 周报与统计页数据正确，趋势与占比图渲染正常。
- [ ] 日期/周切换生效。
- [ ] 图表在空数据时展示空态。

---

## 模块 2：账户管理

### 职责

提供个人数据导出与账户注销，满足用户数据掌控与「被遗忘权」诉求。

### 目录 / 文件

```
src/
├── pages/Account/AccountSettings.tsx    # 账户设置（导出 + 注销入口）
├── services/account.service.ts
└── types/account.types.ts
```

### 类型（types/account.types.ts）

```typescript
import { User } from './api.types';
import { Identity } from './identity.types';
import { Device } from './device.types';

export interface AccountExport {
  user: User;
  identities: Identity[];
  devices: Device[];
  sessions: Array<Record<string, unknown>>; // 音频会话（含转写与提取）
}
```

### 服务层（services/account.service.ts）

```typescript
import { http } from './api';
import { AccountExport } from '@/types/account.types';

export const accountService = {
  export(): Promise<AccountExport> {
    return http.get<AccountExport>('/account/export');
  },
  delete(): Promise<void> {
    return http.delete('/account');
  },
};
```

### 页面 / 交互约定

- 「导出数据」：调用 `/account/export` 获取 JSON，前端用 `Blob` 下载为 `pulse-export-<日期>.json`。
- 「注销账户」：二次确认（输入邮箱或勾选确认），注销成功后清理本地 token 并跳转注册页。
- 注销是不可逆操作，文案需明确警示。

### 验收标准

- [ ] 可导出个人数据并下载为 JSON 文件。
- [ ] 注销有明确二次确认，注销后登出并清理本地状态。

---

## 模块 3：体验健壮性

### 职责

补齐横切关注点，使应用在生产环境下稳定、可访问、体验一致。

### 目录 / 文件

```
src/
├── components/common/ErrorBoundary.tsx  # 路由级错误边界
├── components/common/FullScreenLoading.tsx
├── contexts/AppContext.tsx              # 主题（深色模式）持久化
├── services/api.ts                      # 增加 access token 自动续期
└── hooks/useAsync.ts                    # 统一加载/错误/重试
```

### 关键实现

#### 3.1 access token 自动续期（services/api.ts 增强）

```typescript
let refreshing: Promise<string | null> | null = null;

async function tryRefresh(): Promise<string | null> {
  const rt = storage.getRefreshToken();
  if (!rt) return null;
  if (!refreshing) {
    refreshing = authService.refresh(rt)
      .then((pair) => {
        storage.setTokens(pair.access_token, pair.refresh_token);
        return pair.access_token;
      })
      .catch(() => {
        storage.clearTokens();
        return null;
      })
      .finally(() => { refreshing = null; });
  }
  return refreshing;
}

client.interceptors.response.use(undefined, async (error) => {
  const original = error.config;
  if (error.response?.status === 401 && !original._retried) {
    original._retried = true;
    const token = await tryRefresh();
    if (token) {
      original.headers.Authorization = `Bearer ${token}`;
      return client(original); // 重放原请求
    }
    window.location.assign('/auth/login');
  }
  return Promise.reject(error);
});
```

#### 3.2 错误边界与懒加载

- 每个路由用 `ErrorBoundary` 包裹，崩溃时展示「出错了 + 重试」，避免白屏。
- 路由已按需懒加载（Phase 1 已引入），Phase 3 补充构建时的代码分割（manualChunks）。

#### 3.3 深色模式

- 主题存储于 localStorage，`<html data-theme>` 切换 CSS 变量（`variables.css` 已定义深浅两套）。
- 跟随系统偏好 `prefers-color-scheme`，用户可手动覆盖。

#### 3.4 可访问性

- 交互元素提供 `aria-label`，表单控件关联 label。
- 焦点可见、键盘可操作（Modal/下拉菜单支持 Esc 与焦点陷阱）。
- 色觉无障碍：关键信息不只靠颜色传达（如身份徽标同时显示名称）。

### 验收标准

- [ ] 401 自动续期成功，且并发请求只触发一次刷新。
- [ ] 任一页面崩溃展示错误边界而非白屏。
- [ ] 深色模式切换正常且持久化。
- [ ] 键盘可完成登录、身份创建、设备绑定等核心流程。

---

## 模块 4：部署与 CI/CD

### 职责

提供可复现的构建与部署，接入 CI/CD。

### 目录 / 文件

```
frontend/
├── Dockerfile              # 多阶段构建（node 构建 + nginx 托管）
├── nginx.conf              # SPA 路由回退 + 反向代理 /api
└── .github/workflows/ci.yml # 前端 CI（见 ci-cd.md）
```

### Dockerfile（示意）

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_API_BASE_URL=/api/v1
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### 验收标准

- [ ] `docker build` 成功，镜像启动后可访问应用。
- [ ] 直接访问子路由（如 `/timeline`）刷新不 404（SPA 回退）。
- [ ] `/api` 反向代理到后端正常。

---

## 阶段验收清单

- [ ] 周报 / 统计页可用，图表渲染正常。
- [ ] 数据导出与账户注销可用。
- [ ] 错误边界、懒加载、深色模式、可访问性、token 自动续期就绪。
- [ ] Docker 部署成功，CI/CD 全绿。
- [ ] 生产安全检查通过（见 ci-cd.md）。
