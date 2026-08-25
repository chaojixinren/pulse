import { describe, expect, it } from 'vitest';
import { formatCountdown, formatDuration, formatDurationShort } from './format';

describe('formatDuration', () => {
  it('纯秒数', () => {
    expect(formatDuration(30)).toBe('30 秒');
    expect(formatDuration(0)).toBe('0 秒');
  });

  it('分与秒', () => {
    expect(formatDuration(90)).toBe('1 分 30 秒');
  });

  it('小时与分钟', () => {
    expect(formatDuration(3660)).toBe('1 小时 1 分钟');
  });

  it('负数被截断为 0', () => {
    expect(formatDuration(-5)).toBe('0 秒');
  });
});

describe('formatDurationShort', () => {
  it('纯秒数', () => {
    expect(formatDurationShort(45)).toBe('45s');
  });

  it('分与秒', () => {
    expect(formatDurationShort(90)).toBe('1m 30s');
  });

  it('小时与分钟', () => {
    expect(formatDurationShort(3660)).toBe('1h 1m');
  });

  it('负数被截断为 0', () => {
    expect(formatDurationShort(-1)).toBe('0s');
  });
});

describe('formatCountdown', () => {
  it('格式化为 MM:SS', () => {
    expect(formatCountdown(0)).toBe('00:00');
    expect(formatCountdown(65)).toBe('01:05');
    expect(formatCountdown(600)).toBe('10:00');
  });

  it('负数被截断为 0', () => {
    expect(formatCountdown(-5)).toBe('00:00');
  });
});
