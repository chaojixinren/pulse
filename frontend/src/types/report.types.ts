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

// 按天的一个数据点（趋势图表用）。
export interface DailyPoint {
  date: string; // YYYY-MM-DD
  session_count: number;
  total_duration: number; // 秒
}

// 周报：周一起始日 + 汇总 + 身份拆分 + AI 提取 + 每日趋势。
export interface WeeklyReport {
  week: string; // 周一起始日 YYYY-MM-DD
  session_count: number;
  total_duration: number;
  by_identity: IdentityStat[];
  top_todos: string[];
  commitments_done: number; // 已完成承诺数
  daily_trend: DailyPoint[];
}

// 自定义区间统计。
export interface StatsReport {
  from: string;
  to: string;
  session_count: number;
  total_duration: number;
  by_identity: IdentityStat[];
  daily_trend: DailyPoint[];
}
