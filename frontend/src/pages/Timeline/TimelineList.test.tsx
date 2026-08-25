import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TimelineList from './TimelineList';
import type { Identity } from '@/types/identity.types';
import type { TimelineItem as TimelineItemData } from '@/types/timeline.types';

const mocks = vi.hoisted(() => ({
  listIdentities: vi.fn(),
  listTimeline: vi.fn(),
}));

vi.mock('@/services/identity.service', () => ({
  identityService: { list: mocks.listIdentities },
}));

vi.mock('@/services/timeline.service', () => ({
  timelineService: { list: mocks.listTimeline },
}));

const identity: Identity = {
  id: 'i1',
  user_id: 'u1',
  name: '产品经理',
  color: '#3b82f6',
  icon: '🙂',
  is_default: false,
  created_at: '',
  updated_at: '',
};

const item = (overrides: Partial<TimelineItemData> = {}): TimelineItemData => ({
  session_id: 's1',
  identity_id: 'i1',
  transcript: '今天的会议纪要',
  duration: 90,
  status: 'completed',
  recorded_at: '2024-06-05T09:30:00Z',
  ...overrides,
});

describe('TimelineList 页面', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listIdentities.mockResolvedValue([identity]);
  });

  it('加载并渲染时间线列表', async () => {
    mocks.listTimeline.mockResolvedValue({ items: [item()], total: 1, page: 1, size: 20 });
    render(<TimelineList />);

    expect(await screen.findByText('今天的会议纪要')).toBeInTheDocument();
    const itemEl = screen.getByText('今天的会议纪要').closest('.timeline-item') as HTMLElement;
    expect(within(itemEl).getByText('已完成')).toBeInTheDocument();
    expect(within(itemEl).getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByText(/共 1 条/)).toBeInTheDocument();
  });

  it('分页：多页时可翻页', async () => {
    mocks.listTimeline.mockResolvedValue({ items: [item()], total: 25, page: 1, size: 20 });
    render(<TimelineList />);

    expect(await screen.findByText(/共 25 条/)).toBeInTheDocument();
    const next = screen.getByRole('button', { name: '下一页' });
    expect(next).toBeEnabled();
    await userEvent.click(next);
    await waitFor(() =>
      expect(mocks.listTimeline).toHaveBeenLastCalledWith(
        expect.objectContaining({ page: 2 }),
      ),
    );
  });

  it('按状态过滤会重置页码并携带查询参数', async () => {
    mocks.listTimeline.mockResolvedValue({ items: [], total: 0, page: 1, size: 20 });
    render(<TimelineList />);
    await screen.findByText('暂无时间线记录');

    await userEvent.selectOptions(screen.getByLabelText('状态'), 'completed');
    await waitFor(() =>
      expect(mocks.listTimeline).toHaveBeenLastCalledWith(
        expect.objectContaining({ status: 'completed', page: 1 }),
      ),
    );
  });

  it('按身份过滤携带 identity_id', async () => {
    mocks.listTimeline.mockResolvedValue({ items: [], total: 0, page: 1, size: 20 });
    render(<TimelineList />);
    await screen.findByText('暂无时间线记录');

    await userEvent.selectOptions(screen.getByLabelText('身份'), 'i1');
    await waitFor(() =>
      expect(mocks.listTimeline).toHaveBeenLastCalledWith(
        expect.objectContaining({ identity_id: 'i1' }),
      ),
    );
  });

  it('空态与加载态正常', async () => {
    mocks.listTimeline.mockResolvedValue({ items: [], total: 0, page: 1, size: 20 });
    render(<TimelineList />);
    expect(await screen.findByText('暂无时间线记录')).toBeInTheDocument();
  });

  it('加载失败展示错误态并可重试', async () => {
    mocks.listTimeline.mockRejectedValueOnce(new Error('网络错误'));
    mocks.listTimeline.mockResolvedValueOnce({ items: [item()], total: 1, page: 1, size: 20 });
    render(<TimelineList />);

    expect(await screen.findByText('网络错误')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    expect(await screen.findByText('今天的会议纪要')).toBeInTheDocument();
  });
});
