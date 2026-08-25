import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input } from './Input';

describe('Input', () => {
  it('渲染 label 并关联 input', () => {
    render(<Input label="邮箱" placeholder="you@example.com" />);
    const input = screen.getByPlaceholderText('you@example.com');
    expect(screen.getByText('邮箱')).toBeInTheDocument();
    expect(input).toBeInTheDocument();
  });

  it('输入值双向绑定', async () => {
    render(<Input label="邮箱" />);
    const input = screen.getByLabelText('邮箱');
    await userEvent.type(input, 'a@b.com');
    expect(input).toHaveValue('a@b.com');
  });

  it('错误态渲染错误文案并设置 aria-invalid', () => {
    render(<Input label="密码" error="请输入密码" />);
    expect(screen.getByText('请输入密码')).toBeInTheDocument();
    expect(screen.getByLabelText('密码')).toHaveAttribute('aria-invalid', 'true');
  });

  it('无错误时渲染辅助文案', () => {
    render(<Input label="邮箱" helperText="用于登录" />);
    expect(screen.getByText('用于登录')).toBeInTheDocument();
  });
});
