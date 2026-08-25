import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ExtractedList } from './ExtractedList';

describe('ExtractedList', () => {
  it('渲染条目列表', () => {
    render(<ExtractedList items={['整理需求文档', '评审方案']} />);
    expect(screen.getByText('整理需求文档')).toBeInTheDocument();
    expect(screen.getByText('评审方案')).toBeInTheDocument();
  });

  it('空列表展示空态', () => {
    render(<ExtractedList items={[]} emptyIcon="✅" emptyTitle="暂无待办" />);
    expect(screen.getByText('暂无待办')).toBeInTheDocument();
  });

  it('可勾选模式下点击切换完成状态', async () => {
    render(<ExtractedList items={['整理需求文档']} checkable />);
    const checkbox = screen.getByRole('checkbox', { name: '整理需求文档' });
    expect(checkbox).not.toBeChecked();

    await userEvent.click(checkbox);
    expect(checkbox).toBeChecked();

    await userEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();
  });

  it('非勾选模式不渲染复选框', () => {
    render(<ExtractedList items={['笔记内容']} />);
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
  });
});
