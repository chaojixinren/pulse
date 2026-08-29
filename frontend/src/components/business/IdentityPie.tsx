import { Empty } from '@/components/common/Empty';
import type { IdentityStat } from '@/types/report.types';

export interface IdentityPieProps {
  data: IdentityStat[];
  metric?: 'session_count' | 'total_duration';
  emptyTitle?: string;
}

const PALETTE = [
  '#3b82f6',
  '#10b981',
  '#f59e0b',
  '#ef4444',
  '#8b5cf6',
  '#ec4899',
  '#06b6d4',
  '#6b7280',
];

const SIZE = 200;
const STROKE = 40;
const C = SIZE / 2;
const R = (SIZE - STROKE) / 2;
const CIRCUMFERENCE = 2 * Math.PI * R;

// 自建 SVG 环形图：展示身份占比（会话数或时长），并配文字图例。
export function IdentityPie({
  data,
  metric = 'session_count',
  emptyTitle = '暂无身份数据',
}: IdentityPieProps) {
  if (data.length === 0) {
    return <Empty title={emptyTitle} />;
  }

  const total = data.reduce((sum, d) => sum + d[metric], 0);
  const metricLabel = metric === 'session_count' ? '次' : '时长';

  let offset = 0;
  const segments = data.map((d, i) => {
    const value = d[metric];
    const fraction = total > 0 ? value / total : 0;
    const length = fraction * CIRCUMFERENCE;
    const seg = {
      ...d,
      value,
      fraction,
      length,
      color: PALETTE[i % PALETTE.length],
      dashOffset: offset,
    };
    offset += length;
    return seg;
  });

  return (
    <div className="identity-pie">
      <svg
        className="identity-pie-svg"
        viewBox={`0 0 ${SIZE} ${SIZE}`}
        role="img"
        aria-label={`身份占比图（按${metricLabel}）`}
        preserveAspectRatio="xMidYMid meet"
      >
        <circle
          cx={C}
          cy={C}
          r={R}
          fill="none"
          stroke="var(--color-background-tertiary)"
          strokeWidth={STROKE}
        />
        {segments.map((s) => (
          <circle
            key={s.identity_id || s.name || s.color}
            cx={C}
            cy={C}
            r={R}
            fill="none"
            stroke={s.color}
            strokeWidth={STROKE}
            strokeDasharray={`${s.length} ${CIRCUMFERENCE - s.length}`}
            strokeDashoffset={-s.dashOffset}
            transform={`rotate(-90 ${C} ${C})`}
          >
            <title>{`${s.name || '未分配'}：${s.value}`}</title>
          </circle>
        ))}
      </svg>
      <ul className="identity-pie-legend">
        {segments.map((s) => (
          <li key={s.identity_id || s.name || s.color} className="identity-pie-legend-item">
            <span className="identity-pie-swatch" style={{ backgroundColor: s.color }} />
            <span className="identity-pie-name">{s.name || '未分配'}</span>
            <span className="identity-pie-value">
              {s.value}（{total > 0 ? Math.round(s.fraction * 100) : 0}%）
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
