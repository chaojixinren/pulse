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
});
