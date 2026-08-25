export interface Device {
  id: string;
  user_id: string;
  device_id: string; // 硬件侧唯一标识
  name: string;
  device_type: string;
  firmware_version?: string;
  battery_level?: number; // 0-100
  last_seen_at?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface DeviceBindCode {
  id: string;
  user_id: string;
  code: string;
  expires_at: string;
  used_at?: string;
  created_at: string;
}

export interface BindDeviceResult {
  device: Device;
  device_token: string; // 一次性返回，需提示用户妥善保存
}

export interface DeviceCommand {
  id: string;
  device_id: string;
  user_id: string;
  command: string;
  status: string; // pending / delivered / …
  created_at: string;
  updated_at: string;
}

export interface BindDeviceInput {
  device_id: string;
  name?: string;
  bind_code: string;
}
