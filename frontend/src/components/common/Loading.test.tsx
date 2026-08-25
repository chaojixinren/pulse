import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FullScreenLoading, Loading } from './Loading';

describe('Loading', () => {
  it('渲染默认文案', () => {
    render(<Loading />);
    expect(screen.getByText('加载中…')).toBeInTheDocument();
  });

  it('渲染自定义文案', () => {
    render(<Loading text="处理中…" />);
    expect(screen.getByText('处理中…')).toBeInTheDocument();
  });
});

describe('FullScreenLoading', () => {
  it('渲染全屏加载容器', () => {
    const { container } = render(<FullScreenLoading />);
    expect(container.querySelector('.loading-fullscreen')).not.toBeNull();
  });
});
