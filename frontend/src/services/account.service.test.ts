import { beforeEach, describe, expect, it, vi } from 'vitest';
import { accountService } from './account.service';
import { http } from './api';

vi.mock('./api', () => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('accountService', () => {
  beforeEach(() => {
    vi.mocked(http.get).mockReset();
    vi.mocked(http.delete).mockReset();
  });

  it('export 调用 GET /account/export', async () => {
    const payload = { user: { id: 'u1' }, identities: [], devices: [], sessions: [] };
    vi.mocked(http.get).mockResolvedValue(payload);
    const result = await accountService.export();
    expect(http.get).toHaveBeenCalledWith('/account/export');
    expect(result).toEqual(payload);
  });

  it('delete 调用 DELETE /account', async () => {
    vi.mocked(http.delete).mockResolvedValue(undefined);
    await accountService.delete();
    expect(http.delete).toHaveBeenCalledWith('/account');
  });
});
