import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';

type Theme = 'light' | 'dark';
type ThemePreference = Theme | 'system';

interface AppContextValue {
  theme: Theme;
  themePreference: ThemePreference;
  setThemePreference: (pref: ThemePreference) => void;
  toggleTheme: () => void;
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
}

const THEME_KEY = 'pulse_theme';

const AppContext = createContext<AppContextValue | undefined>(undefined);

function systemTheme(): Theme {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'light';
}

function readPreference(): ThemePreference {
  const stored = localStorage.getItem(THEME_KEY);
  if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
  return 'system';
}

export function AppProvider({ children }: { children: ReactNode }) {
  const [themePreference, setThemePreferenceState] = useState<ThemePreference>(readPreference);
  const [systemThemeValue, setSystemThemeValue] = useState<Theme>(systemTheme);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  // 订阅系统偏好变化；仅在「跟随系统」时影响最终主题。
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    const apply = () => setSystemThemeValue(mql.matches ? 'dark' : 'light');
    apply();
    const onChange = () => apply();
    mql.addEventListener('change', onChange);
    return () => mql.removeEventListener('change', onChange);
  }, []);

  const theme: Theme = themePreference === 'system' ? systemThemeValue : themePreference;

  const setThemePreference = (pref: ThemePreference) => {
    localStorage.setItem(THEME_KEY, pref);
    setThemePreferenceState(pref);
  };

  const toggleTheme = () => {
    setThemePreference(theme === 'light' ? 'dark' : 'light');
  };

  const toggleSidebar = () => {
    setSidebarCollapsed((prev) => !prev);
  };

  // 主题以 data-theme 属性作用到根元素。
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  const value: AppContextValue = {
    theme,
    themePreference,
    setThemePreference,
    toggleTheme,
    sidebarCollapsed,
    toggleSidebar,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useApp(): AppContextValue {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useApp must be used within AppProvider');
  }
  return context;
}
