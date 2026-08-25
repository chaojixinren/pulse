import { describe, expect, it } from 'vitest';
import { COMMAND_LABELS, DEVICE_COMMANDS, formatBattery, isDeviceOnline } from './device';
import type { Device } from '@/types/device.types';

const base: Device = {
  id: 'd1',
  user_id: 'u1',
  device_id: 'hw-001',
  name: '我的设备',
  device_type: 'wearable',
  is_active: true,
  created_at: '',
  updated_at: '',
};

const NOW = new Date('2024-06-05T10:00:00Z').getTime();

describe('isDeviceOnline', () => {
  it('激活且最近心跳在阈值内视为在线', () => {
    const device: Device = { ...base, last_seen_at: '2024-06-05T09:58:00Z' };
    expect(isDeviceOnline(device, NOW)).toBe(true);
  });

  it('超过阈值视为离线', () => {
    const device: Device = { ...base, last_seen_at: '2024-06-05T09:00:00Z' };
    expect(isDeviceOnline(device, NOW)).toBe(false);
  });

  it('未激活视为离线', () => {
    const device: Device = { ...base, is_active: false, last_seen_at: '2024-06-05T09:58:00Z' };
    expect(isDeviceOnline(device, NOW)).toBe(false);
  });

  it('无心跳视为离线', () => {
    expect(isDeviceOnline(base, NOW)).toBe(false);
  });
});

describe('formatBattery', () => {
  it('有电量时展示百分比', () => {
    expect(formatBattery(80)).toBe('80%');
    expect(formatBattery(0)).toBe('0%');
  });

  it('未上报时展示占位符', () => {
    expect(formatBattery(undefined)).toBe('—');
  });
});

describe('DEVICE_COMMANDS / COMMAND_LABELS', () => {
  it('预设指令与后端允许的指令一致', () => {
    expect(DEVICE_COMMANDS.map((c) => c.value).sort()).toEqual([
      'start_recording',
      'stop_recording',
    ]);
    expect(COMMAND_LABELS.start_recording).toBe('开始录音');
    expect(COMMAND_LABELS.stop_recording).toBe('停止录音');
  });
});
