import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AxiosAdapter, AxiosRequestConfig } from 'axios';
import { client, http } from './api';
import { ApiError } from '@/types/api.types';
import { storage } from '@/utils/storage';

// 通过自定义 axios adapter 驱动真实拦截器链：
// - 请求拦截器把 token 附加到 Authorization；
// - 响应拦截器把 {code, message, data} 解包 / 抛 ApiError；
// - 401 时尝试 access token 自动续期并重放。

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

const unauthorized = (config: unknown) => ({
  message: 'Request failed with status code 401',
  response: { status: 401, data: { code: 401, message: '未授权' } },
  config,
});

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

  it('带 token 的 401 续期失败时清理本地 token 并跳转登录页', async () => {
    storage.setTokens('expired', 'refresh');
    const location = stubLocation();
    const prevAdapter = client.defaults.adapter;
    // 让刷新接口失败，从而走「清理 + 跳转」分支。
    client.defaults.adapter = asAdapter((config) =>
      Promise.reject({
        message: 'refresh failed',
        response: { status: 401, data: { code: 401, message: '刷新失败' } },
        config,
      }),
    );
    try {
      await expect(
        http.get('/x', { adapter: withError(unauthorized({})) }),
      ).rejects.toMatchObject({ status: 401 });
      expect(storage.getAccessToken()).toBeNull();
      expect(storage.getRefreshToken()).toBeNull();
      expect(location.replace).toHaveBeenCalledWith('/auth/login');
    } finally {
      location.restore();
      client.defaults.adapter = prevAdapter;
    }
  });

  it('不带 token 的 401 仅拒绝、不跳转', async () => {
    const location = stubLocation();
    try {
      await expect(
        http.get('/x', { adapter: withError(unauthorized({})) }),
      ).rejects.toMatchObject({ status: 401 });
      expect(location.replace).not.toHaveBeenCalled();
    } finally {
      location.restore();
    }
  });

  it('401 自动续期并重放原请求', async () => {
    storage.setTokens('expired', 'refresh-token');
    const location = stubLocation();
    const prevAdapter = client.defaults.adapter;
    let calls = 0;
    const adapter = asAdapter((config) => {
      if (String(config.url ?? '').includes('/auth/refresh')) {
        return Promise.resolve(ok({ access_token: 'new-token', refresh_token: 'new-refresh' }));
      }
      calls += 1;
      if (calls === 1) return Promise.reject(unauthorized(config));
      return Promise.resolve(ok({ name: 'pulse' }));
    });
    client.defaults.adapter = adapter;
    try {
      const result = await http.get<{ name: string }>('/x');
      expect(result).toEqual({ name: 'pulse' });
      expect(storage.getAccessToken()).toBe('new-token');
      expect(storage.getRefreshToken()).toBe('new-refresh');
      expect(location.replace).not.toHaveBeenCalled();
    } finally {
      location.restore();
      client.defaults.adapter = prevAdapter;
    }
  });

  it('并发 401 只触发一次刷新', async () => {
    storage.setTokens('expired', 'refresh-token');
    const location = stubLocation();
    const prevAdapter = client.defaults.adapter;
    let refreshCalls = 0;
    const callsPerUrl = new Map<string, number>();
    const adapter = asAdapter((config) => {
      const url = String(config.url ?? '');
      if (url.includes('/auth/refresh')) {
        refreshCalls += 1;
        return new Promise((resolve) =>
          setTimeout(() => resolve(ok({ access_token: 'new-token', refresh_token: 'new-refresh' })), 20),
        );
      }
      const n = (callsPerUrl.get(url) ?? 0) + 1;
      callsPerUrl.set(url, n);
      if (n === 1) return Promise.reject(unauthorized(config));
      return Promise.resolve(ok({ url }));
    });
    client.defaults.adapter = adapter;
    try {
      const [a, b] = await Promise.all([http.get('/a'), http.get('/b')]);
      expect(a).toEqual({ url: '/a' });
      expect(b).toEqual({ url: '/b' });
      expect(refreshCalls).toBe(1);
      expect(location.replace).not.toHaveBeenCalled();
    } finally {
      location.restore();
      client.defaults.adapter = prevAdapter;
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
