# Pulse 前端设计文档

## 1. 项目概述

Pulse 是一个智能身份空间系统的 Web 前端应用，为用户提供身份管理、时间线查看、设备管理和报告分析等功能。本文档描述了前端架构设计、技术选型和实现方案。

### 1.1 技术栈

- **核心框架**: React 18+
- **开发语言**: TypeScript 5.3+
- **构建工具**: Vite 5.0+
- **路由管理**: React Router 6.21+
- **HTTP 客户端**: Axios 1.6+
- **代码规范**: ESLint + TypeScript ESLint

### 1.2 开发环境要求

- Node.js 18+
- npm 或 yarn
- 现代浏览器（Chrome 90+, Firefox 88+, Safari 14+, Edge 90+）

---

## 2. 项目架构设计

### 2.1 目录结构

```
frontend/
├── public/                 # 静态资源
│   ├── favicon.ico
│   └── index.html
├── src/                    # 源代码
│   ├── components/         # 可复用组件
│   │   ├── common/         # 通用组件（Button, Input, Modal 等）
│   │   ├── layout/         # 布局组件（Header, Sidebar, Footer）
│   │   └── business/       # 业务组件（IdentityCard, TimelineItem 等）
│   ├── pages/              # 页面组件
│   │   ├── Auth/           # 认证相关页面
│   │   ├── Identity/       # 身份管理页面
│   │   ├── Timeline/       # 时间线页面
│   │   ├── Device/         # 设备管理页面
│   │   └── Report/         # 报告页面
│   ├── services/           # API 服务层
│   │   ├── api.ts          # API 配置和拦截器
│   │   ├── auth.service.ts
│   │   ├── identity.service.ts
│   │   ├── timeline.service.ts
│   │   ├── device.service.ts
│   │   └── report.service.ts
│   ├── hooks/              # 自定义 Hooks
│   │   ├── useAuth.ts
│   │   ├── useIdentity.ts
│   │   └── useAsync.ts
│   ├── types/              # TypeScript 类型定义
│   │   ├── api.types.ts
│   │   ├── identity.types.ts
│   │   ├── timeline.types.ts
│   │   └── common.types.ts
│   ├── utils/              # 工具函数
│   │   ├── storage.ts      # 本地存储封装
│   │   ├── date.ts         # 日期处理
│   │   └── validator.ts    # 数据验证
│   ├── styles/             # 样式文件
│   │   ├── variables.css   # CSS 变量
│   │   ├── global.css      # 全局样式
│   │   └── mixins.css      # CSS Mixins
│   ├── App.tsx             # 根组件
│   ├── main.tsx            # 入口文件
│   └── vite-env.d.ts       # Vite 类型声明
├── .eslintrc.json          # ESLint 配置
├── tsconfig.json           # TypeScript 配置
├── vite.config.ts          # Vite 配置
└── package.json            # 项目依赖
```

### 2.2 模块划分原则

#### 2.2.1 分层架构

```
┌─────────────────────────────────────┐
│         Presentation Layer          │
│    (Pages & Components)             │
├─────────────────────────────────────┤
│         Business Logic Layer        │
│    (Custom Hooks & Utils)           │
├─────────────────────────────────────┤
│         Service Layer               │
│    (API Services)                   │
├─────────────────────────────────────┤
│         Data Layer                  │
│    (Types & Interfaces)             │
└─────────────────────────────────────┘
```

#### 2.2.2 组件分类

- **页面组件 (Pages)**: 路由级组件，负责页面布局和数据组装
- **布局组件 (Layout)**: 应用框架组件，如头部、侧边栏
- **业务组件 (Business)**: 特定业务场景的组件
- **通用组件 (Common)**: 可跨项目复用的基础组件

---

## 3. 页面设计

### 3.1 页面路由规划

