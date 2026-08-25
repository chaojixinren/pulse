import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AxiosAdapter, AxiosRequestConfig } from 'axios';
import { http } from './api';
import { ApiError } from '@/types/api.types';
import { storage } from '@/utils/storage';

// 通过自定义 axios adapter 驱动真实拦截器链：
// - 请求拦截器把 token 附加到 Authorization；
// - 响应拦截器把 {code, message, data} 解包 / 抛 ApiError。

const ok = (data: unknown) => ({
  data: { code: 0, message: 'ok', data },
  status: 200,
  statusText: 'OK',
  headers: {},
  config: {},
});

const asAdapter = (fn: (config: AxiosRequestConfig) => Promise<unknown>): AxiosAdapter =>
  fn as unknown as AxiosAdapter;

const withResponse = (response: unknown): AxiosAdapter => asAdapter(() => Promise.resolve(response));
const withError = (error: unknown): AxiosAdapter => asAdapter(() => Promise.reject(error));

function stubLocation() {
  const replace = vi.fn();
  const original = window.location;
  Object.defineProperty(window, 'location', {
    configurable: true,
    writable: true,
    value: { ...original, replace, pathname: '/timeline' },
  });
  return {
    replace,
    restore: () =>
      Object.defineProperty(window, 'location', {
        configurable: true,
        writable: true,
        value: original,
      }),
  };
}

describe('ApiError', () => {
  it('携带 code 与 status', () => {
    const err = new ApiError('失败', 1001, 400);
    expect(err).toBeInstanceOf(Error);
    expect(err.name).toBe('ApiError');
    expect(err.message).toBe('失败');
    expect(err.code).toBe(1001);
    expect(err.status).toBe(400);
  });
});

describe('api 请求拦截器', () => {
  beforeEach(() => storage.clear());

  it('存在 token 时附加 Bearer Authorization', async () => {
    storage.setTokens('secret-token', 'refresh-token');
    let authHeader: unknown;
    await http.get('/x', {
      adapter: asAdapter((config) => {
        authHeader = (config.headers as Record<string, unknown> | undefined)?.Authorization;
        return Promise.resolve(ok(null));
      }),
    });
    expect(authHeader).toBe('Bearer secret-token');
  });

  it('无 token 时不附加 Authorization', async () => {
    let authHeader: unknown;
    await http.get('/x', {
      adapter: asAdapter((config) => {
        authHeader = (config.headers as Record<string, unknown> | undefined)?.Authorization;
        return Promise.resolve(ok(null));
      }),
    });
    expect(authHeader).toBeUndefined();
  });
});

describe('api 响应拦截器', () => {
  it('code=0 时解包并返回业务 data', async () => {
    const result = await http.get<{ name: string }>('/x', {
      adapter: withResponse(ok({ name: 'pulse' })),
    });
    expect(result).toEqual({ name: 'pulse' });
  });

  it('code 非 0 时拒绝为 ApiError', async () => {
    const resp = {
      data: { code: 1001, message: '业务错误' },
      status: 200,
      statusText: 'OK',
      headers: {},
      config: {},
    };
    await expect(http.get('/x', { adapter: withResponse(resp) })).rejects.toMatchObject({
      name: 'ApiError',
      message: '业务错误',
      code: 1001,
      status: 200,
    });
  });

  it('无 response 的网络错误映射为 ApiError', async () => {
    await expect(
      http.get('/x', { adapter: withError({ message: 'Network Error', config: {} }) }),
    ).rejects.toMatchObject({ name: 'ApiError', message: 'Network Error', code: 0, status: 0 });
  });

  it('HTTP 错误携带响应体 message 映射为 ApiError', async () => {
    await expect(
      http.get('/x', {
        adapter: withError({
          message: 'Request failed with status code 500',
          response: { status: 500, data: { code: 500, message: '服务器错误' } },
          config: {},
        }),
      }),
    ).rejects.toMatchObject({ name: 'ApiError', message: '服务器错误', code: 500, status: 500 });
  });
});

describe('api 401 处理', () => {
  beforeEach(() => storage.clear());

  it('带 token 的 401 清理本地 token 并跳转登录页', async () => {
    storage.setTokens('expired', 'refresh');
    const location = stubLocation();
    try {
      await expect(
        http.get('/x', {
          adapter: withError({
            message: 'Request failed with status code 401',
            response: { status: 401, data: { code: 401, message: '未授权' } },
            config: {},
          }),
        }),
      ).rejects.toMatchObject({ status: 401 });
      expect(storage.getAccessToken()).toBeNull();
      expect(storage.getRefreshToken()).toBeNull();
      expect(location.replace).toHaveBeenCalledWith('/auth/login');
    } finally {
      location.restore();
    }
  });

  it('不带 token 的 401 仅拒绝、不跳转', async () => {
    const location = stubLocation();
    try {
      await expect(
        http.get('/x', {
          adapter: withError({
            message: 'Request failed with status code 401',
            response: { status: 401, data: { code: 401, message: '未授权' } },
            config: {},
          }),
        }),
      ).rejects.toMatchObject({ status: 401 });
      expect(location.replace).not.toHaveBeenCalled();
    } finally {
      location.restore();
    }
  });
});

describe('http 方法', () => {
  it('post/put/delete 均可驱动拦截器链', async () => {
    const result = await http.post<{ ok: boolean }>(
      '/x',
      { a: 1 },
      { adapter: withResponse(ok({ ok: true })) },
    );
    expect(result).toEqual({ ok: true });

    const putResult = await http.put<{ ok: boolean }>(
      '/x',
      { a: 1 },
      { adapter: withResponse(ok({ ok: true })) },
    );
    expect(putResult).toEqual({ ok: true });

    const delResult = await http.delete('/x', { adapter: withResponse(ok(undefined)) });
    expect(delResult).toBeUndefined();
  });
});
