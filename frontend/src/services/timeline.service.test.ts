import { beforeEach, describe, expect, it, vi } from 'vitest';
import { timelineService } from './timeline.service';
import { http } from './api';

vi.mock('./api', () => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('timelineService', () => {
  beforeEach(() => {
    vi.mocked(http.get).mockReset();
  });

  it('list 调用 GET /timeline 并透传查询参数', async () => {
    const paginated = { items: [], total: 0, page: 1, size: 20 };
    vi.mocked(http.get).mockResolvedValue(paginated);
    const query = { identity_id: 'i1', status: 'completed', from: '2024-01-01T00:00:00Z', page: 1, size: 20 };
    const result = await timelineService.list(query);
    expect(http.get).toHaveBeenCalledWith('/timeline', { params: query });
    expect(result).toEqual(paginated);
  });
});
