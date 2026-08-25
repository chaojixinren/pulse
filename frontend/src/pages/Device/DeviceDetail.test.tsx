import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ToastProvider } from '@/components/common/Toast';
import DeviceDetail from './DeviceDetail';
import type { Device } from '@/types/device.types';

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  issueCommand: vi.fn(),
  unbind: vi.fn(),
}));

vi.mock('@/services/device.service', () => ({
  deviceService: { get: mocks.get, issueCommand: mocks.issueCommand, unbind: mocks.unbind },
}));

const device: Device = {
  id: 'd1',
  user_id: 'u1',
  device_id: 'hw-001',
  name: '我的设备',
  device_type: 'wearable',
  firmware_version: '1.2.0',
  battery_level: 80,
  last_seen_at: new Date().toISOString(),
  is_active: true,
  created_at: '',
  updated_at: '',
};

function renderPage() {
  return render(
    <ToastProvider>
      <MemoryRouter initialEntries={['/devices/d1']}>
        <Routes>
          <Route path="/devices/:id" element={<DeviceDetail />} />
          <Route path="/devices" element={<div>设备列表页</div>} />
        </Routes>
      </MemoryRouter>
    </ToastProvider>,
  );
}

describe('DeviceDetail 页面', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.get.mockResolvedValue(device);
  });

  it('加载并渲染设备详情', async () => {
    renderPage();
    expect(await screen.findByText('我的设备')).toBeInTheDocument();
    expect(screen.getByText('hw-001')).toBeInTheDocument();
    expect(screen.getByText('在线')).toBeInTheDocument();
    expect(screen.getByText('80%')).toBeInTheDocument();
    expect(screen.getByText('1.2.0')).toBeInTheDocument();
  });

  it('下发预设指令并展示落库结果', async () => {
    mocks.issueCommand.mockResolvedValue({
      id: 'cmd1',
      device_id: 'd1',
      user_id: 'u1',
      command: 'start_recording',
      status: 'pending',
      created_at: '',
      updated_at: '',
    });
    renderPage();
    await screen.findByText('我的设备');

    await userEvent.click(screen.getByRole('button', { name: '开始录音' }));
    await userEvent.click(screen.getByRole('button', { name: '下发' }));
    await waitFor(() => expect(mocks.issueCommand).toHaveBeenCalledWith('d1', 'start_recording'));
    expect(await screen.findByText('指令已下发')).toBeInTheDocument();
    expect(screen.getByText('待下发')).toBeInTheDocument();
  });

  it('解绑需二次确认并跳转列表', async () => {
    mocks.unbind.mockResolvedValue(undefined);
    renderPage();
    await screen.findByText('我的设备');

    await userEvent.click(screen.getByRole('button', { name: '解绑设备' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/确定要解绑设备「我的设备」吗/)).toBeInTheDocument();

    await userEvent.click(within(dialog).getByRole('button', { name: '解绑' }));
    await waitFor(() => expect(mocks.unbind).toHaveBeenCalledWith('d1'));
    await waitFor(() => expect(screen.getByText('设备列表页')).toBeInTheDocument());
  });
});
