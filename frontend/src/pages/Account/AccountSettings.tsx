import { useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { Input } from '@/components/common/Input';
import { Loading } from '@/components/common/Loading';
import { Modal } from '@/components/common/Modal';
import { useToast } from '@/components/common/Toast';
import { useAuth } from '@/contexts/AuthContext';
import { accountService } from '@/services/account.service';
import { todayStr } from '@/utils/date';
import { downloadBlob } from '@/utils/download';
import type {
  AiSettings,
  AiSettingsInput,
  AsrSettings,
  AsrSettingsInput,
} from '@/types/account.types';

export function Component() {
  const navigate = useNavigate();
  const toast = useToast();
  const { user, logout } = useAuth();

  const [exporting, setExporting] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [confirmEmail, setConfirmEmail] = useState('');
  const [deleting, setDeleting] = useState(false);

  const handleExport = async () => {
    setExporting(true);
    try {
      const data = await accountService.export();
      const json = JSON.stringify(data, null, 2);
      const blob = new Blob([json], { type: 'application/json' });
      downloadBlob(blob, 'pulse-export-' + todayStr() + '.json');
      toast.success('数据已导出');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '导出失败');
    } finally {
      setExporting(false);
    }
  };

  const openDelete = () => {
    setConfirmEmail('');
    setDeleteModalOpen(true);
  };

  const confirmDelete = async () => {
    if (confirmEmail.trim() !== user?.email) {
      toast.error('请输入与账户一致的邮箱以确认注销');
      return;
    }
    setDeleting(true);
    try {
      await accountService.delete();
      toast.success('账户已注销');
      setDeleteModalOpen(false);
      await logout();
      navigate('/auth/register', { replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '注销失败');
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">账户设置</h1>
          <p className="page-subtitle">管理你的个人数据与账户。</p>
        </div>
      </div>

      <AsrConfig />

      <AiConfig />

      <div className="account-section">
        <h2 className="section-title">数据导出</h2>
        <div className="card account-row">
          <div>
            <div className="account-row-title">导出个人数据</div>
            <div className="account-row-desc">
              导出你的账户信息、身份、设备与会话数据为 JSON 文件。
            </div>
          </div>
          <Button variant="secondary" loading={exporting} onClick={handleExport}>
            导出数据
          </Button>
        </div>
      </div>

      <div className="account-section">
        <h2 className="section-title">危险操作</h2>
        <div className="card account-row">
          <div>
            <div className="account-row-title">注销账户</div>
            <div className="account-row-desc">注销将删除你的全部数据，此操作不可撤销。</div>
          </div>
          <Button variant="danger" onClick={openDelete}>
            注销账户
          </Button>
        </div>
      </div>

      <Modal
        open={deleteModalOpen}
        onClose={() => {
          if (!deleting) setDeleteModalOpen(false);
        }}
        title="确认注销账户"
        footer={
          <>
            <Button variant="secondary" disabled={deleting} onClick={() => setDeleteModalOpen(false)}>
              取消
            </Button>
            <Button variant="danger" loading={deleting} onClick={confirmDelete}>
              确认注销
            </Button>
          </>
        }
      >
        <p>注销是不可逆操作，将永久删除你的账户与全部数据。请输入邮箱以确认。</p>
        <div className="form-field">
          <Input
            label="邮箱"
            type="email"
            value={confirmEmail}
            onChange={(e) => setConfirmEmail(e.target.value)}
            placeholder={user?.email ?? 'you@example.com'}
          />
        </div>
      </Modal>
    </div>
  );
}

// ===== ASR 配置 =====

const emptyAsrForm = { api_key: '', base_url: '', model: '', language: 'zh', enable_itn: true };

function AsrConfig() {
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [view, setView] = useState<AsrSettings | null>(null);
  const [form, setForm] = useState(emptyAsrForm);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    accountService
      .getAsr()
      .then((v) => {
        if (!active) return;
        setView(v);
        setForm({
          api_key: '',
          base_url: v.base_url ?? '',
          model: v.model ?? '',
          language: v.language ?? 'zh',
          enable_itn: v.enable_itn,
        });
      })
      .catch((err) => {
        if (active) setLoadError(err instanceof Error ? err.message : '加载 ASR 配置失败');
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const input: AsrSettingsInput = {
        base_url: form.base_url.trim(),
        model: form.model.trim(),
        language: form.language.trim(),
        enable_itn: form.enable_itn,
      };
      const key = form.api_key.trim();
      if (key) input.api_key = key;
      const v = await accountService.updateAsr(input);
      setView(v);
      setForm((f) => ({ ...f, api_key: '' }));
      toast.success('ASR 配置已保存');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleClearKey = async () => {
    setClearing(true);
    try {
      const v = await accountService.updateAsr({ api_key: '' });
      setView(v);
      setForm((f) => ({ ...f, api_key: '' }));
      toast.success('已清除密钥');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '清除失败');
    } finally {
      setClearing(false);
    }
  };

  if (loading) return <Loading text="加载 ASR 配置…" />;

  return (
    <div className="account-section">
      <h2 className="section-title">语音转写（ASR）</h2>
      <div className="card">
        {loadError ? (
          <div className="form-error">{loadError}</div>
        ) : (
          <form onSubmit={handleSubmit} noValidate data-testid="asr-form">
            <p className="settings-desc">
              配置你自己的语音转写服务；字段留空则使用全局默认，密钥加密存储。
            </p>
            <div className="form-field">
              <Input
                id="asr-api-key"
                label="API Key"
                type="password"
                value={form.api_key}
                onChange={(e) => setForm({ ...form, api_key: e.target.value })}
                placeholder={
                  view?.has_api_key
                    ? `已配置（${view.api_key_masked}），输入新值以覆盖`
                    : '未配置，留空使用全局默认'
                }
                autoComplete="off"
              />
              {view?.has_api_key && (
                <div className="settings-key-actions">
                  <Button type="button" variant="ghost" size="small" loading={clearing} onClick={handleClearKey}>
                    清除密钥
                  </Button>
                </div>
              )}
            </div>
            <div className="form-field">
              <Input
                id="asr-base-url"
                label="Base URL"
                value={form.base_url}
                onChange={(e) => setForm({ ...form, base_url: e.target.value })}
                placeholder="留空使用全局默认"
              />
            </div>
            <div className="form-field">
              <Input
                id="asr-model"
                label="模型 Model"
                value={form.model}
                onChange={(e) => setForm({ ...form, model: e.target.value })}
                placeholder="留空使用全局默认"
              />
            </div>
            <div className="form-field">
              <Input
                id="asr-language"
                label="语言 Language"
                value={form.language}
                onChange={(e) => setForm({ ...form, language: e.target.value })}
                placeholder="zh"
              />
            </div>
            <div className="form-field">
              <label className="checkbox-row">
                <input
                  type="checkbox"
                  checked={form.enable_itn}
                  onChange={(e) => setForm({ ...form, enable_itn: e.target.checked })}
                />
                <span>启用 ITN（数字与标点归一化）</span>
              </label>
            </div>
            <div className="settings-actions">
              <Button type="submit" loading={saving}>
                保存 ASR 配置
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

// ===== AI 分析配置 =====

const emptyAiForm = { api_key: '', base_url: '', model: '', confidence_threshold: '0.6' };

function AiConfig() {
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [view, setView] = useState<AiSettings | null>(null);
  const [form, setForm] = useState(emptyAiForm);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [thresholdError, setThresholdError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    accountService
      .getAi()
      .then((v) => {
        if (!active) return;
        setView(v);
        setForm({
          api_key: '',
          base_url: v.base_url ?? '',
          model: v.model ?? '',
          confidence_threshold: String(v.confidence_threshold ?? 0.6),
        });
      })
      .catch((err) => {
        if (active) setLoadError(err instanceof Error ? err.message : '加载 AI 配置失败');
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setThresholdError(null);

    const thresholdStr = form.confidence_threshold.trim();
    let threshold: number | undefined;
    if (thresholdStr) {
      threshold = Number(thresholdStr);
      if (Number.isNaN(threshold) || threshold <= 0 || threshold > 1) {
        setThresholdError('置信度阈值需为 0~1 之间的数字');
        return;
      }
    }

    setSaving(true);
    try {
      const input: AiSettingsInput = {
        base_url: form.base_url.trim(),
        model: form.model.trim(),
      };
      if (threshold !== undefined) input.confidence_threshold = threshold;
      const key = form.api_key.trim();
      if (key) input.api_key = key;
      const v = await accountService.updateAi(input);
      setView(v);
      setForm((f) => ({ ...f, api_key: '' }));
      toast.success('AI 配置已保存');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleClearKey = async () => {
    setClearing(true);
    try {
      const v = await accountService.updateAi({ api_key: '' });
      setView(v);
      setForm((f) => ({ ...f, api_key: '' }));
      toast.success('已清除密钥');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '清除失败');
    } finally {
      setClearing(false);
    }
  };

  if (loading) return <Loading text="加载 AI 配置…" />;

  return (
    <div className="account-section">
      <h2 className="section-title">AI 分析</h2>
      <div className="card">
        {loadError ? (
          <div className="form-error">{loadError}</div>
        ) : (
          <form onSubmit={handleSubmit} noValidate data-testid="ai-form">
            <p className="settings-desc">
              配置用于身份识别与信息提取的模型服务；字段留空则使用全局默认，密钥加密存储。
            </p>
            <div className="form-field">
              <Input
                id="ai-api-key"
                label="API Key"
                type="password"
                value={form.api_key}
                onChange={(e) => setForm({ ...form, api_key: e.target.value })}
                placeholder={
                  view?.has_api_key
                    ? `已配置（${view.api_key_masked}），输入新值以覆盖`
                    : '未配置，留空使用全局默认'
                }
                autoComplete="off"
              />
              {view?.has_api_key && (
                <div className="settings-key-actions">
                  <Button type="button" variant="ghost" size="small" loading={clearing} onClick={handleClearKey}>
                    清除密钥
                  </Button>
                </div>
              )}
            </div>
            <div className="form-field">
              <Input
                id="ai-base-url"
                label="Base URL"
                value={form.base_url}
                onChange={(e) => setForm({ ...form, base_url: e.target.value })}
                placeholder="留空使用全局默认"
              />
            </div>
            <div className="form-field">
              <Input
                id="ai-model"
                label="模型 Model"
                value={form.model}
                onChange={(e) => setForm({ ...form, model: e.target.value })}
                placeholder="留空使用全局默认"
              />
            </div>
            <div className="form-field">
              <Input
                id="ai-confidence-threshold"
                label="置信度阈值"
                type="number"
                step="0.05"
                min="0"
                max="1"
                value={form.confidence_threshold}
                onChange={(e) => setForm({ ...form, confidence_threshold: e.target.value })}
                error={thresholdError ?? undefined}
                helperText={thresholdError ? undefined : '置信度达到该阈值时才自动绑定身份，留空使用默认'}
              />
            </div>
            <div className="settings-actions">
              <Button type="submit" loading={saving}>
                保存 AI 配置
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

export default Component;
