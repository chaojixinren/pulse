import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { authService } from '@/services/auth.service';
import { User } from '@/types/api.types';
import { storage } from '@/utils/storage';

interface AuthContextValue {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const init = async () => {
      if (storage.getAccessToken()) {
        try {
          const userData = await authService.me();
          if (!cancelled) setUser(userData);
        } catch {
          // token 失效或网络错误：清理本地 token，交由路由守卫跳转登录页。
          storage.clear();
        }
      }
      if (!cancelled) setLoading(false);
    };

    init();

    return () => {
      cancelled = true;
    };
  }, []);

  const login = async (email: string, password: string) => {
    const pair = await authService.login({ email, password });
    storage.setTokens(pair.access_token, pair.refresh_token);
    const userData = await authService.me();
    setUser(userData);
  };

  const register = async (email: string, password: string, name: string) => {
    await authService.register({ email, password, name });
    // 注册后自动登录
    await login(email, password);
  };

  const logout = async () => {
    const refreshToken = storage.getRefreshToken();
    try {
      if (refreshToken) await authService.logout(refreshToken);
    } catch {
      // 登出请求失败也继续清理本地状态
    } finally {
      storage.clear();
      setUser(null);
    }
  };

  const refreshUser = async () => {
    const userData = await authService.me();
    setUser(userData);
  };

  const value: AuthContextValue = {
    user,
    isAuthenticated: Boolean(user),
    loading,
    login,
    register,
    logout,
    refreshUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
