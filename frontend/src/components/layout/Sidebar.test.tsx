import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { Sidebar } from './Sidebar';

const app = vi.hoisted(() => ({ sidebarCollapsed: false }));

vi.mock('@/contexts/AppContext', () => ({
  useApp: () => app,
}));

describe('Sidebar', () => {
  beforeEach(() => {
    app.sidebarCollapsed = false;
  });

  it('渲染导航项', () => {
    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>,
    );
    expect(screen.getByRole('link', { name: /身份管理/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /时间线/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /日报/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /周报/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /统计/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /账户设置/ })).toBeInTheDocument();
  });

  it('折叠时不渲染', () => {
    app.sidebarCollapsed = true;
    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>,
    );
    expect(screen.queryByRole('link', { name: /身份管理/ })).not.toBeInTheDocument();
  });
});