```typescript
// src/router/routes.ts
import { RouteObject } from 'react-router-dom';

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <Layout />,
    children: [
      {
        index: true,
        element: <Navigate to="/identity" replace />
      },
      {
        path: 'identity',
        element: <IdentityPage />,
        children: [
          {
            path: ':identityId',
            element: <IdentityDetailPage />
          }
        ]
      },
      {
        path: 'timeline',
        element: <TimelinePage />
      },
      {
        path: 'devices',
        element: <DevicePage />
      },
      {
        path: 'reports',
        element: <ReportPage />,
        children: [
          {
            path: ':reportId',
            element: <ReportDetailPage />
          }
        ]
      }
    ]
  },
  {
    path: '/auth',
    children: [
      {
        path: 'login',
        element: <LoginPage />
      },
      {
        path: 'register',
        element: <RegisterPage />
      }
    ]
  },
  {
    path: '*',
    element: <NotFoundPage />
  }
];
```

### 3.2 主要页面说明

| 页面 | 路径 | 功能描述 |
|------|------|----------|
| 身份管理 | `/identity` | 展示和管理用户身份信息 |
| 身份详情 | `/identity/:id` | 查看特定身份的详细信息 |
| 时间线 | `/timeline` | 展示用户活动时间线 |
| 设备管理 | `/devices` | 管理关联设备 |
| 报告列表 | `/reports` | 展示报告列表 |
| 报告详情 | `/reports/:id` | 查看报告详细内容 |
| 登录 | `/auth/login` | 用户登录 |
| 注册 | `/auth/register` | 用户注册 |

### 3.3 页面组件示例

```typescript
// src/pages/Identity/IdentityPage.tsx
import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Identity } from '@/types/identity.types';
import { identityService } from '@/services/identity.service';
import { IdentityCard } from '@/components/business/IdentityCard';
import { Loading } from '@/components/common/Loading';
import { ErrorMessage } from '@/components/common/ErrorMessage';

export const IdentityPage: React.FC = () => {
  const [identities, setIdentities] = useState<Identity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    loadIdentities();
  }, []);

  const loadIdentities = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await identityService.getIdentities();
      setIdentities(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
    } finally {
      setLoading(false);
    }
  };

  const handleIdentityClick = (id: string) => {
    navigate(`/identity/${id}`);
  };

  if (loading) return <Loading />;
  if (error) return <ErrorMessage message={error} onRetry={loadIdentities} />;

  return (
    <div className="identity-page">
      <div className="page-header">
        <h1>我的身份</h1>
        <button onClick={() => navigate('/identity/create')}>
          创建新身份
        </button>
      </div>
      <div className="identity-grid">
        {identities.map(identity => (
          <IdentityCard
            key={identity.id}
            identity={identity}
            onClick={() => handleIdentityClick(identity.id)}
          />
        ))}
      </div>
    </div>
  );
};
```

---

## 4. 状态管理方案

### 4.1 状态管理策略

Pulse 项目采用**轻量级状态管理**策略，不引入 Redux、MobX 等重量级状态管理库。主要依靠以下方案：

1. **React Context API**: 管理全局状态（用户认证、主题配置等）
2. **Custom Hooks**: 封装可复用的状态逻辑
3. **组件 State**: 管理局部 UI 状态

### 4.2 全局状态管理

#### 4.2.1 认证状态 (AuthContext)

```typescript
// src/contexts/AuthContext.tsx
import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { User } from '@/types/api.types';
import { authService } from '@/services/auth.service';
import { storage } from '@/utils/storage';

interface AuthContextValue {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    initAuth();
  }, []);

  const initAuth = async () => {
    const token = storage.getToken();
    if (token) {
      try {
        const userData = await authService.getCurrentUser();
        setUser(userData);
      } catch (error) {
        storage.removeToken();
      }
    }
    setLoading(false);
  };

  const login = async (email: string, password: string) => {
    const { user, token } = await authService.login(email, password);
    storage.setToken(token);
    setUser(user);
  };

  const logout = async () => {
    await authService.logout();
    storage.removeToken();
    setUser(null);
  };

  const refreshUser = async () => {
    const userData = await authService.getCurrentUser();
    setUser(userData);
  };

  return (
    <AuthContext.Provider 
      value={{
        user,
        isAuthenticated: !!user,
        loading,
        login,
        logout,
        refreshUser
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
```

