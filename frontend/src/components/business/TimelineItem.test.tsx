import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TimelineItem } from './TimelineItem';
import type { TimelineItem as TimelineItemData } from '@/types/timeline.types';

const base: TimelineItemData = {
  session_id: 's1',
  identity_id: 'i1',
  transcript: '这是一段转写文本',
  duration: 90,
  status: 'completed',
  recorded_at: '2024-06-05T09:30:00Z',
};

describe('TimelineItem', () => {
  it('渲染转写文本、时长与状态', () => {
    render(<TimelineItem item={base} identityName="产品经理" />);
    expect(screen.getByText('这是一段转写文本')).toBeInTheDocument();
    expect(screen.getByText(/时长 1m 30s/)).toBeInTheDocument();
    expect(screen.getByText('已完成')).toBeInTheDocument();
    expect(screen.getByText('产品经理')).toBeInTheDocument();
  });

  it('无身份 id 时不渲染身份徽标', () => {
    render(<TimelineItem item={{ ...base, identity_id: undefined }} />);
    expect(screen.queryByText('未知身份')).not.toBeInTheDocument();
  });

  it('无转写文本时展示占位文案', () => {
    render(<TimelineItem item={{ ...base, transcript: '' }} />);
    expect(screen.getByText('（无转写文本）')).toBeInTheDocument();
  });

  it('长文本默认折叠，可展开/收起', async () => {
    const long = '很长的转写内容。'.repeat(40); // > 200 字符
    render(<TimelineItem item={{ ...base, transcript: long }} />);
    const toggle = screen.getByRole('button', { name: '展开' });
    expect(toggle).toBeInTheDocument();

    await userEvent.click(toggle);
    expect(screen.getByRole('button', { name: '收起' })).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '收起' }));
    expect(screen.getByRole('button', { name: '展开' })).toBeInTheDocument();
  });
});
