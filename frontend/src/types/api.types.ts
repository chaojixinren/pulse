// 后端统一响应：code=0 表示成功
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  settings?: string;
  created_at: string;
  updated_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface Paginated<T> {
  items: T[];
  total: number;
  page: number;
  size: number;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly code: number,
    public readonly status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
