import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { IdentityPie } from './IdentityPie';
import type { IdentityStat } from '@/types/report.types';

const data: IdentityStat[] = [
  { identity_id: 'i1', name: '产品经理', session_count: 3, total_duration: 180 },
  { identity_id: 'i2', name: '健身教练', session_count: 1, total_duration: 60 },
];

describe('IdentityPie', () => {
  it('渲染图例与占比', () => {
    render(<IdentityPie data={data} />);
    expect(screen.getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByText('健身教练')).toBeInTheDocument();
    expect(screen.getByText('3（75%）')).toBeInTheDocument();
    expect(screen.getByText('1（25%）')).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /身份占比图/ })).toBeInTheDocument();
  });

  it('空数据展示空态', () => {
    render(<IdentityPie data={[]} />);
    expect(screen.getByText('暂无身份数据')).toBeInTheDocument();
  });

  it('未命名身份显示「未分配」', () => {
    render(
      <IdentityPie data={[{ identity_id: '', name: '', session_count: 2, total_duration: 0 }]} />,
    );
    expect(screen.getByText('未分配')).toBeInTheDocument();
  });
});
