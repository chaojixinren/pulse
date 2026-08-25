import { beforeEach, describe, expect, it, vi } from 'vitest';
import { reportService } from './report.service';
import { http } from './api';

vi.mock('./api', () => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('reportService', () => {
  beforeEach(() => {
    vi.mocked(http.get).mockReset();
  });

  it('daily 调用 GET /reports/daily 并携带日期', async () => {
    const report = { date: '2024-01-01', session_count: 1, total_duration: 60, by_identity: [], todos: [], notes: [] };
    vi.mocked(http.get).mockResolvedValue(report);
    const result = await reportService.daily('2024-01-01');
    expect(http.get).toHaveBeenCalledWith('/reports/daily', { params: { date: '2024-01-01' } });
    expect(result).toEqual(report);
  });

  it('weekly 调用 GET /reports/weekly 并携带周', async () => {
    const report = {
      week: '2024-06-03',
      session_count: 2,
      total_duration: 120,
      by_identity: [],
      top_todos: [],
      commitments_done: 0,
      daily_trend: [],
    };
    vi.mocked(http.get).mockResolvedValue(report);
    const result = await reportService.weekly('2024-06-03');
    expect(http.get).toHaveBeenCalledWith('/reports/weekly', { params: { week: '2024-06-03' } });
    expect(result).toEqual(report);
  });

  it('weekly 不传 week 时接口参数为 undefined', async () => {
    vi.mocked(http.get).mockResolvedValue({});
    await reportService.weekly();
    expect(http.get).toHaveBeenCalledWith('/reports/weekly', { params: { week: undefined } });
  });

  it('stats 调用 GET /reports/stats 并携带区间', async () => {
    const report = {
      from: '2024-05-01',
      to: '2024-05-30',
      session_count: 2,
      total_duration: 120,
      by_identity: [],
      daily_trend: [],
    };
    vi.mocked(http.get).mockResolvedValue(report);
    const result = await reportService.stats('2024-05-01', '2024-05-30');
    expect(http.get).toHaveBeenCalledWith('/reports/stats', {
      params: { from: '2024-05-01', to: '2024-05-30' },
    });
    expect(result).toEqual(report);
  });
});
