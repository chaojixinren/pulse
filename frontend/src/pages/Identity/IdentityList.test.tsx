import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToastProvider } from '@/components/common/Toast';
import IdentityList from './IdentityList';
import type { Identity } from '@/types/identity.types';

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  setDefault: vi.fn(),
}));

vi.mock('@/services/identity.service', () => ({
  identityService: {
    list: mocks.list,
    create: mocks.create,
    update: mocks.update,
    remove: mocks.remove,
    setDefault: mocks.setDefault,
  },
}));

const identity = (overrides: Partial<Identity> = {}): Identity => ({
  id: 'i1',
  user_id: 'u1',
  name: '产品经理',
  description: '负责规划',
  color: '#3b82f6',
  icon: '🙂',
  is_default: false,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

function renderPage() {
  return render(
    <ToastProvider>
      <IdentityList />
    </ToastProvider>,
  );
}

describe('IdentityList 页面', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('加载并渲染身份列表', async () => {
    mocks.list.mockResolvedValue([
      identity({ id: 'i1', name: '产品经理', is_default: true }),
      identity({ id: 'i2', name: '健身教练' }),
    ]);
    renderPage();

    expect(await screen.findByText('产品经理')).toBeInTheDocument();
    expect(screen.getByText('健身教练')).toBeInTheDocument();
    expect(screen.getByText('默认')).toBeInTheDocument();
  });

  it('空态展示引导', async () => {
    mocks.list.mockResolvedValue([]);
    renderPage();
    expect(await screen.findByText('还没有身份')).toBeInTheDocument();
  });

  it('加载失败展示错误态并可重试', async () => {
    mocks.list.mockRejectedValueOnce(new Error('网络错误'));
    mocks.list.mockResolvedValueOnce([identity()]);
    renderPage();

    expect(await screen.findByText('网络错误')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '重试' }));
    expect(await screen.findByText('产品经理')).toBeInTheDocument();
  });

  it('创建身份后刷新列表', async () => {
    mocks.list.mockResolvedValue([]);
    mocks.create.mockResolvedValue(identity({ id: 'new', name: '新身份' }));
    renderPage();

    await userEvent.click(await screen.findByRole('button', { name: '+ 创建身份' }));
    const dialog = await screen.findByRole('dialog');
    await userEvent.type(within(dialog).getByPlaceholderText('例如：产品经理、健身教练'), '新身份');
    await userEvent.click(within(dialog).getByRole('button', { name: '保存' }));

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({ name: '新身份' }),
      ),
    );
    await waitFor(() => expect(screen.getByText('身份已创建')).toBeInTheDocument());
  });

  it('编辑身份后生效', async () => {
    mocks.list.mockResolvedValue([identity()]);
    mocks.update.mockResolvedValue(identity({ name: '改后的名字' }));
    renderPage();

    await userEvent.click(await screen.findByRole('button', { name: '编辑' }));
    const dialog = await screen.findByRole('dialog');
    const nameInput = within(dialog).getByPlaceholderText('例如：产品经理、健身教练');
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, '改后的名字');
    await userEvent.click(within(dialog).getByRole('button', { name: '保存' }));

    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledWith(
        'i1',
        expect.objectContaining({ name: '改后的名字' }),
      ),
    );
  });

  it('删除需二次确认并更新列表', async () => {
    mocks.list.mockResolvedValue([identity()]);
    mocks.remove.mockResolvedValue(undefined);
    renderPage();

    await userEvent.click(await screen.findByRole('button', { name: '删除' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/确定要删除身份「产品经理」吗/)).toBeInTheDocument();

    await userEvent.click(within(dialog).getByRole('button', { name: '删除' }));
    await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith('i1'));
    await waitFor(() => expect(screen.getByText('身份已删除')).toBeInTheDocument());
  });

  it('设为默认后调用接口', async () => {
    mocks.list.mockResolvedValue([identity()]);
    mocks.setDefault.mockResolvedValue(undefined);
    renderPage();

    await userEvent.click(await screen.findByRole('button', { name: '设为默认' }));
    await waitFor(() => expect(mocks.setDefault).toHaveBeenCalledWith('i1'));
    await waitFor(() => expect(screen.getByText('已设为默认身份')).toBeInTheDocument());
  });
});
