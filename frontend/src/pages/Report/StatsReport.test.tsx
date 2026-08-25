import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import StatsReport from './StatsReport';
import { shiftDate, todayStr } from '@/utils/date';
import type { StatsReport as StatsReportData } from '@/types/report.types';

const mocks = vi.hoisted(() => ({ stats: vi.fn() }));

vi.mock('@/services/report.service', () => ({
  reportService: { stats: mocks.stats },
}));

const report: StatsReportData = {
  from: '2024-05-01',
  to: '2024-05-31',
  session_count: 2,
  total_duration: 3660,
  by_identity: [{ identity_id: 'i1', name: '产品经理', session_count: 2, total_duration: 3660 }],
  daily_trend: [
    { date: '2024-05-01', session_count: 1, total_duration: 1830 },
    { date: '2024-05-02', session_count: 1, total_duration: 1830 },
  ],
};

describe('StatsReport 页面', () => {
  beforeEach(() => vi.clearAllMocks());

  it('渲染汇总、趋势与占比', async () => {
    mocks.stats.mockResolvedValue(report);
    render(<StatsReport />);

    expect(await screen.findByText('每日会话数趋势')).toBeInTheDocument();
    expect(screen.getByText('会话数')).toBeInTheDocument();
    expect(screen.getByText('1 小时 1 分钟')).toBeInTheDocument();
    expect(screen.getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /每日会话数趋势图/ })).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /每日时长趋势图/ })).toBeInTheDocument();
  });

  it('默认查询近 30 天', async () => {
    mocks.stats.mockResolvedValue(report);
    render(<StatsReport />);
    await screen.findByText('每日会话数趋势');
    expect(mocks.stats).toHaveBeenCalledWith(shiftDate(todayStr(), -29), todayStr());
  });

  it('修改区间后查询', async () => {
    mocks.stats.mockResolvedValue(report);
    render(<StatsReport />);
    await screen.findByText('每日会话数趋势');

    fireEvent.change(screen.getByLabelText('起始日期'), { target: { value: '2024-05-01' } });
    fireEvent.change(screen.getByLabelText('结束日期'), { target: { value: '2024-05-31' } });
    await userEvent.click(screen.getByRole('button', { name: '查询' }));

    await waitFor(() => expect(mocks.stats).toHaveBeenLastCalledWith('2024-05-01', '2024-05-31'));
  });

  it('起始晚于结束时提示且不查询', async () => {
    mocks.stats.mockResolvedValue(report);
    render(<StatsReport />);
    await screen.findByText('每日会话数趋势');

    fireEvent.change(screen.getByLabelText('起始日期'), { target: { value: '2024-06-01' } });
    fireEvent.change(screen.getByLabelText('结束日期'), { target: { value: '2024-05-01' } });
    await userEvent.click(screen.getByRole('button', { name: '查询' }));

    expect(screen.getByText('起始日期不能晚于结束日期')).toBeInTheDocument();
    expect(mocks.stats).toHaveBeenCalledTimes(1);
  });

  it('无数据展示空态', async () => {
    mocks.stats.mockResolvedValue({
      ...report,
      session_count: 0,
      daily_trend: [],
      by_identity: [],
    });
    render(<StatsReport />);
    expect(await screen.findByText('该区间暂无数据')).toBeInTheDocument();
  });
});
