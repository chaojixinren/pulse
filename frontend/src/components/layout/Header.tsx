import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useApp } from '@/contexts/AppContext';
import { useAuth } from '@/contexts/AuthContext';

export function Header() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme, sidebarCollapsed, toggleSidebar } = useApp();
  const [loggingOut, setLoggingOut] = useState(false);
  const navigate = useNavigate();

  const handleLogout = async () => {
    setLoggingOut(true);
    try {
      await logout();
      navigate('/auth/login', { replace: true });
    } finally {
      setLoggingOut(false);
    }
  };

  return (
    <header className="header">
      <div className="header-left">
        <button
          type="button"
          className="btn btn-ghost btn-small"
          onClick={toggleSidebar}
          aria-label="切换侧边栏"
          title={sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'}
        >
          ☰
        </button>
        <span className="header-logo" onClick={() => navigate('/')} role="button" tabIndex={0}>
          <span className="header-logo-dot" />
          Pulse
        </span>
      </div>
      <div className="header-right">
        <button
          type="button"
          className="btn btn-ghost btn-small"
          onClick={toggleTheme}
          title="切换主题"
        >
          {theme === 'light' ? '🌙' : '☀️'}
        </button>
        {user && <span className="header-user">{user.name}</span>}
        <button
          type="button"
          className="btn btn-secondary btn-small"
          onClick={handleLogout}
          disabled={loggingOut}
        >
          {loggingOut ? '退出中…' : '退出登录'}
        </button>
      </div>
    </header>
  );
}
