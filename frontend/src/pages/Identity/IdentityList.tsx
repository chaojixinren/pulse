import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { Button } from '@/components/common/Button';
import { Empty } from '@/components/common/Empty';
import { Loading } from '@/components/common/Loading';
import { Modal } from '@/components/common/Modal';
import { useToast } from '@/components/common/Toast';
import { IdentityCard } from '@/components/business/IdentityCard';
import { identityService } from '@/services/identity.service';
import type { Identity, IdentityInput } from '@/types/identity.types';

const COLOR_OPTIONS = [
  '#3b82f6',
  '#10b981',
  '#f59e0b',
  '#ef4444',
  '#8b5cf6',
  '#ec4899',
  '#06b6d4',
  '#6b7280',
];
const DEFAULT_COLOR = '#3b82f6';

interface IdentityFormState {
  name: string;
  description: string;
  color: string;
  icon: string;
  is_default: boolean;
}

const emptyForm: IdentityFormState = {
  name: '',
  description: '',
  color: DEFAULT_COLOR,
  icon: '',
  is_default: false,
};

export function Component() {
  const toast = useToast();
  const [identities, setIdentities] = useState<Identity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<IdentityFormState>(emptyForm);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [deleteTarget, setDeleteTarget] = useState<Identity | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const list = await identityService.list();
      setIdentities(list);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const openCreate = () => {
    setEditingId(null);
    setForm(emptyForm);
    setFormError(null);
    setModalOpen(true);
  };

  const openEdit = (identity: Identity) => {
    setEditingId(identity.id);
    setForm({
      name: identity.name,
      description: identity.description ?? '',
      color: identity.color || DEFAULT_COLOR,
      icon: identity.icon || '',
      is_default: identity.is_default,
    });
    setFormError(null);
    setModalOpen(true);
  };

  const closeModal = () => {
    if (submitting) return;
    setModalOpen(false);
  };

  const save = async () => {
    setFormError(null);
    const name = form.name.trim();
    if (!name) {
      setFormError('请填写身份名称');
      return;
    }

    const payload: IdentityInput = {
      name,
      description: form.description.trim() || undefined,
      color: form.color,
      icon: form.icon.trim() || undefined,
      is_default: form.is_default,
    };

    setSubmitting(true);
    try {
      if (editingId) {
        await identityService.update(editingId, payload);
        toast.success('身份已更新');
      } else {
        await identityService.create(payload);
        toast.success('身份已创建');
      }
      setModalOpen(false);
      await load();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleFormSubmit = (e: FormEvent) => {
    e.preventDefault();
    save();
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await identityService.remove(deleteTarget.id);
      toast.success('身份已删除');
      setDeleteTarget(null);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '删除失败');
    } finally {
      setDeleting(false);
    }
  };

  const handleSetDefault = async (identity: Identity) => {
    try {
      await identityService.setDefault(identity.id);
      toast.success('已设为默认身份');
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '设置失败');
    }
  };

  if (loading) return <Loading />;
  if (error) {
    return (
      <div className="error-state">
        <div>{error}</div>
        <Button onClick={load}>重试</Button>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">身份管理</h1>
          <p className="page-subtitle">管理你的身份，用于时间线与日报的身份维度。</p>
        </div>
        <Button onClick={openCreate}>+ 创建身份</Button>
      </div>

      {identities.length === 0 ? (
        <Empty
          title="还没有身份"
          description="创建你的第一个身份，开始记录语音会话。"
          action={<Button onClick={openCreate}>创建身份</Button>}
        />
      ) : (
        <div className="identity-grid">
          {identities.map((identity) => (
            <IdentityCard
              key={identity.id}
              identity={identity}
              onEdit={openEdit}
              onDelete={setDeleteTarget}
              onSetDefault={handleSetDefault}
            />
          ))}
        </div>
      )}

      <Modal
        open={modalOpen}
        onClose={closeModal}
        title={editingId ? '编辑身份' : '创建身份'}
        footer={
          <>
            <Button variant="secondary" disabled={submitting} onClick={closeModal}>
              取消
            </Button>
            <Button loading={submitting} onClick={save}>
              保存
            </Button>
          </>
        }
      >
        <form onSubmit={handleFormSubmit} noValidate>
          <div className="form-field">
            <label className="form-label">名称 *</label>
            <input
              className="form-input"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="例如：产品经理、健身教练"
            />
          </div>
          <div className="form-field">
            <label className="form-label">描述</label>
            <textarea
              className="form-textarea"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="可选，一句话说明这个身份"
            />
          </div>
          <div className="form-field">
            <label className="form-label">颜色</label>
            <div className="color-swatches">
              {COLOR_OPTIONS.map((color) => (
                <button
                  key={color}
                  type="button"
                  className={'color-swatch' + (form.color === color ? ' selected' : '')}
                  style={{ backgroundColor: color }}
                  onClick={() => setForm({ ...form, color })}
                  aria-label={color}
                />
              ))}
            </div>
          </div>
          <div className="form-field">
            <label className="form-label">图标</label>
            <input
              className="form-input"
              value={form.icon}
              onChange={(e) => setForm({ ...form, icon: e.target.value })}
              placeholder="可选，留空则显示首字符"
            />
          </div>
          <div className="form-field">
            <label className="checkbox-row">
              <input
                type="checkbox"
                checked={form.is_default}
                onChange={(e) => setForm({ ...form, is_default: e.target.checked })}
              />
              设为默认身份
            </label>
          </div>
          {formError && <div className="form-error">{formError}</div>}
        </form>
      </Modal>

      <Modal
        open={Boolean(deleteTarget)}
        onClose={() => setDeleteTarget(null)}
        title="确认删除"
        footer={
          <>
            <Button variant="secondary" disabled={deleting} onClick={() => setDeleteTarget(null)}>
              取消
            </Button>
            <Button variant="danger" loading={deleting} onClick={confirmDelete}>
              删除
            </Button>
          </>
        }
      >
        <p>确定要删除身份「{deleteTarget?.name}」吗？该操作不可撤销。</p>
      </Modal>
    </div>
  );
}

export default Component;