#### 4.2.2 应用配置状态 (AppContext)

```typescript
// src/contexts/AppContext.tsx
import React, { createContext, useContext, useState, ReactNode } from 'react';

type Theme = 'light' | 'dark';

interface AppContextValue {
  theme: Theme;
  toggleTheme: () => void;
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
}

const AppContext = createContext<AppContextValue | undefined>(undefined);

export const AppProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [theme, setTheme] = useState<Theme>('light');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const toggleTheme = () => {
    setTheme(prev => prev === 'light' ? 'dark' : 'light');
  };

  const toggleSidebar = () => {
    setSidebarCollapsed(prev => !prev);
  };

  return (
    <AppContext.Provider value={{ theme, toggleTheme, sidebarCollapsed, toggleSidebar }}>
      {children}
    </AppContext.Provider>
  );
};

export const useApp = () => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useApp must be used within AppProvider');
  }
  return context;
};
```

### 4.3 自定义 Hooks

#### 4.3.1 异步请求 Hook

```typescript
// src/hooks/useAsync.ts
import { useState, useEffect, useCallback } from 'react';

interface AsyncState<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
}

export function useAsync<T>(
  asyncFunction: () => Promise<T>,
  immediate = true
) {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: immediate,
    error: null
  });

  const execute = useCallback(async () => {
    setState({ data: null, loading: true, error: null });
    try {
      const data = await asyncFunction();
      setState({ data, loading: false, error: null });
      return data;
    } catch (error) {
      setState({ data: null, loading: false, error: error as Error });
      throw error;
    }
  }, [asyncFunction]);

  useEffect(() => {
    if (immediate) {
      execute();
    }
  }, [execute, immediate]);

  return { ...state, execute };
}
```

#### 4.3.2 身份数据 Hook

```typescript
// src/hooks/useIdentity.ts
import { useState, useEffect } from 'react';
import { Identity } from '@/types/identity.types';
import { identityService } from '@/services/identity.service';

export function useIdentity(identityId: string | undefined) {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!identityId) return;

    const loadIdentity = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await identityService.getIdentityById(identityId);
        setIdentity(data);
      } catch (err) {
        setError(err as Error);
      } finally {
        setLoading(false);
      }
    };

    loadIdentity();
  }, [identityId]);

  const refresh = async () => {
    if (!identityId) return;
    const data = await identityService.getIdentityById(identityId);
    setIdentity(data);
  };

  return { identity, loading, error, refresh };
}
```

---

## 5. API 集成方案

### 5.1 API 配置

```typescript
// src/services/api.ts
import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig } from 'axios';
import { storage } from '@/utils/storage';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api';

// 创建 axios 实例
export const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器：添加认证 token
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = storage.getToken();
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  }
);

// 响应拦截器：统一错误处理
api.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error: AxiosError) => {
    // 处理认证错误
    if (error.response?.status === 401) {
      storage.removeToken();
      window.location.href = '/auth/login';
      return Promise.reject(new Error('未授权，请重新登录'));
    }

    // 处理服务器错误
    if (error.response?.status === 500) {
      return Promise.reject(new Error('服务器错误，请稍后重试'));
    }

    // 处理网络错误
    if (!error.response) {
      return Promise.reject(new Error('网络连接失败'));
    }

    // 返回错误信息
    const message = (error.response?.data as any)?.message || error.message;
    return Promise.reject(new Error(message));
  }
);
```

### 5.2 API Service 层设计

#### 5.2.1 认证服务

