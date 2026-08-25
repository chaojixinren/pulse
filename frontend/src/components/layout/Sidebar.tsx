import { NavLink } from 'react-router-dom';
import { useApp } from '@/contexts/AppContext';

const navItems = [
  { to: '/identity', label: '身份管理', icon: '👤' },
  { to: '/devices', label: '设备管理', icon: '📡' },
  { to: '/timeline', label: '时间线', icon: '🕒' },
  { to: '/reports/daily', label: '日报', icon: '📊' },
];

export function Sidebar() {
  const { sidebarCollapsed } = useApp();

  if (sidebarCollapsed) return null;

  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              `sidebar-link${isActive ? ' active' : ''}`
            }
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
