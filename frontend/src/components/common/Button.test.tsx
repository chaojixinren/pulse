import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';

describe('Button', () => {
  it('渲染子元素并响应点击', async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>点击</Button>);
    await userEvent.click(screen.getByRole('button', { name: '点击' }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('应用 variant 与 size 样式类', () => {
    render(
      <Button variant="danger" size="small">
        删除
      </Button>,
    );
    const btn = screen.getByRole('button', { name: '删除' });
    expect(btn.className).toContain('btn-danger');
    expect(btn.className).toContain('btn-small');
  });

  it('默认 type 为 button', () => {
    render(<Button>提交</Button>);
    expect(screen.getByRole('button', { name: '提交' })).toHaveAttribute('type', 'button');
  });

  it('loading 时禁用并渲染 spinner', () => {
    render(<Button loading>保存</Button>);
    const btn = screen.getByRole('button', { name: '保存' });
    expect(btn).toBeDisabled();
    expect(btn.querySelector('.btn-spinner')).not.toBeNull();
  });

  it('disabled 时不触发点击', async () => {
    const onClick = vi.fn();
    render(
      <Button disabled onClick={onClick}>
        禁止
      </Button>,
    );
    await userEvent.click(screen.getByRole('button', { name: '禁止' }));
    expect(onClick).not.toHaveBeenCalled();
  });
});
