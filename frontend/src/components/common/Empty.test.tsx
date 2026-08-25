import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Empty } from './Empty';

describe('Empty', () => {
  it('渲染默认内容', () => {
    render(<Empty />);
    expect(screen.getByText('暂无数据')).toBeInTheDocument();
  });

  it('渲染自定义图标/标题/描述/操作', () => {
    render(
      <Empty icon="🎭" title="还没有身份" description="创建第一个身份" action={<button>创建</button>} />,
    );
    expect(screen.getByText('🎭')).toBeInTheDocument();
    expect(screen.getByText('还没有身份')).toBeInTheDocument();
    expect(screen.getByText('创建第一个身份')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建' })).toBeInTheDocument();
  });

  it('不渲染描述与操作（未传入时）', () => {
    render(<Empty icon="📭" />);
    expect(screen.queryByText('暂无数据')).toBeInTheDocument();
  });
});
