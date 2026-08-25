import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { Input } from '@/components/common/Input';
import { Modal } from '@/components/common/Modal';
import { useToast } from '@/components/common/Toast';
import { useAuth } from '@/contexts/AuthContext';
import { accountService } from '@/services/account.service';
import { todayStr } from '@/utils/date';
import { downloadBlob } from '@/utils/download';

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

export default Component;
