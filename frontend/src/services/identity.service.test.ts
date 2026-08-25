import { beforeEach, describe, expect, it, vi } from 'vitest';
import { identityService } from './identity.service';
import { http } from './api';

vi.mock('./api', () => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

const identity = {
  id: 'i1',
  user_id: 'u1',
  name: '产品经理',
  color: '#3b82f6',
  icon: '🙂',
  is_default: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('identityService', () => {
  beforeEach(() => {
    vi.mocked(http.get).mockReset();
    vi.mocked(http.post).mockReset();
    vi.mocked(http.put).mockReset();
    vi.mocked(http.delete).mockReset();
  });

  it('list 调用 GET /identities', async () => {
    vi.mocked(http.get).mockResolvedValue([identity]);
    const result = await identityService.list();
    expect(http.get).toHaveBeenCalledWith('/identities');
    expect(result).toEqual([identity]);
  });

  it('create 调用 POST /identities', async () => {
    vi.mocked(http.post).mockResolvedValue(identity);
    await identityService.create({ name: '产品经理' });
    expect(http.post).toHaveBeenCalledWith('/identities', { name: '产品经理' });
  });

  it('update 调用 PUT /identities/:id', async () => {
    vi.mocked(http.put).mockResolvedValue(identity);
    await identityService.update('i1', { name: '新名字' });
    expect(http.put).toHaveBeenCalledWith('/identities/i1', { name: '新名字' });
  });

  it('remove 调用 DELETE /identities/:id', async () => {
    vi.mocked(http.delete).mockResolvedValue(undefined);
    await identityService.remove('i1');
    expect(http.delete).toHaveBeenCalledWith('/identities/i1');
  });

  it('setDefault 调用 PUT /identities/:id/default', async () => {
    vi.mocked(http.put).mockResolvedValue(undefined);
    await identityService.setDefault('i1');
    expect(http.put).toHaveBeenCalledWith('/identities/i1/default');
  });
});
