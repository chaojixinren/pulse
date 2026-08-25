import { beforeEach, describe, expect, it } from 'vitest';
import { storage } from './storage';

describe('storage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('存取 access/refresh token', () => {
    storage.setTokens('access-1', 'refresh-1');
    expect(storage.getAccessToken()).toBe('access-1');
    expect(storage.getRefreshToken()).toBe('refresh-1');
  });

  it('未设置时返回 null', () => {
    expect(storage.getAccessToken()).toBeNull();
    expect(storage.getRefreshToken()).toBeNull();
  });

  it('setAccessToken 只更新 access token', () => {
    storage.setTokens('access-1', 'refresh-1');
    storage.setAccessToken('access-2');
    expect(storage.getAccessToken()).toBe('access-2');
    expect(storage.getRefreshToken()).toBe('refresh-1');
  });

  it('clear 清空全部 token', () => {
    storage.setTokens('access-1', 'refresh-1');
    storage.clear();
    expect(storage.getAccessToken()).toBeNull();
    expect(storage.getRefreshToken()).toBeNull();
  });
});
