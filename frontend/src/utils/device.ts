import type { Device } from '@/types/device.types';

// 在线判定阈值：最近一次心跳在 5 分钟内视为在线。
const ONLINE_THRESHOLD_MS = 5 * 60 * 1000;

// 设备是否在线：需设备处于激活状态，且最近一次心跳在阈值内。
export function isDeviceOnline(device: Device, now: number = Date.now()): boolean {
  if (!device.is_active || !device.last_seen_at) return false;
  const lastSeen = new Date(device.last_seen_at).getTime();
  if (Number.isNaN(lastSeen)) return false;
  return now - lastSeen <= ONLINE_THRESHOLD_MS;
}

// 电量展示：未上报时返回占位符。
export function formatBattery(level: number | undefined): string {
  if (level === undefined || level === null) return '—';
  return `${level}%`;
}

// 指令下发的预设选项（与后端允许的指令集合保持一致）。
export const DEVICE_COMMANDS: { value: string; label: string }[] = [
  { value: 'start_recording', label: '开始录音' },
  { value: 'stop_recording', label: '停止录音' },
];

export const COMMAND_LABELS: Record<string, string> = {
  start_recording: '开始录音',
  stop_recording: '停止录音',
  pending: '待下发',
  delivered: '已送达',
};
