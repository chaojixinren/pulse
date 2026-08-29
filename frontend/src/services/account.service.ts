import type {
  AccountExport,
  AiSettings,
  AiSettingsInput,
  AsrSettings,
  AsrSettingsInput,
} from '@/types/account.types';
import { http } from './api';

export const accountService = {
  export(): Promise<AccountExport> {
    return http.get<AccountExport>('/account/export');
  },
  delete(): Promise<void> {
    return http.delete('/account');
  },
  getAsr(): Promise<AsrSettings> {
    return http.get<AsrSettings>('/account/asr');
  },
  updateAsr(input: AsrSettingsInput): Promise<AsrSettings> {
    return http.put<AsrSettings>('/account/asr', input);
  },
  getAi(): Promise<AiSettings> {
    return http.get<AiSettings>('/account/ai');
  },
  updateAi(input: AiSettingsInput): Promise<AiSettings> {
    return http.put<AiSettings>('/account/ai', input);
  },
};
