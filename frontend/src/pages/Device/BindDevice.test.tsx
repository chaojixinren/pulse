import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ToastProvider } from '@/components/common/Toast';
import BindDevice from './BindDevice';

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  copyText: vi.fn(),
}));

vi.mock('@/services/device.service', () => ({
  deviceService: { create: mocks.create },
}));

vi.mock('@/utils/clipboard', () => ({
  copyText: mocks.copyText,
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

const token = 'a'.repeat(64);

describe('BindDevice 页面', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('渲染绑定表单', () => {
    renderPage();
    expect(screen.getByLabelText('设备 ID *')).toBeInTheDocument();
    expect(screen.getByLabelText('设备名称')).toBeInTheDocument();
  });

  it('设备 ID 为空时展示校验错误', async () => {
    renderPage();
    await userEvent.click(screen.getByRole('button', { name: '创建并获取 token' }));
    expect(await screen.findByText('请填写设备 ID')).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it('创建成功后展示 token 并可复制', async () => {
    mocks.create.mockResolvedValue({ device, device_token: token });
    mocks.copyText.mockResolvedValue(true);
    renderPage();

    await userEvent.type(screen.getByLabelText('设备 ID *'), 'hw-001');
    await userEvent.type(screen.getByLabelText('设备名称'), '手表');
    await userEvent.click(screen.getByRole('button', { name: '创建并获取 token' }));

    expect(await screen.findByText('设备已创建')).toBeInTheDocument();
    expect(screen.getByTestId('device-token')).toBeInTheDocument();
    expect(mocks.create).toHaveBeenCalledWith({ device_id: 'hw-001', name: '手表' });

    await userEvent.click(screen.getByRole('button', { name: '复制 token' }));
    expect(mocks.copyText).toHaveBeenCalledWith(token);
  });

  it('点击「我已抄录，完成」跳转设备详情', async () => {
    mocks.create.mockResolvedValue({ device, device_token: token });
    renderPage();

    await userEvent.type(screen.getByLabelText('设备 ID *'), 'hw-001');
    await userEvent.click(screen.getByRole('button', { name: '创建并获取 token' }));

    await screen.findByText('设备已创建');
    await userEvent.click(screen.getByRole('button', { name: '我已抄录，完成' }));
    expect(await screen.findByText('详情页')).toBeInTheDocument();
  });

  it('创建失败展示后端错误信息', async () => {
    mocks.create.mockRejectedValue(new Error('该设备已被其他用户绑定'));
    renderPage();

    await userEvent.type(screen.getByLabelText('设备 ID *'), 'hw-001');
    await userEvent.click(screen.getByRole('button', { name: '创建并获取 token' }));

    expect(await screen.findByText('该设备已被其他用户绑定')).toBeInTheDocument();
  });
});
