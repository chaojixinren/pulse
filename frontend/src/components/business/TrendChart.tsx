import { Empty } from '@/components/common/Empty';
import type { DailyPoint } from '@/types/report.types';
import { formatMonthDay } from '@/utils/date';

export interface TrendChartProps {
  points: DailyPoint[];
  metric: 'session_count' | 'total_duration';
  formatValue?: (value: number) => string;
  emptyTitle?: string;
}

const WIDTH = 720;
const HEIGHT = 220;
const PAD_X = 8;
const PAD_TOP = 28;
const PAD_BOTTOM = 28;

// 自建 SVG 柱状图：展示每日会话数/时长趋势，避免引入重依赖。
export function TrendChart({
  points,
  metric,
  formatValue,
  emptyTitle = '暂无趋势数据',
}: TrendChartProps) {
  if (points.length === 0) {
    return <Empty icon="📈" title={emptyTitle} />;
  }

  const values = points.map((p) => p[metric]);
  const max = Math.max(1, ...values);
  const chartWidth = WIDTH - PAD_X * 2;
  const chartHeight = HEIGHT - PAD_TOP - PAD_BOTTOM;
  const barGap = Math.min(8, chartWidth / points.length / 3);
  const barWidth = (chartWidth - barGap * (points.length - 1)) / points.length;
  const fmt = formatValue ?? ((v: number) => String(v));
  const labelStep = Math.max(1, Math.ceil(points.length / 10));
  const metricLabel = metric === 'session_count' ? '会话数' : '时长';

  return (
    <svg
      className="trend-chart"
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      role="img"
      aria-label={`每日${metricLabel}趋势图`}
      preserveAspectRatio="xMidYMid meet"
    >
      <line
        className="trend-axis"
        x1={PAD_X}
        y1={HEIGHT - PAD_BOTTOM}
        x2={WIDTH - PAD_X}
        y2={HEIGHT - PAD_BOTTOM}
      />
      {points.map((p, i) => {
        const value = p[metric];
        const barH = (value / max) * chartHeight;
        const x = PAD_X + i * (barWidth + barGap);
        const y = HEIGHT - PAD_BOTTOM - barH;
        const showLabel = i % labelStep === 0 || i === points.length - 1;
        return (
          <g key={p.date}>
            <rect className="trend-bar" x={x} y={y} width={barWidth} height={barH} rx={2}>
              <title>{`${p.date}：${fmt(value)}`}</title>
            </rect>
            {value > 0 && barWidth >= 14 && (
              <text className="trend-value" x={x + barWidth / 2} y={y - 4} textAnchor="middle">
                {fmt(value)}
              </text>
            )}
            {showLabel && (
              <text
                className="trend-label"
                x={x + barWidth / 2}
                y={HEIGHT - PAD_BOTTOM + 16}
                textAnchor="middle"
              >
                {formatMonthDay(p.date)}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}