```typescript
// src/services/auth.service.ts
import { api } from './api';
import { User, LoginRequest, LoginResponse, RegisterRequest } from '@/types/api.types';

class AuthService {
  async login(email: string, password: string): Promise<LoginResponse> {
    const response = await api.post<LoginRequest, LoginResponse>('/auth/login', {
      email,
      password
    });
    return response;
  }

  async register(data: RegisterRequest): Promise<User> {
    const response = await api.post<RegisterRequest, User>('/auth/register', data);
    return response;
  }

  async logout(): Promise<void> {
    await api.post('/auth/logout');
  }

  async getCurrentUser(): Promise<User> {
    const response = await api.get<User>('/auth/me');
    return response;
  }

  async refreshToken(): Promise<{ token: string }> {
    const response = await api.post<any, { token: string }>('/auth/refresh');
    return response;
  }
}

export const authService = new AuthService();
```

#### 5.2.2 身份服务

```typescript
// src/services/identity.service.ts
import { api } from './api';
import { Identity, CreateIdentityRequest, UpdateIdentityRequest } from '@/types/identity.types';

class IdentityService {
  async getIdentities(): Promise<Identity[]> {
    const response = await api.get<Identity[]>('/identities');
    return response;
  }

  async getIdentityById(id: string): Promise<Identity> {
    const response = await api.get<Identity>(`/identities/${id}`);
    return response;
  }

  async createIdentity(data: CreateIdentityRequest): Promise<Identity> {
    const response = await api.post<CreateIdentityRequest, Identity>('/identities', data);
    return response;
  }

  async updateIdentity(id: string, data: UpdateIdentityRequest): Promise<Identity> {
    const response = await api.put<UpdateIdentityRequest, Identity>(`/identities/${id}`, data);
    return response;
  }

  async deleteIdentity(id: string): Promise<void> {
    await api.delete(`/identities/${id}`);
  }
}

export const identityService = new IdentityService();
```

### 5.3 类型定义

```typescript
// src/types/api.types.ts
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

export interface User {
  id: string;
  email: string;
  username: string;
  createdAt: string;
  updatedAt: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  token: string;
}

export interface RegisterRequest {
  email: string;
  username: string;
  password: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}
```

```typescript
// src/types/identity.types.ts
export interface Identity {
  id: string;
  name: string;
  type: string;
  status: 'active' | 'inactive' | 'suspended';
  createdAt: string;
  updatedAt: string;
  metadata?: Record<string, any>;
}

export interface CreateIdentityRequest {
  name: string;
  type: string;
  metadata?: Record<string, any>;
}

export interface UpdateIdentityRequest {
  name?: string;
  type?: string;
  status?: 'active' | 'inactive' | 'suspended';
  metadata?: Record<string, any>;
}
```

### 5.4 本地存储封装

```typescript
// src/utils/storage.ts
const TOKEN_KEY = 'pulse_auth_token';

export const storage = {
  getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  },

  setToken(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
  },

  removeToken(): void {
    localStorage.removeItem(TOKEN_KEY);
  },

  clear(): void {
    localStorage.clear();
  }
};
```

---

## 6. UI/UX 设计原则

### 6.1 设计理念

- **简洁直观**: 界面清晰，减少认知负担
- **响应式设计**: 适配不同屏幕尺寸
- **一致性**: 统一的视觉语言和交互模式
- **可访问性**: 遵循 WCAG 2.1 标准
- **性能优先**: 快速响应，流畅体验

### 6.2 视觉设计规范

#### 6.2.1 颜色系统

```css
/* src/styles/variables.css */
:root {
  /* 主色调 */
  --color-primary: #3b82f6;
  --color-primary-hover: #2563eb;
  --color-primary-active: #1d4ed8;

  /* 语义色 */
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-error: #ef4444;
  --color-info: #3b82f6;

  /* 中性色 */
  --color-text-primary: #1f2937;
  --color-text-secondary: #6b7280;
  --color-text-disabled: #9ca3af;
  --color-border: #e5e7eb;
  --color-background: #ffffff;
  --color-background-secondary: #f9fafb;

  /* 阴影 */
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1);

  /* 圆角 */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;

  /* 间距 */
  --spacing-xs: 4px;
  --spacing-sm: 8px;
  --spacing-md: 16px;
  --spacing-lg: 24px;
  --spacing-xl: 32px;
}

/* 深色主题 */
[data-theme="dark"] {
  --color-text-primary: #f9fafb;
  --color-text-secondary: #d1d5db;
  --color-text-disabled: #6b7280;
  --color-border: #374151;
  --color-background: #111827;
  --color-background-secondary: #1f2937;
}
```

