import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { RequireAuth } from './RequireAuth';

const auth = vi.hoisted(() => ({ isAuthenticated: false, loading: false }));

vi.mock('@/contexts/AuthContext', () => ({
  useAuth: () => auth,
}));

vi.mock('@/components/layout/Layout', () => ({
  Layout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="layout">{children}</div>
  ),
}));

function renderGuard() {
  return render(
    <MemoryRouter initialEntries={['/protected']}>
      <Routes>
        <Route path="/auth/login" element={<div>登录页</div>} />
        <Route element={<RequireAuth />}>
          <Route path="/protected" element={<div>受保护内容</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe('RequireAuth 路由守卫', () => {
  beforeEach(() => {
    auth.isAuthenticated = false;
    auth.loading = false;
  });

  it('加载中渲染全屏加载态', () => {
    auth.loading = true;
    renderGuard();
    expect(screen.queryByText('受保护内容')).not.toBeInTheDocument();
    expect(document.querySelector('.loading-fullscreen')).not.toBeNull();
  });

  it('未登录跳转登录页', () => {
    auth.isAuthenticated = false;
    renderGuard();
    expect(screen.getByText('登录页')).toBeInTheDocument();
    expect(screen.queryByText('受保护内容')).not.toBeInTheDocument();
  });

  it('已登录渲染受保护内容（带布局）', () => {
    auth.isAuthenticated = true;
    renderGuard();
    expect(screen.getByTestId('layout')).toBeInTheDocument();
    expect(screen.getByText('受保护内容')).toBeInTheDocument();
  });
});
