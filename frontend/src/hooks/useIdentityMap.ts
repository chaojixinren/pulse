import { useMemo } from 'react';
import type { Identity } from '@/types/identity.types';

// 将身份列表转为 id → Identity 映射，供时间线/报告快速取名称、颜色、图标。
export function useIdentityMap(identities: Identity[] | undefined): Map<string, Identity> {
  return useMemo(() => {
    const map = new Map<string, Identity>();
    identities?.forEach((it) => map.set(it.id, it));
    return map;
  }, [identities]);
}
