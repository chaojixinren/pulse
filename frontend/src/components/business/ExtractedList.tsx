import { useState } from 'react';
import { Empty } from '@/components/common/Empty';

export interface ExtractedListProps {
  items: string[];
  emptyTitle?: string;
  // 待办列表可勾选（本地完成状态，Phase 3 可选持久化）。
  checkable?: boolean;
}

// AI 提取结果（待办/承诺/笔记）的结构化列表展示。
export function ExtractedList({
  items,
  emptyTitle = '暂无内容',
  checkable = false,
}: ExtractedListProps) {
  const [checked, setChecked] = useState<Set<number>>(() => new Set());

  if (items.length === 0) {
    return <Empty title={emptyTitle} />;
  }

  const toggle = (idx: number) => {
    setChecked((prev) => {
      const next = new Set(prev);
      if (next.has(idx)) {
        next.delete(idx);
      } else {
        next.add(idx);
      }
      return next;
    });
  };

  return (
    <ul className="extracted-list">
      {items.map((item, idx) => {
        if (!checkable) {
          return (
            <li key={idx} className="extracted-item">
              {item}
            </li>
          );
        }

        const done = checked.has(idx);
        return (
          <li key={idx} className="extracted-item">
            <label className="extracted-checkable">
              <input
                type="checkbox"
                checked={done}
                onChange={() => toggle(idx)}
                aria-label={item}
              />
              <span className={'extracted-text' + (done ? ' extracted-text-done' : '')}>
                {item}
              </span>
            </label>
          </li>
        );
      })}
    </ul>
  );
}
