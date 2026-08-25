import type { ReactNode } from 'react';

export interface EmptyProps {
  icon?: string;
  title?: string;
  description?: string;
  action?: ReactNode;
}

export function Empty({ icon = '📭', title = '暂无数据', description, action }: EmptyProps) {
  return (
    <div className="empty">
      <div className="empty-icon">{icon}</div>
      <div className="empty-title">{title}</div>
      {description && <div className="empty-description">{description}</div>}
      {action}
    </div>
  );
}
