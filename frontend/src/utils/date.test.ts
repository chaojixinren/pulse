import { describe, expect, it } from 'vitest';
import { formatDate, formatDateTime, shiftDate, todayStr } from './date';

describe('formatDate', () => {
  it('将 ISO 时间转为本地时区的 YYYY-MM-DD', () => {
    const d = new Date(2024, 0, 15, 12, 0, 0);
    expect(formatDate(d.toISOString())).toBe('2024-01-15');
  });

  it('对非法输入原样返回', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date');
  });
});

describe('formatDateTime', () => {
  it('将 ISO 时间转为本地时区的日期时间', () => {
    const d = new Date(2024, 5, 5, 9, 30, 0);
    expect(formatDateTime(d.toISOString())).toBe('2024-06-05 09:30');
  });

  it('对非法输入原样返回', () => {
    expect(formatDateTime('invalid')).toBe('invalid');
  });
});

describe('todayStr', () => {
  it('返回今天的本地日期 YYYY-MM-DD', () => {
    const now = new Date();
    const pad = (n: number) => String(n).padStart(2, '0');
    const expected = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
    expect(todayStr()).toBe(expected);
  });
});

describe('shiftDate', () => {
  it('按天数向前/向后偏移（本地时区）', () => {
    expect(shiftDate('2024-01-31', 1)).toBe('2024-02-01');
    expect(shiftDate('2024-03-01', -1)).toBe('2024-02-29'); // 闰年
  });

  it('跨年正确进位', () => {
    expect(shiftDate('2024-12-31', 1)).toBe('2025-01-01');
    expect(shiftDate('2025-01-01', -1)).toBe('2024-12-31');
  });

  it('对非法输入原样返回', () => {
    expect(shiftDate('bad-date', 1)).toBe('bad-date');
  });
});
