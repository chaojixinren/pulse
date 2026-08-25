import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ErrorBoundary } from './ErrorBoundary';

let throwNext = true;

function Flaky() {
  if (throwNext) throw new Error('组件崩溃');
  return <div>恢复成功</div>;
}

describe('ErrorBoundary', () => {
  it('正常渲染子组件', () => {
    throwNext = false;
    render(
      <ErrorBoundary>
        <Flaky />
      </ErrorBoundary>,
    );
    expect(screen.getByText('恢复成功')).toBeInTheDocument();
  });

  it('子组件抛错时展示错误态而非白屏', () => {
    throwNext = true;
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(
      <ErrorBoundary>
        <Flaky />
      </ErrorBoundary>,
    );
    expect(screen.getByText('页面出错了')).toBeInTheDocument();
    expect(screen.getByText('组件崩溃')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument();
    spy.mockRestore();
  });

  it('点击重试可恢复', async () => {
    throwNext = true;
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(
      <ErrorBoundary>
        <Flaky />
      </ErrorBoundary>,
    );
    expect(screen.getByText('页面出错了')).toBeInTheDocument();

    throwNext = false;
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    expect(screen.getByText('恢复成功')).toBeInTheDocument();
    spy.mockRestore();
  });
});
