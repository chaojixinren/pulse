import type { User } from './api.types';
import type { Identity } from './identity.types';
import type { Device } from './device.types';

// 个人数据导出：用户 + 身份 + 设备 + 音频会话（含转写与提取）。
export interface AccountExport {
  user: User;
  identities: Identity[];
  devices: Device[];
  sessions: Array<Record<string, unknown>>;
}
