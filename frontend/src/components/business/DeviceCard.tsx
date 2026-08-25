import type { KeyboardEvent } from 'react';
import type { Device } from '@/types/device.types';
import { formatDateTime } from '@/utils/date';
import { formatBattery, isDeviceOnline } from '@/utils/device';

export interface DeviceCardProps {
  device: Device;
  onClick?: (device: Device) => void;
}

// 设备卡片：在线状态 / 电量 / 固件版本 / 最后活跃时间。
export function DeviceCard({ device, onClick }: DeviceCardProps) {
  const online = isDeviceOnline(device);
  const clickable = Boolean(onClick);

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (onClick && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault();
      onClick(device);
    }
  };

  return (
    <div
      className={'device-card' + (clickable ? ' device-card-clickable' : '')}
      role={clickable ? 'button' : undefined}
      tabIndex={clickable ? 0 : undefined}
      onClick={clickable ? () => onClick?.(device) : undefined}
      onKeyDown={clickable ? handleKeyDown : undefined}
    >
      <div className="device-card-head">
        <div className="device-avatar">📡</div>
        <div className="device-card-title">
          <div className="device-name-row">
            <span className="device-name">{device.name}</span>
            <span className={'device-status ' + (online ? 'device-online' : 'device-offline')}>
              {online ? '在线' : '离线'}
            </span>
          </div>
          <div className="device-type">{device.device_type}</div>
        </div>
      </div>
      <div className="device-meta">
        <span className="device-meta-item">电量 {formatBattery(device.battery_level)}</span>
        <span className="device-meta-item">
          固件 {device.firmware_version || '—'}
        </span>
        <span className="device-meta-item">
          最后活跃 {device.last_seen_at ? formatDateTime(device.last_seen_at) : '—'}
        </span>
      </div>
    </div>
  );
}
