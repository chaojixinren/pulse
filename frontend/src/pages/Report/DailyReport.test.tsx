import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DailyReport from './DailyReport';
import { shiftDate, todayStr } from '@/utils/date';
import type { DailyReport as DailyReportData } from '@/types/report.types';

const mocks = vi.hoisted(() => ({
  daily: vi.fn(),
}));

vi.mock('@/services/report.service', () => ({
  reportService: { daily: mocks.daily },
}));

const report: DailyReportData = {
  date: '2024-06-05',
  session_count: 2,
  total_duration: 3660,
  by_identity: [{ identity_id: 'i1', name: '产品经理', session_count: 2, total_duration: 3660 }],
  todos: ['整理需求文档'],
  notes: ['讨论了产品方案'],
};

describe('DailyReport 页面', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('渲染汇总、按身份拆分、待办与笔记', async () => {
    mocks.daily.mockResolvedValue(report);
    render(<DailyReport />);

    expect(await screen.findByText('2')).toBeInTheDocument(); // 会话数
    expect(screen.getByText('1 小时 1 分钟')).toBeInTheDocument(); // 总时长
    expect(screen.getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByText('2 次 · 1 小时 1 分钟')).toBeInTheDocument();
    expect(screen.getByText('整理需求文档')).toBeInTheDocument();
    expect(screen.getByText('讨论了产品方案')).toBeInTheDocument();
  });

  it('默认加载今天', async () => {
    mocks.daily.mockResolvedValue(report);
    render(<DailyReport />);
    await screen.findByText('2');
    expect(mocks.daily).toHaveBeenCalledWith(todayStr());
  });

  it('无数据日期展示空态', async () => {
    mocks.daily.mockResolvedValue({
      date: '2024-06-05',
      session_count: 0,
      total_duration: 0,
      by_identity: [],
      todos: [],
      notes: [],
    });
    render(<DailyReport />);
    expect(await screen.findByText('该日期暂无数据')).toBeInTheDocument();
  });

  it('切换日期加载对应日报', async () => {
    mocks.daily.mockResolvedValue(report);
    render(<DailyReport />);
    await screen.findByText('2');

    await userEvent.click(screen.getByRole('button', { name: '← 前一天' }));
    await waitFor(() => expect(mocks.daily).toHaveBeenLastCalledWith(shiftDate(todayStr(), -1)));

    await userEvent.click(screen.getByRole('button', { name: '后一天 →' }));
    await waitFor(() => expect(mocks.daily).toHaveBeenLastCalledWith(todayStr()));
  });

  it('加载失败展示错误态并可重试', async () => {
    mocks.daily.mockRejectedValueOnce(new Error('网络错误'));
    mocks.daily.mockResolvedValueOnce(report);
    render(<DailyReport />);

    expect(await screen.findByText('网络错误')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    expect(await screen.findByText('2')).toBeInTheDocument();
  });
});
