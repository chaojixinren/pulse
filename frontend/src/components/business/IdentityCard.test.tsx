import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IdentityCard } from './IdentityCard';
import type { Identity } from '@/types/identity.types';

const base: Identity = {
  id: 'i1',
  user_id: 'u1',
  name: '产品经理',
  description: '负责产品规划',
  color: '#3b82f6',
  icon: '🙂',
  is_default: false,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('IdentityCard', () => {
  it('渲染名称、描述与图标', () => {
    render(<IdentityCard identity={base} />);
    expect(screen.getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByText('负责产品规划')).toBeInTheDocument();
    expect(screen.getByText('🙂')).toBeInTheDocument();
  });

  it('默认身份显示「默认」徽标且不显示设为默认按钮', () => {
    render(<IdentityCard identity={{ ...base, is_default: true }} />);
    expect(screen.getByText('默认')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '设为默认' })).not.toBeInTheDocument();
  });

  it('非默认身份显示设为默认按钮并触发回调', async () => {
    const onSetDefault = vi.fn();
    render(<IdentityCard identity={base} onSetDefault={onSetDefault} />);
    await userEvent.click(screen.getByRole('button', { name: '设为默认' }));
    expect(onSetDefault).toHaveBeenCalledWith(base);
  });

  it('触发编辑与删除回调', async () => {
    const onEdit = vi.fn();
    const onDelete = vi.fn();
    render(<IdentityCard identity={base} onEdit={onEdit} onDelete={onDelete} />);
    await userEvent.click(screen.getByRole('button', { name: '编辑' }));
    await userEvent.click(screen.getByRole('button', { name: '删除' }));
    expect(onEdit).toHaveBeenCalledWith(base);
    expect(onDelete).toHaveBeenCalledWith(base);
  });

  it('默认身份删除按钮禁用', () => {
    render(<IdentityCard identity={{ ...base, is_default: true }} />);
    expect(screen.getByRole('button', { name: '删除' })).toBeDisabled();
  });
});
