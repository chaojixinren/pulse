# Pulse 前端模块开发文档（总览）

> 本文档是 Pulse 前端的分阶段开发指南，按「跑通基础闭环 → 设备与 AI → 生产化」三个阶段组织。
> 每个阶段一个独立文档，内部按模块拆分，包含职责、目录/文件、数据结构、接口签名、页面/路由与验收标准。
>
> 与 `docs/frontend-design.md` 的关系：后者是**总体设计蓝图**（架构、页面、状态管理、UI 规范、组件设计），本目录是对它的**分阶段落地拆解**，并以后端实际 API 契约为准修正类型与交互细节。二者冲突时，以本目录为准。

## 技术栈（已确定）

- **核心框架**：React 18+
- **语言**：TypeScript 5.3+
- **构建工具**：Vite 5+
- **路由**：React Router 6.21+
- **HTTP 客户端**：Axios 1.6+
- **状态管理**：React Context + 自定义 Hooks（轻量级，不引入 Redux/Zustand/MobX）
- **UI 方案**：自建通用组件 + CSS 变量设计系统（不引入 Ant Design/MUI 等第三方组件库）
- **代码规范**：ESLint + TypeScript ESLint
- **测试**：Vitest（单测）+ React Testing Library（组件）+ Playwright（E2E，可选）

> 说明：前端状态管理与 UI 方案已在 `docs/frontend-design.md` 中定案（Context + Hooks、自建组件库），本目录沿用该决策。

## 阶段总览

| 阶段 | 目标 | 核心模块 | 完成标志 |
|------|------|----------|----------|
| [Phase 1：MVP](phase-1-mvp.md) | 跑通「登录 → 身份 → 时间线 → 日报」最小闭环 | 骨架、认证、身份管理、时间线、日报 | 用户可登录注册、管理身份、按身份查看转写时间线、查看日报 |
| [Phase 2：设备与 AI](phase-2-device-ai.md) | 设备管理 + AI 结果展示 | 设备管理、AI 结果展示（待办/承诺/笔记、身份识别） | 用户可绑定设备查看状态，并能在会话中看到 AI 提取的结构化信息 |
| [Phase 3：生产化](phase-3-production.md) | 达到可上线标准 | 报告增强、账户管理、体验健壮性、部署与 CI/CD | 通过体验与安全检查，可灰度上线 |

## 前端目录结构（规划）

```
frontend/
├── public/                    # 静态资源（favicon、index.html 模板）
├── src/
│   ├── main.tsx               # 应用入口（挂载 Provider + Router）
│   ├── App.tsx                # 根组件（路由表）
│   ├── router/
│   │   └── routes.tsx         # 路由配置 + 懒加载 + 路由守卫
│   ├── contexts/              # React Context（全局状态）
│   │   ├── AuthContext.tsx    # 认证状态
│   │   └── AppContext.tsx     # 主题 / 侧边栏等应用级状态
│   ├── components/            # 可复用组件
│   │   ├── common/            # 通用组件（Button / Input / Modal / Loading / Empty …）
│   │   ├── layout/            # 布局组件（Header / Sidebar / Layout）
│   │   └── business/          # 业务组件（IdentityCard / TimelineItem / TodoList …）
│   ├── pages/                 # 页面组件
│   │   ├── Auth/              # Login / Register
│   │   ├── Identity/          # 身份列表 / 创建 / 详情
│   │   ├── Timeline/          # 时间线列表 / 会话详情
│   │   ├── Device/            # 设备列表 / 绑定 / 详情
│   │   ├── Report/            # 日报 / 周报 / 统计
│   │   └── Account/           # 数据导出 / 账户注销
│   ├── services/              # API 服务层
│   │   ├── api.ts             # axios 实例 + 拦截器 + 统一响应解析
│   │   ├── auth.service.ts
│   │   ├── identity.service.ts
│   │   ├── timeline.service.ts
│   │   ├── device.service.ts
│   │   ├── report.service.ts
│   │   └── account.service.ts
│   ├── hooks/                 # 自定义 Hooks（useAsync / useAuth / useIdentity …）
│   ├── types/                 # TypeScript 类型（与后端 model 对齐）
│   │   ├── api.types.ts       # 统一响应、分页、User、TokenPair
│   │   ├── identity.types.ts
│   │   ├── timeline.types.ts
│   │   ├── device.types.ts
│   │   └── report.types.ts
│   ├── utils/                 # 工具函数（storage / date / format …）
│   └── styles/                # 样式（variables.css / global.css …）
├── .env.example               # 环境变量模板
├── index.html
├── tsconfig.json
├── vite.config.ts
└── package.json
```

## 分层约定

```
页面组件（pages）→ 业务/通用组件（components）→ 自定义 Hooks（hooks）
                                    ↘
                            API 服务层（services）→ Axios → 后端
                                    ↘
                            类型定义（types）
```

