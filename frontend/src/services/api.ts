import axios, { AxiosError, AxiosRequestConfig, AxiosResponse } from 'axios';
import { ApiError, ApiResponse } from '@/types/api.types';
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
    // 解包业务数据；这里用 unknown 修正 axios 的静态返回类型，
    // 最终泛型由 http 方法在调用处确定。
    return body.data as unknown as AxiosResponse;
  },
  (error: AxiosError<ApiResponse>) => {
    const status = error.response?.status ?? 0;
    const body = error.response?.data;
    const hadToken = Boolean(storage.getAccessToken());

    // Phase 1 不做 access token 自动续期：带 token 的请求 401 时清理并跳转登录页。
    if (status === 401 && hadToken) {
      storage.clear();
      if (window.location.pathname !== '/auth/login') {
        window.location.replace('/auth/login');
      }
    }

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
