import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { Header } from './Header';

const mocks = vi.hoisted(() => ({
  user: null as { name: string } | null,
  logout: vi.fn(),
  theme: 'light' as 'light' | 'dark',
  toggleTheme: vi.fn(),
  sidebarCollapsed: false,
  toggleSidebar: vi.fn(),
}));

vi.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({ user: mocks.user, logout: mocks.logout }),
}));

vi.mock('@/contexts/AppContext', () => ({
  useApp: () => ({
    theme: mocks.theme,
    toggleTheme: mocks.toggleTheme,
    sidebarCollapsed: mocks.sidebarCollapsed,
    toggleSidebar: mocks.toggleSidebar,
  }),
}));

function renderHeader() {
  return render(
    <MemoryRouter>
      <Header />
    </MemoryRouter>,
  );
}

describe('Header', () => {
  beforeEach(() => {
    mocks.user = { name: 'Alice' };
    mocks.logout.mockReset();
    mocks.theme = 'light';
    mocks.toggleTheme.mockReset();
    mocks.sidebarCollapsed = false;
    mocks.toggleSidebar.mockReset();
  });

  it('渲染用户名', () => {
    renderHeader();
    expect(screen.getByText('Alice')).toBeInTheDocument();
  });

  it('切换主题', async () => {
    renderHeader();
    await userEvent.click(screen.getByTitle('切换主题'));
    expect(mocks.toggleTheme).toHaveBeenCalledTimes(1);
  });

  it('切换侧边栏', async () => {
    renderHeader();
    await userEvent.click(screen.getByRole('button', { name: /切换侧边栏/ }));
    expect(mocks.toggleSidebar).toHaveBeenCalledTimes(1);
  });

  it('登出调用 logout', async () => {
    mocks.logout.mockResolvedValue(undefined);
    renderHeader();
    await userEvent.click(screen.getByRole('button', { name: '退出登录' }));
    expect(mocks.logout).toHaveBeenCalledTimes(1);
  });
});
