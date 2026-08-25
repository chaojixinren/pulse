import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToastProvider, useToast } from './Toast';

function ToastHarness() {
  const toast = useToast();
  return (
    <div>
      <button onClick={() => toast.success('成功消息')}>success</button>
      <button onClick={() => toast.error('失败消息')}>error</button>
      <button onClick={() => toast.info('普通消息')}>info</button>
    </div>
  );
}

afterEach(() => {
  vi.useRealTimers();
});

describe('Toast', () => {
  it('show 后渲染对应类型与文案', async () => {
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>,
    );
    await userEvent.click(screen.getByRole('button', { name: 'success' }));
    const toast = screen.getByText('成功消息');
    expect(toast).toBeInTheDocument();
    expect(toast.closest('.toast')).toHaveClass('toast-success');
  });

  it('error 类型渲染 toast-error', async () => {
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>,
    );
    await userEvent.click(screen.getByRole('button', { name: 'error' }));
    expect(screen.getByText('失败消息').closest('.toast')).toHaveClass('toast-error');
  });

  it('3 秒后自动移除', () => {
    vi.useFakeTimers();
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'info' }));
    expect(screen.getByText('普通消息')).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(3000);
    });
    expect(screen.queryByText('普通消息')).not.toBeInTheDocument();
  });

  it('在 Provider 外使用 useToast 抛错', () => {
    // 屏蔽 React 的错误输出，仅验证抛错行为。
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    expect(() => render(<ToastHarness />)).toThrow('useToast must be used within ToastProvider');
    spy.mockRestore();
  });
});
