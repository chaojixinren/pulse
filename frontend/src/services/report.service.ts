import { DailyReport } from '@/types/report.types';
import { http } from './api';

export const reportService = {
  daily(date: string): Promise<DailyReport> {
    return http.get<DailyReport>('/reports/daily', { params: { date } });
  },
};
