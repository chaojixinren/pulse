export interface Identity {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  color: string;
  icon: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface IdentityInput {
  name: string; // 必填
  description?: string;
  color?: string;
  icon?: string;
  is_default?: boolean;
}