#### 6.2.2 排版系统

```css
/* src/styles/variables.css */
:root {
  /* 字体 */
  --font-family-base: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --font-family-mono: "SF Mono", Monaco, "Cascadia Code", monospace;

  /* 字号 */
  --font-size-xs: 12px;
  --font-size-sm: 14px;
  --font-size-base: 16px;
  --font-size-lg: 18px;
  --font-size-xl: 20px;
  --font-size-2xl: 24px;
  --font-size-3xl: 30px;

  /* 行高 */
  --line-height-tight: 1.25;
  --line-height-base: 1.5;
  --line-height-relaxed: 1.75;

  /* 字重 */
  --font-weight-normal: 400;
  --font-weight-medium: 500;
  --font-weight-semibold: 600;
  --font-weight-bold: 700;
}
```

### 6.3 交互设计规范

#### 6.3.1 加载状态

- 使用 Skeleton 占位符进行内容加载
- 长时间操作显示进度条
- 提供取消操作的选项

#### 6.3.2 错误处理

- 使用 Toast 通知显示操作结果
- 错误信息明确，提供重试选项
- 表单验证实时反馈

#### 6.3.3 确认操作

- 危险操作（删除、清空等）需要二次确认
- 使用 Modal 或 Popconfirm 组件

### 6.4 响应式断点

```typescript
// src/utils/responsive.ts
export const breakpoints = {
  mobile: 640,    // < 640px
  tablet: 768,    // 640px - 768px
  desktop: 1024,  // 768px - 1024px
  wide: 1280      // > 1280px
} as const;

export type Breakpoint = keyof typeof breakpoints;
```

---

## 7. 组件设计

### 7.1 通用组件库

#### 7.1.1 Button 组件

```typescript
// src/components/common/Button/Button.tsx
import React, { ButtonHTMLAttributes } from 'react';
import './Button.css';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
export type ButtonSize = 'small' | 'medium' | 'large';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  icon?: React.ReactNode;
  block?: boolean;
}

export const Button: React.FC<ButtonProps> = ({
  children,
  variant = 'primary',
  size = 'medium',
  loading = false,
  icon,
  block = false,
  disabled,
  className = '',
  ...props
}) => {
  const classNames = [
    'btn',
    `btn-${variant}`,
    `btn-${size}`,
    block ? 'btn-block' : '',
    loading ? 'btn-loading' : '',
    className
  ].filter(Boolean).join(' ');

  return (
    <button
      className={classNames}
      disabled={disabled || loading}
      {...props}
    >
      {loading && <span className="btn-spinner" />}
      {!loading && icon && <span className="btn-icon">{icon}</span>}
      {children && <span className="btn-text">{children}</span>}
    </button>
  );
};
```

```css
/* src/components/common/Button/Button.css */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border: none;
  border-radius: var(--radius-md);
  font-family: var(--font-family-base);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all 0.2s ease;
  outline: none;
}

.btn:focus-visible {
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.3);
}

.btn-primary {
  background-color: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background-color: var(--color-primary-hover);
}

.btn-secondary {
  background-color: var(--color-background-secondary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.btn-danger {
  background-color: var(--color-error);
  color: white;
}

.btn-ghost {
  background-color: transparent;
  color: var(--color-primary);
}

.btn-small {
  padding: 4px 12px;
  font-size: var(--font-size-sm);
}

.btn-medium {
  padding: 8px 16px;
  font-size: var(--font-size-base);
}

.btn-large {
  padding: 12px 24px;
  font-size: var(--font-size-lg);
}

.btn-block {
  width: 100%;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-loading {
  pointer-events: none;
}

.btn-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
```

#### 7.1.2 Input 组件

