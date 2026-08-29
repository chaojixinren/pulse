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

  it('create 调用 POST /devices', async () => {
    const res = { device, device_token: 'a'.repeat(64) };
    vi.mocked(http.post).mockResolvedValue(res);
    const result = await deviceService.create({ device_id: 'hw-001', name: '手表' });
    expect(http.post).toHaveBeenCalledWith('/devices', {
      device_id: 'hw-001',
      name: '手表',
    });
    expect(result).toEqual(res);
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
