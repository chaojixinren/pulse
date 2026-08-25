import axios, {
  AxiosError,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from 'axios';
import { ApiError, ApiResponse, TokenPair } from '@/types/api.types';
import { storage } from '@/utils/storage';

export const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 30000,
});

// 请求拦截器：附加 Bearer token
client.interceptors.request.use((config) => {
  const token = storage.getAccessToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// 这些认证相关接口自身返回 401 时不应触发续期重放，避免死循环。
const REFRESH_EXCLUDED = new Set(['/auth/refresh', '/auth/login', '/auth/register']);

// 并发 401 只触发一次刷新（单飞）。
let refreshing: Promise<string | null> | null = null;

async function tryRefresh(): Promise<string | null> {
  const rt = storage.getRefreshToken();
  if (!rt) return null;
  if (!refreshing) {
    refreshing = (client.post('/auth/refresh', { refresh_token: rt }) as Promise<TokenPair>)
      .then((pair) => {
        storage.setTokens(pair.access_token, pair.refresh_token);
        return pair.access_token;
      })
      .catch(() => {
        storage.clear();
        return null;
      })
      .finally(() => {
        refreshing = null;
      });
  }
  return refreshing;
}

type RetriableConfig = InternalAxiosRequestConfig & { _retried?: boolean };

// 响应拦截器：统一解析 {code, message, data}，并对 401 做 access token 自动续期。
client.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse;
    if (body.code !== 0) {
      return Promise.reject(new ApiError(body.message || '请求失败', body.code, response.status));
    }
    // 解包业务数据；这里用 unknown 修正 axios 的静态返回类型，
    // 最终泛型由 http 方法在调用处确定。
    return body.data as unknown as AxiosResponse;
  },
  async (error: AxiosError<ApiResponse>) => {
    const status = error.response?.status ?? 0;
    const original = error.config as RetriableConfig | undefined;
    const hadToken = Boolean(storage.getAccessToken());

    if (status === 401 && hadToken) {
      const isExcluded = original?.url ? REFRESH_EXCLUDED.has(original.url) : false;
      if (original && !original._retried && !isExcluded) {
        original._retried = true;
        const token = await tryRefresh();
        if (token) {
          if (original.headers) {
            original.headers.Authorization = `Bearer ${token}`;
          }
          return client(original); // 重放原请求
        }
      }
      // 续期失败或不可续期：清理本地 token 并跳转登录页。
      storage.clear();
      if (window.location.pathname !== '/auth/login') {
        window.location.replace('/auth/login');
      }
    }

    const body = error.response?.data;
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
