import type { AccountExport } from '@/types/account.types';
import { http } from './api';

export const accountService = {
  export(): Promise<AccountExport> {
    return http.get<AccountExport>('/account/export');
  },
  delete(): Promise<void> {
    return http.delete('/account');
  },
};
