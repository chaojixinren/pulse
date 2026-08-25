import { describe, expect, it } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useIdentityMap } from './useIdentityMap';
import type { Identity } from '@/types/identity.types';

const identities: Identity[] = [
  {
    id: 'i1',
    user_id: 'u1',
    name: '产品经理',
    color: '#3b82f6',
    icon: '🙂',
    is_default: true,
    created_at: '',
    updated_at: '',
  },
  {
    id: 'i2',
    user_id: 'u1',
    name: '健身教练',
    color: '#10b981',
    icon: '🏋️',
    is_default: false,
    created_at: '',
    updated_at: '',
  },
];

describe('useIdentityMap', () => {
  it('将身份列表转为 id → Identity 映射', () => {
    const { result } = renderHook(() => useIdentityMap(identities));
    expect(result.current.get('i1')?.name).toBe('产品经理');
    expect(result.current.get('i2')?.icon).toBe('🏋️');
    expect(result.current.size).toBe(2);
  });

  it('undefined 输入返回空映射', () => {
    const { result } = renderHook(() => useIdentityMap(undefined));
    expect(result.current.size).toBe(0);
  });
});
