import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider, useAuth } from './AuthContext';
import { authService } from '@/services/auth.service';
import { storage } from '@/utils/storage';

vi.mock('@/services/auth.service', () => ({
  authService: {
    register: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
    me: vi.fn(),
  },
}));

const user = { id: 'u1', email: 'a@b.com', name: 'Alice', created_at: '', updated_at: '' };

function Harness() {
  const { user, isAuthenticated, loading, login, register, logout } = useAuth();
  return (
    <div>
      <span data-testid="loading">{String(loading)}</span>
      <span data-testid="auth">{String(isAuthenticated)}</span>
      <span data-testid="user">{user?.name ?? 'none'}</span>
      <button onClick={() => login('a@b.com', 'password')}>login</button>
      <button onClick={() => register('a@b.com', 'password', 'Alice')}>register</button>
      <button onClick={() => logout()}>logout</button>
    </div>
  );
}

describe('AuthContext', () => {
  beforeEach(() => {
    storage.clear();
    vi.mocked(authService.me).mockReset();
    vi.mocked(authService.login).mockReset();
    vi.mocked(authService.register).mockReset();
    vi.mocked(authService.logout).mockReset();
  });

  it('无本地 token 时初始化完成且未登录', async () => {
    vi.mocked(authService.me).mockResolvedValue(user);
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
    expect(screen.getByTestId('auth')).toHaveTextContent('false');
    expect(authService.me).not.toHaveBeenCalled();
  });

  it('存在本地 token 时通过 me() 恢复会话', async () => {
    storage.setTokens('access', 'refresh');
    vi.mocked(authService.me).mockResolvedValue(user);
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
    expect(authService.me).toHaveBeenCalled();
    expect(screen.getByTestId('auth')).toHaveTextContent('true');
    expect(screen.getByTestId('user')).toHaveTextContent('Alice');
  });

  it('me() 失败时清理本地 token', async () => {
    storage.setTokens('access', 'refresh');
    vi.mocked(authService.me).mockRejectedValue(new Error('unauthorized'));
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
    expect(storage.getAccessToken()).toBeNull();
    expect(screen.getByTestId('auth')).toHaveTextContent('false');
  });

  it('login 写入 token 并设置用户', async () => {
    vi.mocked(authService.me).mockResolvedValue(user);
    vi.mocked(authService.login).mockResolvedValue({ access_token: 'access', refresh_token: 'refresh' });
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
    await userEvent.click(screen.getByRole('button', { name: 'login' }));

    await waitFor(() => expect(screen.getByTestId('auth')).toHaveTextContent('true'));
    expect(storage.getAccessToken()).toBe('access');
    expect(storage.getRefreshToken()).toBe('refresh');
    expect(authService.login).toHaveBeenCalledWith({ email: 'a@b.com', password: 'password' });
  });

  it('register 先注册再自动登录', async () => {
    vi.mocked(authService.me).mockResolvedValue(user);
    vi.mocked(authService.register).mockResolvedValue(user);
    vi.mocked(authService.login).mockResolvedValue({ access_token: 'access', refresh_token: 'refresh' });
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
    await userEvent.click(screen.getByRole('button', { name: 'register' }));

    await waitFor(() => expect(screen.getByTestId('auth')).toHaveTextContent('true'));
    expect(authService.register).toHaveBeenCalledWith({ email: 'a@b.com', password: 'password', name: 'Alice' });
    expect(authService.login).toHaveBeenCalled();
  });

  it('logout 调用后端并清理本地状态', async () => {
    storage.setTokens('access', 'refresh');
    vi.mocked(authService.me).mockResolvedValue(user);
    vi.mocked(authService.logout).mockResolvedValue(undefined);
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
    await userEvent.click(screen.getByRole('button', { name: 'logout' }));

    await waitFor(() => expect(screen.getByTestId('auth')).toHaveTextContent('false'));
    expect(authService.logout).toHaveBeenCalledWith('refresh');
    expect(storage.getAccessToken()).toBeNull();
  });
});
