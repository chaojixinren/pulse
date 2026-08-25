import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TrendChart } from './TrendChart';
import type { DailyPoint } from '@/types/report.types';

const points: DailyPoint[] = [
  { date: '2024-06-03', session_count: 1, total_duration: 60 },
  { date: '2024-06-04', session_count: 3, total_duration: 180 },
];

describe('TrendChart', () => {
  it('渲染柱状图与无障碍标签', () => {
    render(<TrendChart points={points} metric="session_count" />);
    expect(screen.getByRole('img', { name: /每日会话数趋势图/ })).toBeInTheDocument();
    expect(document.querySelectorAll('.trend-bar')).toHaveLength(2);
  });

  it('时长指标使用对应标签与格式化', () => {
    render(<TrendChart points={points} metric="total_duration" formatValue={(v) => v + 's'} />);
    expect(screen.getByRole('img', { name: /每日时长趋势图/ })).toBeInTheDocument();
  });

  it('空数据展示空态', () => {
    render(<TrendChart points={[]} metric="session_count" />);
    expect(screen.getByText('暂无趋势数据')).toBeInTheDocument();
  });
});
