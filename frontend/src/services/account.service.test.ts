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
    vi.mocked(http.put).mockReset();
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

  it('getAsr 调用 GET /account/asr', async () => {
    const payload = { base_url: '', model: '', language: 'zh', enable_itn: true, has_api_key: false, api_key_masked: '' };
    vi.mocked(http.get).mockResolvedValue(payload);
    const result = await accountService.getAsr();
    expect(http.get).toHaveBeenCalledWith('/account/asr');
    expect(result).toEqual(payload);
  });

  it('updateAsr 调用 PUT /account/asr', async () => {
    const input = { model: 'step-asr' };
    vi.mocked(http.put).mockResolvedValue(input);
    await accountService.updateAsr(input);
    expect(http.put).toHaveBeenCalledWith('/account/asr', input);
  });

  it('getAi 调用 GET /account/ai', async () => {
    const payload = { base_url: '', model: '', confidence_threshold: 0.6, has_api_key: false, api_key_masked: '' };
    vi.mocked(http.get).mockResolvedValue(payload);
    const result = await accountService.getAi();
    expect(http.get).toHaveBeenCalledWith('/account/ai');
    expect(result).toEqual(payload);
  });

  it('updateAi 调用 PUT /account/ai', async () => {
    const input = { confidence_threshold: 0.8 };
    vi.mocked(http.put).mockResolvedValue(input);
    await accountService.updateAi(input);
    expect(http.put).toHaveBeenCalledWith('/account/ai', input);
  });
});
