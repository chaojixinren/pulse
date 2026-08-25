import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TimelineItem } from './TimelineItem';
import type { Identity } from '@/types/identity.types';
import type { TimelineItem as TimelineItemData } from '@/types/timeline.types';

const base: TimelineItemData = {
  session_id: 's1',
  identity_id: 'i1',
  transcript: '这是一段转写文本',
  duration: 90,
  status: 'completed',
  recorded_at: '2024-06-05T09:30:00Z',
};

const identity: Identity = {
  id: 'i1',
  user_id: 'u1',
  name: '产品经理',
  color: '#3b82f6',
  icon: '🙂',
  is_default: true,
  created_at: '',
  updated_at: '',
};

describe('TimelineItem', () => {
  it('渲染转写文本、时长与状态，并显示身份徽标', () => {
    render(<TimelineItem item={base} identity={identity} />);
    expect(screen.getByText('这是一段转写文本')).toBeInTheDocument();
    expect(screen.getByText(/时长 1m 30s/)).toBeInTheDocument();
    expect(screen.getByText('已完成')).toBeInTheDocument();
    expect(screen.getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByText('🙂')).toBeInTheDocument();
  });

  it('无身份时展示「未识别」', () => {
    render(<TimelineItem item={{ ...base, identity_id: undefined }} />);
    expect(screen.getByText('未识别')).toBeInTheDocument();
  });

  it('无转写文本时展示占位文案', () => {
    render(<TimelineItem item={{ ...base, transcript: '' }} identity={identity} />);
    expect(screen.getByText('（无转写文本）')).toBeInTheDocument();
  });

  it('长文本默认折叠，可展开/收起', async () => {
    const long = '很长的转写内容。'.repeat(40); // > 200 字符
    render(<TimelineItem item={{ ...base, transcript: long }} identity={identity} />);
    const toggle = screen.getByRole('button', { name: '展开' });
    expect(toggle).toBeInTheDocument();

    await userEvent.click(toggle);
    expect(screen.getByRole('button', { name: '收起' })).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '收起' }));
    expect(screen.getByRole('button', { name: '展开' })).toBeInTheDocument();
  });
});
