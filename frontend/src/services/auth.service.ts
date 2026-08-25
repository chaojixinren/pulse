import { TokenPair, User } from '@/types/api.types';
import { http } from './api';

export const authService = {
  register(data: { email: string; password: string; name: string }): Promise<User> {
    return http.post<User>('/auth/register', data);
  },
  login(data: { email: string; password: string }): Promise<TokenPair> {
    return http.post<TokenPair>('/auth/login', data);
  },
  logout(refresh_token: string): Promise<void> {
    return http.post<void>('/auth/logout', { refresh_token });
  },
  refresh(refresh_token: string): Promise<TokenPair> {
    return http.post<TokenPair>('/auth/refresh', { refresh_token });
  },
  me(): Promise<User> {
    return http.get<User>('/auth/me');
  },
};
