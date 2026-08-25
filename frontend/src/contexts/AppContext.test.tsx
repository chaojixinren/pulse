import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AppProvider, useApp } from './AppContext';

const originalMatchMedia = window.matchMedia;

function stubMatchMedia(matches: boolean) {
  const mql = {
    matches,
    media: '(prefers-color-scheme: dark)',
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    onchange: null,
    dispatchEvent: vi.fn(),
  } as unknown as MediaQueryList;
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn(() => mql),
  });
}

function Harness() {
  const { theme, themePreference, setThemePreference, toggleTheme, sidebarCollapsed, toggleSidebar } =
    useApp();
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <span data-testid="preference">{themePreference}</span>
      <span data-testid="sidebar">{String(sidebarCollapsed)}</span>
      <button onClick={toggleTheme}>toggle-theme</button>
      <button onClick={() => setThemePreference('dark')}>set-dark</button>
      <button onClick={() => setThemePreference('system')}>set-system</button>
      <button onClick={toggleSidebar}>toggle-sidebar</button>
    </div>
  );
}

describe('AppContext', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  afterEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      writable: true,
      value: originalMatchMedia,
    });
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

  it('默认 themePreference 为 system', () => {
    render(
      <AppProvider>
        <Harness />
      </AppProvider>,
    );
    expect(screen.getByTestId('preference')).toHaveTextContent('system');
  });

  it('无偏好时跟随系统深色', () => {
    stubMatchMedia(true);
    render(
      <AppProvider>
        <Harness />
      </AppProvider>,
    );
    expect(screen.getByTestId('theme')).toHaveTextContent('dark');
    expect(screen.getByTestId('preference')).toHaveTextContent('system');
  });

  it('手动覆盖主题并持久化，可切回跟随系统', async () => {
    stubMatchMedia(true);
    render(
      <AppProvider>
        <Harness />
      </AppProvider>,
    );
    await userEvent.click(screen.getByRole('button', { name: 'set-dark' }));
    expect(screen.getByTestId('preference')).toHaveTextContent('dark');
    expect(localStorage.getItem('pulse_theme')).toBe('dark');

    await userEvent.click(screen.getByRole('button', { name: 'set-system' }));
    expect(screen.getByTestId('preference')).toHaveTextContent('system');
    expect(screen.getByTestId('theme')).toHaveTextContent('dark'); // 跟随系统深色
  });
});
