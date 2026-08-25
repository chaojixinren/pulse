export interface IdentityStat {
  identity_id: string;
  name: string;
  session_count: number;
  total_duration: number;
}

export interface DailyReport {
  date: string; // YYYY-MM-DD
  session_count: number;
  total_duration: number; // 秒
  by_identity: IdentityStat[];
  todos: string[]; // AI 提取的待办
  notes: string[]; // AI 提取的笔记
}
