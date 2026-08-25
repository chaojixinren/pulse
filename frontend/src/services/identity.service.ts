import { Identity, IdentityInput } from '@/types/identity.types';
import { http } from './api';

export const identityService = {
  list(): Promise<Identity[]> {
    return http.get<Identity[]>('/identities');
  },
  create(data: IdentityInput): Promise<Identity> {
    return http.post<Identity>('/identities', data);
  },
  update(id: string, data: IdentityInput): Promise<Identity> {
    return http.put<Identity>(`/identities/${id}`, data);
  },
  remove(id: string): Promise<void> {
    return http.delete(`/identities/${id}`);
  },
  setDefault(id: string): Promise<void> {
    return http.put<void>(`/identities/${id}/default`);
  },
};
