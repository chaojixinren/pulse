import { useState } from 'react';
import { Button } from '@/components/common/Button';
import type { TimelineItem as TimelineItemData, TimelineStatus } from '@/types/timeline.types';
import { formatDateTime } from '@/utils/date';
import { formatDurationShort } from '@/utils/format';

const STATUS_LABELS: Record<TimelineStatus, string> = {
  pending: '待处理',
  processing: '处理中',
  completed: '已完成',
  failed: '失败',
};

const COLLAPSE_THRESHOLD = 200;

export interface TimelineItemProps {
  item: TimelineItemData;
  identityName?: string;
  identityColor?: string;
}

export function TimelineItem({ item, identityName, identityColor }: TimelineItemProps) {
  const [expanded, setExpanded] = useState(false);
  const shouldCollapse = item.transcript.length > COLLAPSE_THRESHOLD;

  return (
    <div className="timeline-item">
      <div className="timeline-item-head">
        {item.identity_id && (
          <span className="identity-badge">
            <span
              className="identity-badge-dot"
              style={{ backgroundColor: identityColor || '#9ca3af' }}
            />
            {identityName || '未知身份'}
          </span>
        )}
        <span className={'badge badge-status-' + item.status}>
          {STATUS_LABELS[item.status] || item.status}
        </span>
      </div>
      <div className="timeline-meta">
        <span>{formatDateTime(item.recorded_at)}</span>
        <span>时长 {formatDurationShort(item.duration)}</span>
      </div>
      {item.transcript ? (
        <>
          <p
            className={
              'timeline-transcript' + (shouldCollapse && !expanded ? ' collapsed' : '')
            }
          >
            {item.transcript}
          </p>
          {shouldCollapse && (
            <div className="timeline-toggle">
              <Button size="small" variant="ghost" onClick={() => setExpanded((v) => !v)}>
                {expanded ? '收起' : '展开'}
              </Button>
            </div>
          )}
        </>
      ) : (
        <p className="timeline-transcript" style={{ color: 'var(--color-text-secondary)' }}>
          {item.status === 'completed' ? '（无转写文本）' : '（转写尚未生成）'}
        </p>
      )}
    </div>
  );
}
