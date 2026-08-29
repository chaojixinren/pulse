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

// 语音转写（ASR）配置回显：API Key 不回传明文，仅给掩码。
export interface AsrSettings {
  base_url: string;
  model: string;
  language: string;
  enable_itn: boolean;
  has_api_key: boolean;
  api_key_masked: string;
}

// AI 分析配置回显。
export interface AiSettings {
  base_url: string;
  model: string;
  confidence_threshold: number;
  has_api_key: boolean;
  api_key_masked: string;
}

// ASR 更新入参：字段缺省表示不改变；api_key 传空字符串表示清除。
export interface AsrSettingsInput {
  api_key?: string;
  base_url?: string;
  model?: string;
  language?: string;
  enable_itn?: boolean;
}

// AI 更新入参：字段缺省表示不改变；api_key 传空字符串表示清除。
export interface AiSettingsInput {
  api_key?: string;
  base_url?: string;
  model?: string;
  confidence_threshold?: number;
}
