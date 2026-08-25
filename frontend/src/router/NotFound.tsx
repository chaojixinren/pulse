import { Link } from 'react-router-dom';

export function NotFound() {
  return (
    <div className="empty" style={{ minHeight: '60vh' }}>
      <div className="empty-icon">🧭</div>
      <div className="empty-title">页面不存在</div>
      <p className="empty-description">你访问的地址可能已被移动或删除。</p>
      <Link to="/" className="btn btn-primary">
        返回首页
      </Link>
    </div>
  );
}
