import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WeeklyReport from './WeeklyReport';
import { mondayOf, shiftDate, todayStr } from '@/utils/date';
import type { WeeklyReport as WeeklyReportData } from '@/types/report.types';

const mocks = vi.hoisted(() => ({ weekly: vi.fn() }));

vi.mock('@/services/report.service', () => ({
  reportService: { weekly: mocks.weekly },
}));

const report: WeeklyReportData = {
  week: '2024-06-03',
  session_count: 2,
  total_duration: 3660,
  by_identity: [{ identity_id: 'i1', name: '产品经理', session_count: 2, total_duration: 3660 }],
  top_todos: ['整理需求文档'],
  commitments_done: 1,
  daily_trend: [
    { date: '2024-06-03', session_count: 1, total_duration: 1830 },
    { date: '2024-06-04', session_count: 1, total_duration: 1830 },
  ],
};

describe('WeeklyReport 页面', () => {
  beforeEach(() => vi.clearAllMocks());

  it('渲染汇总、趋势、占比与 Top 待办', async () => {
    mocks.weekly.mockResolvedValue(report);
    render(<WeeklyReport />);

    expect(await screen.findByText('整理需求文档')).toBeInTheDocument();
    expect(screen.getByText('会话数')).toBeInTheDocument();
    expect(screen.getByText('总时长')).toBeInTheDocument();
    expect(screen.getByText('完成承诺')).toBeInTheDocument();
    expect(screen.getByText('1 小时 1 分钟')).toBeInTheDocument();
    expect(screen.getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /每日会话数趋势图/ })).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /身份占比图/ })).toBeInTheDocument();
  });

  it('默认加载本周（周一）', async () => {
    mocks.weekly.mockResolvedValue(report);
    render(<WeeklyReport />);
    await screen.findByText('整理需求文档');
    expect(mocks.weekly).toHaveBeenCalledWith(mondayOf(todayStr()));
  });

  it('切换周加载对应周报', async () => {
    mocks.weekly.mockResolvedValue(report);
    render(<WeeklyReport />);
    await screen.findByText('整理需求文档');

    await userEvent.click(screen.getByRole('button', { name: '← 上一周' }));
    await waitFor(() =>
      expect(mocks.weekly).toHaveBeenLastCalledWith(shiftDate(mondayOf(todayStr()), -7)),
    );

    await userEvent.click(screen.getByRole('button', { name: '下一周 →' }));
    await waitFor(() => expect(mocks.weekly).toHaveBeenLastCalledWith(mondayOf(todayStr())));
  });

  it('无数据展示空态', async () => {
    mocks.weekly.mockResolvedValue({
      ...report,
      session_count: 0,
      daily_trend: [],
      by_identity: [],
      top_todos: [],
      commitments_done: 0,
    });
    render(<WeeklyReport />);
    expect(await screen.findByText('本周暂无数据')).toBeInTheDocument();
  });

  it('加载失败展示错误态并可重试', async () => {
    mocks.weekly.mockRejectedValueOnce(new Error('网络错误'));
    mocks.weekly.mockResolvedValueOnce(report);
    render(<WeeklyReport />);

    expect(await screen.findByText('网络错误')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    expect(await screen.findByText('整理需求文档')).toBeInTheDocument();
  });
});
