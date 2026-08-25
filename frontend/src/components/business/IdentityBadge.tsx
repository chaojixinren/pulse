export interface IdentityBadgeProps {
  // 传入 undefined 表示 AI 未识别出身份。
  identity?: { name: string; color: string; icon?: string };
}

// 身份徽标：颜色 + 图标 + 名称；低置信度（无身份）时显示「未识别」灰标。
export function IdentityBadge({ identity }: IdentityBadgeProps) {
  if (!identity) {
    return <span className="identity-badge identity-badge-unrecognized">未识别</span>;
  }

  return (
    <span className="identity-badge">
      <span
        className="identity-badge-dot"
        style={{ backgroundColor: identity.color || '#9ca3af' }}
      />
      {identity.icon && <span className="identity-badge-icon">{identity.icon}</span>}
      <span className="identity-badge-name">{identity.name}</span>
    </span>
  );
}
