import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ToastProvider } from '@/components/common/Toast';
import DeviceList from './DeviceList';
import type { Device } from '@/types/device.types';

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
}));

vi.mock('@/services/device.service', () => ({
  deviceService: { list: mocks.list },
}));

const device = (overrides: Partial<Device> = {}): Device => ({
  id: 'd1',
  user_id: 'u1',
  device_id: 'hw-001',
  name: '我的设备',
  device_type: 'wearable',
  battery_level: 80,
  last_seen_at: new Date().toISOString(),
  is_active: true,
  created_at: '',
  updated_at: '',
  ...overrides,
});

function renderPage() {
  return render(
    <ToastProvider>
      <MemoryRouter initialEntries={['/devices']}>
        <Routes>
          <Route path="/devices" element={<DeviceList />} />
          <Route path="/devices/bind" element={<div>绑定页</div>} />
          <Route path="/devices/:id" element={<div>详情页</div>} />
        </Routes>
      </MemoryRouter>
    </ToastProvider>,
  );
}

describe('DeviceList 页面', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('加载并渲染设备列表与在线状态', async () => {
    mocks.list.mockResolvedValue([device()]);
    renderPage();

    expect(await screen.findByText('我的设备')).toBeInTheDocument();
    expect(screen.getByText('在线')).toBeInTheDocument();
    expect(screen.getByText(/电量 80%/)).toBeInTheDocument();
  });

  it('空态展示引导', async () => {
    mocks.list.mockResolvedValue([]);
    renderPage();
    expect(await screen.findByText('还没有设备')).toBeInTheDocument();
  });

  it('加载失败展示错误态并可重试', async () => {
    mocks.list.mockRejectedValueOnce(new Error('网络错误'));
    mocks.list.mockResolvedValueOnce([device()]);
    renderPage();

    expect(await screen.findByText('网络错误')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    expect(await screen.findByText('我的设备')).toBeInTheDocument();
  });

  it('点击绑定设备跳转绑定页', async () => {
    mocks.list.mockResolvedValue([]);
    renderPage();
    await screen.findByText('还没有设备');

    await userEvent.click(screen.getByRole('button', { name: '绑定设备' }));
    expect(await screen.findByText('绑定页')).toBeInTheDocument();
  });

  it('点击设备卡片跳转详情页', async () => {
    mocks.list.mockResolvedValue([device()]);
    renderPage();
    await screen.findByText('我的设备');

    await userEvent.click(screen.getByText('我的设备'));
    await waitFor(() => expect(screen.getByText('详情页')).toBeInTheDocument());
  });
});
