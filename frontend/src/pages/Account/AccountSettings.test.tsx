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
  getAsr: vi.fn(),
  updateAsr: vi.fn(),
  getAi: vi.fn(),
  updateAi: vi.fn(),
}));

vi.mock('@/services/account.service', () => ({
  accountService: {
    export: mocks.exportData,
    delete: mocks.deleteAccount,
    getAsr: mocks.getAsr,
    updateAsr: mocks.updateAsr,
    getAi: mocks.getAi,
    updateAi: mocks.updateAi,
  },
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
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getAsr.mockResolvedValue({
      base_url: '',
      model: '',
      language: 'zh',
      enable_itn: true,
      has_api_key: false,
      api_key_masked: '',
    });
    mocks.getAi.mockResolvedValue({
      base_url: '',
      model: '',
      confidence_threshold: 0.6,
      has_api_key: false,
      api_key_masked: '',
    });
  });

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

  it('保存 ASR 配置时提交表单字段并回填掩码', async () => {
    mocks.updateAsr.mockResolvedValue({
      base_url: '',
      model: 'step-asr',
      language: 'zh',
      enable_itn: true,
      has_api_key: false,
      api_key_masked: '',
    });
    renderPage();

    await screen.findByRole('button', { name: '保存 ASR 配置' });
    const asrForm = screen.getByTestId('asr-form');
    await userEvent.type(within(asrForm).getByLabelText('模型 Model'), 'step-asr');
    await userEvent.click(within(asrForm).getByRole('button', { name: '保存 ASR 配置' }));

    await waitFor(() => expect(mocks.updateAsr).toHaveBeenCalled());
    const input = mocks.updateAsr.mock.calls[0][0] as Record<string, unknown>;
    expect(input.model).toBe('step-asr');
    expect(input).not.toHaveProperty('api_key');
  });

  it('已配置密钥时显示掩码与清除入口', async () => {
    mocks.getAsr.mockResolvedValue({
      base_url: '',
      model: '',
      language: 'zh',
      enable_itn: true,
      has_api_key: true,
      api_key_masked: '****abcd',
    });
    renderPage();

    const asrForm = await screen.findByTestId('asr-form');
    const apiKeyInput = await within(asrForm).findByLabelText('API Key');
    expect(apiKeyInput).toHaveAttribute('placeholder', expect.stringContaining('****abcd'));

    await userEvent.click(within(asrForm).getByRole('button', { name: '清除密钥' }));
    await waitFor(() => expect(mocks.updateAsr).toHaveBeenCalledWith({ api_key: '' }));
  });

  it('AI 阈值越界时阻止提交并提示', async () => {
    renderPage();

    const threshold = await screen.findByLabelText('置信度阈值');
    await userEvent.clear(threshold);
    await userEvent.type(threshold, '2');
    await userEvent.click(screen.getByRole('button', { name: '保存 AI 配置' }));

    expect(mocks.updateAi).not.toHaveBeenCalled();
    await screen.findByText('置信度阈值需为 0~1 之间的数字');
  });
});
