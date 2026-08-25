import { beforeEach, describe, expect, it, vi } from 'vitest';
import { authService } from './auth.service';
import { http } from './api';

vi.mock('./api', () => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

const user = {
  id: 'u1',
  email: 'a@b.com',
  name: 'Alice',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('authService', () => {
  beforeEach(() => {
    vi.mocked(http.get).mockReset();
    vi.mocked(http.post).mockReset();
  });

  it('register 调用 POST /auth/register', async () => {
    vi.mocked(http.post).mockResolvedValue(user);
    await authService.register({ email: 'a@b.com', password: '12345678', name: 'Alice' });
    expect(http.post).toHaveBeenCalledWith('/auth/register', {
      email: 'a@b.com',
      password: '12345678',
      name: 'Alice',
    });
  });

  it('login 调用 POST /auth/login', async () => {
    vi.mocked(http.post).mockResolvedValue({ access_token: 'a', refresh_token: 'r' });
    const pair = await authService.login({ email: 'a@b.com', password: '12345678' });
    expect(http.post).toHaveBeenCalledWith('/auth/login', { email: 'a@b.com', password: '12345678' });
    expect(pair).toEqual({ access_token: 'a', refresh_token: 'r' });
  });

  it('logout 调用 POST /auth/logout', async () => {
    vi.mocked(http.post).mockResolvedValue(undefined);
    await authService.logout('r');
    expect(http.post).toHaveBeenCalledWith('/auth/logout', { refresh_token: 'r' });
  });

  it('refresh 调用 POST /auth/refresh', async () => {
    vi.mocked(http.post).mockResolvedValue({ access_token: 'a2', refresh_token: 'r2' });
    await authService.refresh('r');
    expect(http.post).toHaveBeenCalledWith('/auth/refresh', { refresh_token: 'r' });
  });

  it('me 调用 GET /auth/me', async () => {
    vi.mocked(http.get).mockResolvedValue(user);
    const result = await authService.me();
    expect(http.get).toHaveBeenCalledWith('/auth/me');
    expect(result).toEqual(user);
  });
});
