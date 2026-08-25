import { describe, expect, it, vi } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import { useAsync } from './useAsync';

describe('useAsync', () => {
  it('加载成功后返回 data', async () => {
    const fn = vi.fn().mockResolvedValue({ value: 42 });
    const { result } = renderHook(() => useAsync(fn));
    expect(result.current.loading).toBe(true);

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toEqual({ value: 42 });
    expect(result.current.error).toBeNull();
  });

  it('加载失败写入 error', async () => {
    const fn = vi.fn().mockRejectedValue(new Error('加载失败'));
    const { result } = renderHook(() => useAsync(fn));

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe('加载失败');
    expect(result.current.data).toBeNull();
  });

  it('execute 可手动刷新并返回数据', async () => {
    const fn = vi
      .fn()
      .mockResolvedValueOnce({ value: 1 })
      .mockResolvedValueOnce({ value: 2 });
    const { result } = renderHook(() => useAsync(fn));

    await waitFor(() => expect(result.current.data).toEqual({ value: 1 }));

    let refreshed: unknown;
    await act(async () => {
      refreshed = await result.current.execute();
    });
    expect(refreshed).toEqual({ value: 2 });
    expect(result.current.data).toEqual({ value: 2 });
  });

  it('非 Error 异常转为通用文案', async () => {
    const fn = vi.fn().mockRejectedValue('boom');
    const { result } = renderHook(() => useAsync(fn));

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe('加载失败');
  });
});
