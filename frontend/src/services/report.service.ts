import { DailyReport, StatsReport, WeeklyReport } from '@/types/report.types';
import { http } from './api';

export const reportService = {
  daily(date: string): Promise<DailyReport> {
    return http.get<DailyReport>('/reports/daily', { params: { date } });
  },
  weekly(week?: string): Promise<WeeklyReport> {
    return http.get<WeeklyReport>('/reports/weekly', { params: { week } });
  },
  stats(from: string, to: string): Promise<StatsReport> {
    return http.get<StatsReport>('/reports/stats', { params: { from, to } });
  },
};
