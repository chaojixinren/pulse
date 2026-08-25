import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { ToastProvider } from '@/components/common/Toast';
import AccountSettings from './AccountSettings';

const mocks = vi.hoisted(() => ({
  exportData: vi.fn(),
  deleteAccount: vi.fn(),
  logout: vi.fn(),
  downloadBlob: vi.fn(),
}));

vi.mock('@/services/account.service', () => ({
  accountService: { export: mocks.exportData, delete: mocks.deleteAccount },
}));

vi.mock('@/utils/download', () => ({
  downloadBlob: mocks.downloadBlob,
}));

vi.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({ user: { email: 'user@example.com' }, logout: mocks.logout }),
}));

function renderPage() {
  return render(
    <MemoryRouter>
      <ToastProvider>
        <AccountSettings />
      </ToastProvider>
    </MemoryRouter>,
  );
}

describe('AccountSettings 页面', () => {
  beforeEach(() => vi.clearAllMocks());

  it('导出数据调用接口并下载 JSON', async () => {
    mocks.exportData.mockResolvedValue({
      user: { email: 'user@example.com' },
      identities: [],
      devices: [],
      sessions: [],
    });
    renderPage();

    await userEvent.click(screen.getByRole('button', { name: '导出数据' }));
    await waitFor(() => expect(mocks.exportData).toHaveBeenCalled());
    await waitFor(() => expect(mocks.downloadBlob).toHaveBeenCalled());

    const [blob, filename] = mocks.downloadBlob.mock.calls[0] as [Blob, string];
    expect(filename).toMatch(/^pulse-export-\d{4}-\d{2}-\d{2}\.json$/);
    expect(blob.type).toBe('application/json');
  });

  it('注销需邮箱二次确认', async () => {
    mocks.deleteAccount.mockResolvedValue(undefined);
    mocks.logout.mockResolvedValue(undefined);
    renderPage();

    await userEvent.click(screen.getByRole('button', { name: '注销账户' }));
    const dialog = await screen.findByRole('dialog');

    // 邮箱不匹配时不调用删除
    await userEvent.type(within(dialog).getByLabelText('邮箱'), 'wrong@example.com');
    await userEvent.click(within(dialog).getByRole('button', { name: '确认注销' }));
    expect(mocks.deleteAccount).not.toHaveBeenCalled();

    // 输入正确邮箱后注销并登出
    await userEvent.clear(within(dialog).getByLabelText('邮箱'));
    await userEvent.type(within(dialog).getByLabelText('邮箱'), 'user@example.com');
    await userEvent.click(within(dialog).getByRole('button', { name: '确认注销' }));

    await waitFor(() => expect(mocks.deleteAccount).toHaveBeenCalled());
    await waitFor(() => expect(mocks.logout).toHaveBeenCalled());
  });
});
