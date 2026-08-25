import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DeviceCard } from './DeviceCard';
import type { Device } from '@/types/device.types';

const device: Device = {
  id: 'd1',
  user_id: 'u1',
  device_id: 'hw-001',
  name: '我的设备',
  device_type: 'wearable',
  firmware_version: '1.2.0',
  battery_level: 80,
  last_seen_at: new Date().toISOString(),
  is_active: true,
  created_at: '',
  updated_at: '',
};

describe('DeviceCard', () => {
  it('展示名称、类型、电量、固件与在线状态', () => {
    render(<DeviceCard device={device} />);
    expect(screen.getByText('我的设备')).toBeInTheDocument();
    expect(screen.getByText('wearable')).toBeInTheDocument();
    expect(screen.getByText(/电量 80%/)).toBeInTheDocument();
    expect(screen.getByText(/固件 1\.2\.0/)).toBeInTheDocument();
    expect(screen.getByText('在线')).toBeInTheDocument();
  });

  it('无心跳时展示离线与占位符', () => {
    render(<DeviceCard device={{ ...device, last_seen_at: undefined, battery_level: undefined }} />);
    expect(screen.getByText('离线')).toBeInTheDocument();
    expect(screen.getByText(/电量 —/)).toBeInTheDocument();
  });

  it('可点击时触发 onClick', async () => {
    const onClick = vi.fn();
    render(<DeviceCard device={device} onClick={onClick} />);
    await userEvent.click(screen.getByText('我的设备'));
    expect(onClick).toHaveBeenCalledWith(device);
  });
});
