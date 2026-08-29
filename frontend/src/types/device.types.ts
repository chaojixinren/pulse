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

export interface DeviceCommand {
  id: string;
  device_id: string;
  user_id: string;
  command: string;
  status: string; // pending / delivered / …
  created_at: string;
  updated_at: string;
}

// App 创建/绑定设备，一次性返回 device_token 供手抄到硬件 config.json。
export interface CreateDeviceInput {
  device_id: string; // 硬件侧唯一标识，需与 config.json 中 cloud.device_id 一致
  name?: string;
}

export interface CreateDeviceResult {
  device: Device;
  device_token: string; // 一次性明文 token，仅此响应返回，服务端只存 hash
}
