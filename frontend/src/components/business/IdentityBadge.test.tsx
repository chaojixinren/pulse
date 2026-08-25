import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { IdentityBadge } from './IdentityBadge';

describe('IdentityBadge', () => {
  it('渲染颜色、图标与名称', () => {
    render(<IdentityBadge identity={{ name: '产品经理', color: '#3b82f6', icon: '🙂' }} />);
    expect(screen.getByText('产品经理')).toBeInTheDocument();
    expect(screen.getByText('🙂')).toBeInTheDocument();
    const dot = document.querySelector('.identity-badge-dot') as HTMLElement;
    expect(dot.style.backgroundColor).toBe('rgb(59, 130, 246)');
  });

  it('无身份时展示「未识别」', () => {
    render(<IdentityBadge identity={undefined} />);
    expect(screen.getByText('未识别')).toBeInTheDocument();
    expect(document.querySelector('.identity-badge-unrecognized')).toBeInTheDocument();
  });

  it('无图标时仍展示名称', () => {
    render(<IdentityBadge identity={{ name: '无图标', color: '#10b981' }} />);
    expect(screen.getByText('无图标')).toBeInTheDocument();
  });
});