- **pages 层**：路由级组件，只做数据组装与页面布局，不直接写请求逻辑。
- **components 层**：纯展示或轻交互，通过 props 接收数据、通过回调上抛事件，不直接调用 service。
- **services 层**：唯一的网络出口，封装 axios，方法返回业务数据（已解析后的类型化数据）。
- **hooks 层**：封装可复用的状态与副作用逻辑（加载态、错误态、数据刷新）。
- **types 层**：与后端 `model` 的 JSON 字段一一对应，是前后端契约的唯一来源。

## 通用约定

### 1. 统一响应结构

后端统一返回：

```json
{ "code": 0, "message": "ok", "data": { } }
```

- `code` 为 `0` 表示成功；非 0 表示业务/系统错误。
- 失败时 `data` 缺省，`message` 为可展示的错误信息。

axios 响应拦截器约定：

- 成功（HTTP 2xx 且 `code === 0`）：解析并返回 `data`。
- 业务错误（HTTP 2xx 但 `code !== 0` 或非 2xx）：抛出带 `message` 的统一错误。
- 401：清理本地 token 并跳转登录页（Phase 3 增加 access token 自动续期）。

### 2. 错误处理

- service 层方法拒绝（reject）统一 `ApiError`（含 `message`、`code`、`status`）。
- 页面/组件层通过 `useAsync` 或 try/catch 捕获，展示错误态并支持重试。
- 表单校验错误就近展示；网络/服务端错误用 Toast 提示。

### 3. 命名与风格

- 组件文件名用 PascalCase（`IdentityCard.tsx`），工具/服务用 camelCase（`auth.service.ts`）。
- 组件统一用函数式组件 + Hooks，禁止类组件。
- API 方法命名：`getXxx / createXxx / updateXxx / deleteXxx`。
- 类型定义与后端 JSON 字段同名（snake_case），前端通过类型标注避免歧义。

### 4. 时间处理

- 后端统一返回 RFC3339（UTC）。前端展示时用 `Intl.DateTimeFormat` 或工具函数转本地时区。
- 日期区间查询（`from/to/date/week`）用 `YYYY-MM-DD` / `YYYY-Www` 字符串。

### 5. 环境变量

```env
# .env.example
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

- 所有网络请求的 baseURL 来自 `VITE_API_BASE_URL`。
- 生产构建时通过 `.env.production` 覆盖为线上地址。

## 模块地图

| 模块 | 阶段 | 涉及目录 | 依赖 |
|------|------|----------|------|
| 项目骨架 | 1 | main/App/router/components/common/layout、services/api、types、styles | 无 |
| 认证 | 1 | pages/Auth、contexts/AuthContext、services/auth | 骨架 |
| 身份管理 | 1 | pages/Identity、services/identity | 认证 |
| 时间线 | 1 | pages/Timeline、services/timeline | 认证、身份 |
| 日报 | 1 | pages/Report、services/report | 时间线 |
| 设备管理 | 2 | pages/Device、services/device | 认证 |
| AI 结果展示 | 2 | pages/Timeline、components/business | 时间线、身份 |
| 报告增强 | 3 | pages/Report、services/report | 时间线、AI |
| 账户管理 | 3 | pages/Account、services/account | 认证 |
| 体验健壮性 | 3 | 全局（错误边界/懒加载/空态/深色模式） | 各模块 |
| 部署与 CI/CD | 3 | Dockerfile、.github/workflows | 全部 |

## 测试与 CI

| 类型 | 工具/命令 | 说明 |
|------|-----------|------|
| 单元测试 | Vitest | 工具函数、Hooks |
| 组件测试 | React Testing Library | 组件交互与渲染 |
| E2E | Playwright（可选） | 关键用户路径 |
| 类型检查 | `tsc --noEmit` | 构建前必跑 |
| 代码检查 | `eslint` | 提交前必跑 |

CI 在 push / PR 时执行 `lint` → `type-check` → `test` → `build`，详见 [前端 CI/CD](ci-cd.md)。

## 开发顺序建议

严格按阶段推进，**上一阶段验收通过再进入下一阶段**：

1. Phase 1 先搭骨架 + 认证，因为所有受保护页面都依赖登录态与 axios 封装。
2. Phase 1 的身份 → 时间线 → 日报是强依赖链，按此顺序开发。
3. Phase 2 的设备管理较独立，可与 AI 结果展示并行；AI 结果展示依赖 Phase 1 时间线。
4. Phase 3 的报告增强、账户管理、体验健壮性相互独立，可并行推进；部署与 CI/CD 收尾。

## 相关文档

- [Phase 1：MVP 开发文档](phase-1-mvp.md)
- [Phase 2：设备与 AI 开发文档](phase-2-device-ai.md)
- [Phase 3：生产化开发文档](phase-3-production.md)
- [前端 CI/CD](ci-cd.md)
- [前端总体设计蓝图](../frontend-design.md)
- [后端 API 约定](../backend-design.md)
