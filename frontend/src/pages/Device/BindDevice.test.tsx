import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ToastProvider } from '@/components/common/Toast';
import BindDevice from './BindDevice';

const mocks = vi.hoisted(() => ({
  bind: vi.fn(),
}));

vi.mock('@/services/device.service', () => ({
  deviceService: { bind: mocks.bind },
}));

function renderPage() {
  return render(
    <ToastProvider>
      <MemoryRouter initialEntries={['/devices/bind']}>
        <Routes>
          <Route path="/devices/bind" element={<BindDevice />} />
          <Route path="/devices" element={<div>设备列表页</div>} />
          <Route path="/devices/:id" element={<div>详情页</div>} />
        </Routes>
      </MemoryRouter>
    </ToastProvider>,
  );
}

const device = {
  id: 'd1',
  user_id: 'u1',
  device_id: 'hw-001',
  name: '我的设备',
  device_type: 'wearable',
  is_active: true,
  created_at: '',
  updated_at: '',
};

describe('BindDevice 页面', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('渲染绑定表单', () => {
    renderPage();
    expect(screen.getByLabelText('设备 ID *')).toBeInTheDocument();
    expect(screen.getByLabelText('绑定码 *')).toBeInTheDocument();
  });

  it('设备 ID 为空时展示校验错误', async () => {
    renderPage();
    await userEvent.click(screen.getByRole('button', { name: '绑定' }));
    expect(await screen.findByText('请填写设备 ID')).toBeInTheDocument();
    expect(mocks.bind).not.toHaveBeenCalled();
  });

  it('绑定成功后展示一次性设备 Token', async () => {
    mocks.bind.mockResolvedValue({ device, device_token: 'token-abc' });
    renderPage();

    await userEvent.type(screen.getByLabelText('设备 ID *'), 'hw-001');
    await userEvent.type(screen.getByLabelText('绑定码 *'), '123456');
    await userEvent.click(screen.getByRole('button', { name: '绑定' }));

    expect(await screen.findByText('token-abc')).toBeInTheDocument();
    expect(screen.getByText(/仅在本/)).toBeInTheDocument();
    expect(mocks.bind).toHaveBeenCalledWith({
      device_id: 'hw-001',
      name: undefined,
      bind_code: '123456',
    });
  });

  it('绑定成功后返回设备列表', async () => {
    mocks.bind.mockResolvedValue({ device, device_token: 'token-abc' });
    renderPage();

    await userEvent.type(screen.getByLabelText('设备 ID *'), 'hw-001');
    await userEvent.type(screen.getByLabelText('绑定码 *'), '123456');
    await userEvent.click(screen.getByRole('button', { name: '绑定' }));
    await screen.findByText('token-abc');

    await userEvent.click(screen.getByRole('button', { name: '返回设备列表' }));
    await waitFor(() => expect(screen.getByText('设备列表页')).toBeInTheDocument());
  });
});
