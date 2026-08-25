export type TimelineStatus = 'pending' | 'processing' | 'completed' | 'failed';

export interface TimelineItem {
  session_id: string;
  identity_id?: string; // 低置信度时后端可能不返回
  transcript: string;
  duration: number; // 秒
  status: TimelineStatus;
  recorded_at: string;
}

export interface TimelineQuery {
  identity_id?: string;
  from?: string; // RFC3339
  to?: string; // RFC3339
  status?: string;
  page?: number;
  size?: number;
}
