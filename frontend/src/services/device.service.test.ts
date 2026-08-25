import { beforeEach, describe, expect, it, vi } from 'vitest';
import { deviceService } from './device.service';
import { http } from './api';

vi.mock('./api', () => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

const device = {
  id: 'd1',
  user_id: 'u1',
  device_id: 'hw-001',
  name: '我的设备',
  device_type: 'wearable',
  is_active: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('deviceService', () => {
  beforeEach(() => {
    vi.mocked(http.get).mockReset();
    vi.mocked(http.post).mockReset();
    vi.mocked(http.put).mockReset();
    vi.mocked(http.delete).mockReset();
  });

  it('generateBindCode 调用 POST /devices/bind-code', async () => {
    const code = { id: 'c1', user_id: 'u1', code: '123456', expires_at: '2024-01-01T00:10:00Z', created_at: '' };
    vi.mocked(http.post).mockResolvedValue(code);
    const result = await deviceService.generateBindCode();
    expect(http.post).toHaveBeenCalledWith('/devices/bind-code');
    expect(result).toEqual(code);
  });

  it('bind 调用 POST /devices/bind', async () => {
    const res = { device, device_token: 'token' };
    vi.mocked(http.post).mockResolvedValue(res);
    await deviceService.bind({ device_id: 'hw-001', bind_code: '123456' });
    expect(http.post).toHaveBeenCalledWith('/devices/bind', {
      device_id: 'hw-001',
      bind_code: '123456',
    });
  });

  it('list 调用 GET /devices', async () => {
    vi.mocked(http.get).mockResolvedValue([device]);
    const result = await deviceService.list();
    expect(http.get).toHaveBeenCalledWith('/devices');
    expect(result).toEqual([device]);
  });

  it('get 调用 GET /devices/:id', async () => {
    vi.mocked(http.get).mockResolvedValue(device);
    await deviceService.get('d1');
    expect(http.get).toHaveBeenCalledWith('/devices/d1');
  });

  it('unbind 调用 DELETE /devices/:id', async () => {
    vi.mocked(http.delete).mockResolvedValue(undefined);
    await deviceService.unbind('d1');
    expect(http.delete).toHaveBeenCalledWith('/devices/d1');
  });

  it('issueCommand 调用 POST /devices/:id/command', async () => {
    const cmd = { id: 'cmd1', device_id: 'd1', user_id: 'u1', command: 'start_recording', status: 'pending', created_at: '', updated_at: '' };
    vi.mocked(http.post).mockResolvedValue(cmd);
    await deviceService.issueCommand('d1', 'start_recording');
    expect(http.post).toHaveBeenCalledWith('/devices/d1/command', { command: 'start_recording' });
  });
});