```typescript
// src/components/common/Input/Input.tsx
import React, { InputHTMLAttributes, forwardRef } from 'react';
import './Input.css';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
  prefix?: React.ReactNode;
  suffix?: React.ReactNode;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(({
  label,
  error,
  helperText,
  prefix,
  suffix,
  className = '',
  disabled,
  ...props
}, ref) => {
  const hasError = !!error;

  return (
    <div className={`input-wrapper ${className}`}>
      {label && (
        <label className="input-label">
          {label}
          {props.required && <span className="input-required">*</span>}
        </label>
      )}
      <div className={`input-container ${hasError ? 'input-error' : ''} ${disabled ? 'input-disabled' : ''}`}>
        {prefix && <span className="input-prefix">{prefix}</span>}
        <input
          ref={ref}
          className="input-field"
          disabled={disabled}
          aria-invalid={hasError}
          aria-describedby={error ? 'input-error' : helperText ? 'input-helper' : undefined}
          {...props}
        />
        {suffix && <span className="input-suffix">{suffix}</span>}
      </div>
      {error && <span className="input-error-text" id="input-error">{error}</span>}
      {!error && helperText && <span className="input-helper-text" id="input-helper">{helperText}</span>}
    </div>
  );
});

Input.displayName = 'Input';
```

#### 7.1.3 Modal 组件

```typescript
// src/components/common/Modal/Modal.tsx
import React, { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import './Modal.css';

export interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  width?: number | string;
  closeOnOverlayClick?: boolean;
  closeOnEscape?: boolean;
}

export const Modal: React.FC<ModalProps> = ({
  open,
  onClose,
  title,
  children,
  footer,
  width = 520,
  closeOnOverlayClick = true,
  closeOnEscape = true
}) => {
  const modalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    const handleEscape = (e: KeyboardEvent) => {
      if (closeOnEscape && e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [open, closeOnEscape, onClose]);

  if (!open) return null;

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (closeOnOverlayClick && e.target === e.currentTarget) {
      onClose();
    }
  };

  return createPortal(
    <div className="modal-overlay" onClick={handleOverlayClick}>
      <div
        ref={modalRef}
        className="modal-container"
        style={{ width: typeof width === 'number' ? `${width}px` : width }}
        role="dialog"
        aria-modal="true"
      >
        {title && (
          <div className="modal-header">
            <h3 className="modal-title">{title}</h3>
            <button className="modal-close" onClick={onClose} aria-label="关闭">
              ×
            </button>
          </div>
        )}
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>,
    document.body
  );
};
```

### 7.2 业务组件示例

#### 7.2.1 IdentityCard 组件

```typescript
// src/components/business/IdentityCard/IdentityCard.tsx
import React from 'react';
import { Identity } from '@/types/identity.types';
import './IdentityCard.css';

export interface IdentityCardProps {
  identity: Identity;
  onClick?: () => void;
}

export const IdentityCard: React.FC<IdentityCardProps> = ({ identity, onClick }) => {
  const statusColorMap: Record<Identity['status'], string> = {
    active: 'green',
    inactive: 'gray',
    suspended: 'red'
  };

  const statusTextMap: Record<Identity['status'], string> = {
    active: '活跃',
    inactive: '未激活',
    suspended: '已暂停'
  };

  return (
    <div className="identity-card" onClick={onClick}>
      <div className="identity-card-header">
        <h3 className="identity-card-title">{identity.name}</h3>
        <span 
          className="identity-card-status" 
          data-status={statusColorMap[identity.status]}
        >
          {statusTextMap[identity.status]}
        </span>
      </div>
      <div className="identity-card-body">
        <p className="identity-card-type">类型: {identity.type}</p>
        <p className="identity-card-date">
          创建于: {new Date(identity.createdAt).toLocaleDateString('zh-CN')}
        </p>
      </div>
    </div>
  );
};
```

### 7.3 布局组件

#### 7.3.1 主布局

