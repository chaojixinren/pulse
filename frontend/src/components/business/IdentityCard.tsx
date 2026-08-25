import { Button } from '@/components/common/Button';
import type { Identity } from '@/types/identity.types';

export interface IdentityCardProps {
  identity: Identity;
  onEdit?: (identity: Identity) => void;
  onDelete?: (identity: Identity) => void;
  onSetDefault?: (identity: Identity) => void;
  busy?: boolean;
}

export function IdentityCard({
  identity,
  onEdit,
  onDelete,
  onSetDefault,
  busy = false,
}: IdentityCardProps) {
  return (
    <div className="identity-card">
      <div className="identity-card-head">
        <div
          className="identity-avatar"
          style={{ backgroundColor: identity.color || '#3b82f6' }}
        >
          {identity.icon || '🙂'}
        </div>
        <div style={{ minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <span className="identity-name">{identity.name}</span>
            {identity.is_default && <span className="badge badge-default">默认</span>}
          </div>
          {identity.description && <div className="identity-desc">{identity.description}</div>}
        </div>
      </div>
      <div className="identity-card-actions">
        {!identity.is_default && (
          <Button
            size="small"
            variant="ghost"
            disabled={busy}
            onClick={() => onSetDefault?.(identity)}
          >
            设为默认
          </Button>
        )}
        <Button
          size="small"
          variant="secondary"
          disabled={busy}
          onClick={() => onEdit?.(identity)}
        >
          编辑
        </Button>
        <Button
          size="small"
          variant="danger"
          disabled={busy || identity.is_default}
          title={identity.is_default ? '默认身份不可删除' : undefined}
          onClick={() => onDelete?.(identity)}
        >
          删除
        </Button>
      </div>
    </div>
  );
}
