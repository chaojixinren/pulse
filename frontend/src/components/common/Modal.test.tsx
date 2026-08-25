import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { Modal } from './Modal';

describe('Modal', () => {
  it('关闭时不渲染', () => {
    render(
      <Modal open={false} onClose={() => {}}>
        内容
      </Modal>,
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('打开时渲染标题与内容（portal 到 body）', () => {
    render(
      <Modal open onClose={() => {}} title="创建身份">
        <p>表单内容</p>
      </Modal>,
    );
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('创建身份')).toBeInTheDocument();
    expect(screen.getByText('表单内容')).toBeInTheDocument();
  });

  it('点击关闭按钮触发 onClose', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} title="标题">
        内容
      </Modal>,
    );
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('点击遮罩层触发 onClose', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose}>
        内容
      </Modal>,
    );
    fireEvent.click(document.querySelector('.modal-overlay') as Element);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('按 Escape 触发 onClose', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose}>
        内容
      </Modal>,
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('closeOnEscape=false 时 Escape 不关闭', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} closeOnEscape={false}>
        内容
      </Modal>,
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
  });

  it('渲染 footer', () => {
    render(
      <Modal open onClose={() => {}} footer={<button>保存</button>}>
        内容
      </Modal>,
    );
    expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument();
  });
});