```typescript
// src/components/layout/Layout.tsx
import React from 'react';
import { Outlet } from 'react-router-dom';
import { Header } from './Header';
import { Sidebar } from './Sidebar';
import { useApp } from '@/contexts/AppContext';
import './Layout.css';

export const Layout: React.FC = () => {
  const { sidebarCollapsed } = useApp();

  return (
    <div className="layout">
      <Header />
      <div className="layout-main">
        <Sidebar collapsed={sidebarCollapsed} />
        <main className="layout-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
};
```

---

## 8. 开发规范

### 8.1 代码风格

- 使用 ESLint + Prettier 进行代码格式化
- 组件使用函数式组件 + Hooks
- 优先使用 TypeScript 类型推断，避免过度类型标注
- 文件命名：PascalCase 用于组件，camelCase 用于工具函数

### 8.2 组件开发规范

```typescript
// ✅ 推荐写法
export const MyComponent: React.FC<MyComponentProps> = ({ prop1, prop2 }) => {
  // Hooks 放在顶部
  const [state, setState] = useState();
  const context = useContext(MyContext);
  
  // 副作用
  useEffect(() => {
    // ...
  }, []);
  
  // 事件处理函数
  const handleClick = () => {
    // ...
  };
  
  // 条件渲染的提前返回
  if (loading) return <Loading />;
  if (error) return <Error />;
  
  // 主渲染
  return (
    <div>
      {/* ... */}
    </div>
  );
};

// ❌ 不推荐
class MyComponent extends React.Component {
  // 避免使用类组件
}
```

### 8.3 性能优化建议

1. **组件懒加载**
```typescript
const ReportPage = lazy(() => import('@/pages/Report/ReportPage'));
```

2. **使用 memo 优化渲染**
```typescript
export const ExpensiveComponent = React.memo<Props>(({ data }) => {
  // ...
});
```

3. **使用 useMemo 和 useCallback**
```typescript
const memoizedValue = useMemo(() => computeExpensiveValue(a, b), [a, b]);
const memoizedCallback = useCallback(() => doSomething(a, b), [a, b]);
```

4. **虚拟列表**
```typescript
// 对于长列表，考虑使用虚拟化库如 react-window
```

### 8.4 测试策略

- 单元测试：使用 Vitest 测试工具函数和 Hooks
- 组件测试：使用 React Testing Library
- E2E 测试：使用 Playwright 或 Cypress

---

## 9. 构建与部署

### 9.1 环境变量配置

```bash
# .env.development
VITE_API_BASE_URL=http://localhost:8080/api

# .env.production
VITE_API_BASE_URL=https://api.pulse.example.com/api
```

### 9.2 构建命令

```bash
# 开发环境
npm run dev

# 类型检查
npm run type-check

# 代码检查
npm run lint

# 生产构建
npm run build

# 预览构建结果
npm run preview
```

### 9.3 部署方案

1. **静态资源部署**: 构建后的 `dist` 目录可部署到任何静态服务器
2. **CDN 加速**: 将静态资源上传到 CDN
3. **容器化部署**: 使用 Nginx 作为 Web 服务器

```dockerfile
# Dockerfile 示例
FROM node:18-alpine as builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

---

## 10. 未来规划

### 10.1 短期计划

- [ ] 完善单元测试覆盖率
- [ ] 添加国际化支持 (i18n)
- [ ] 实现深色模式
- [ ] 优化移动端响应式

### 10.2 长期计划

- [ ] 考虑迁移到 React Native 实现移动应用
- [ ] 引入 PWA 支持
- [ ] 实现离线功能
- [ ] 性能监控和错误追踪

---

## 附录

### A. 参考资源

- [React 官方文档](https://react.dev/)
- [TypeScript 官方文档](https://www.typescriptlang.org/)
- [Vite 官方文档](https://vitejs.dev/)
- [React Router 文档](https://reactrouter.com/)
- [Axios 文档](https://axios-http.com/)

### B. 项目贡献指南

详见项目根目录的 `CONTRIBUTING.md` 文件。

### C. 变更日志

详见项目根目录的 `CHANGELOG.md` 文件。

---

**文档版本**: v1.0.0  
**最后更新**: 2026-08-23  
**维护者**: Pulse 前端团队
