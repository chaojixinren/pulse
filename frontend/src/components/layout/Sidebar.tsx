import { NavLink } from 'react-router-dom';
import { useApp } from '@/contexts/AppContext';

const navItems = [
  { to: '/identity', label: '身份管理' },
  { to: '/devices', label: '设备管理' },
  { to: '/timeline', label: '时间线' },
  { to: '/reports/daily', label: '日报' },
  { to: '/reports/weekly', label: '周报' },
  { to: '/reports/stats', label: '统计' },
  { to: '/account', label: '账户设置' },
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
            className={({ isActive }) => `sidebar-link${isActive ? ' active' : ''}`}
          >
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
