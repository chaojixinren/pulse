import { beforeEach, describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AppProvider, useApp } from './AppContext';

function Harness() {
  const { theme, toggleTheme, sidebarCollapsed, toggleSidebar } = useApp();
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <span data-testid="sidebar">{String(sidebarCollapsed)}</span>
      <button onClick={toggleTheme}>toggle-theme</button>
      <button onClick={toggleSidebar}>toggle-sidebar</button>
    </div>
  );
}

describe('AppContext', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('默认主题为 light 并设置到根元素', () => {
    render(
      <AppProvider>
        <Harness />
      </AppProvider>,
    );
    expect(screen.getByTestId('theme')).toHaveTextContent('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });

  it('toggleTheme 在 light/dark 间切换并持久化', async () => {
    render(
      <AppProvider>
        <Harness />
      </AppProvider>,
    );
    await userEvent.click(screen.getByRole('button', { name: 'toggle-theme' }));
    expect(screen.getByTestId('theme')).toHaveTextContent('dark');
    expect(localStorage.getItem('pulse_theme')).toBe('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('从 localStorage 恢复主题', () => {
    localStorage.setItem('pulse_theme', 'dark');
    render(
      <AppProvider>
        <Harness />
      </AppProvider>,
    );
    expect(screen.getByTestId('theme')).toHaveTextContent('dark');
  });

  it('toggleSidebar 切换侧边栏折叠态', async () => {
    render(
      <AppProvider>
        <Harness />
      </AppProvider>,
    );
    expect(screen.getByTestId('sidebar')).toHaveTextContent('false');
    await userEvent.click(screen.getByRole('button', { name: 'toggle-sidebar' }));
    expect(screen.getByTestId('sidebar')).toHaveTextContent('true');
  });
});
