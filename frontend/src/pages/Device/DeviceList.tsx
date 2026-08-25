import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { Empty } from '@/components/common/Empty';
import { Loading } from '@/components/common/Loading';
import { Modal } from '@/components/common/Modal';
import { useToast } from '@/components/common/Toast';
import { DeviceCard } from '@/components/business/DeviceCard';
import { deviceService } from '@/services/device.service';
import type { Device, DeviceBindCode } from '@/types/device.types';
import { copyText } from '@/utils/clipboard';
import { formatCountdown } from '@/utils/format';

export function Component() {
  const navigate = useNavigate();
  const toast = useToast();

  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [codeModalOpen, setCodeModalOpen] = useState(false);
  const [bindCode, setBindCode] = useState<DeviceBindCode | null>(null);
  const [generating, setGenerating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const list = await deviceService.list();
      setDevices(list);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const openGenerateCode = async () => {
    setCodeModalOpen(true);
    setBindCode(null);
    setGenerating(true);
    try {
      const code = await deviceService.generateBindCode();
      setBindCode(code);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '生成绑定码失败');
      setCodeModalOpen(false);
    } finally {
      setGenerating(false);
    }
  };

  const closeCodeModal = () => {
    if (generating) return;
    setCodeModalOpen(false);
    setBindCode(null);
  };

  const handleCopyCode = async () => {
    if (!bindCode) return;
    const ok = await copyText(bindCode.code);
    toast[ok ? 'success' : 'error'](ok ? '绑定码已复制' : '复制失败，请手动复制');
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">设备管理</h1>
          <p className="page-subtitle">绑定并管理你的 Pulse 硬件设备，查看在线状态与下发指令。</p>
        </div>
        <div className="page-header-actions">
          <Button variant="secondary" onClick={openGenerateCode}>
            生成绑定码
          </Button>
          <Button onClick={() => navigate('/devices/bind')}>绑定设备</Button>
        </div>
      </div>

      {loading ? (
        <Loading />
      ) : error ? (
        <div className="error-state">
          <div>{error}</div>
          <Button onClick={load}>重试</Button>
        </div>
      ) : devices.length === 0 ? (
        <Empty
          icon="📡"
          title="还没有设备"
          description="生成绑定码，在硬件侧完成首次连接后即可在此看到设备。"
          action={<Button onClick={() => navigate('/devices/bind')}>去绑定设备</Button>}
        />
      ) : (
        <div className="device-grid">
          {devices.map((device) => (
            <DeviceCard key={device.id} device={device} onClick={() => navigate(`/devices/${device.id}`)} />
          ))}
        </div>
      )}

      <Modal
        open={codeModalOpen}
        onClose={closeCodeModal}
        title="设备绑定码"
        footer={
          <Button variant="secondary" disabled={generating} onClick={closeCodeModal}>
            关闭
          </Button>
        }
      >
        {generating ? (
          <Loading text="生成中…" />
        ) : bindCode ? (
          <BindCodeDisplay code={bindCode} onCopy={handleCopyCode} />
        ) : null}
      </Modal>
    </div>
  );
}

function BindCodeDisplay({ code, onCopy }: { code: DeviceBindCode; onCopy: () => void }) {
  const [remaining, setRemaining] = useState(() => secondsUntil(code.expires_at));

  useEffect(() => {
    setRemaining(secondsUntil(code.expires_at));
    const timer = window.setInterval(() => {
      setRemaining(secondsUntil(code.expires_at));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [code.expires_at]);

  const expired = remaining <= 0;

  return (
    <div className="bind-code">
      <div className="bind-code-value">{code.code}</div>
      <div className="bind-code-hint">
        {expired ? '绑定码已过期，请重新生成' : `有效期剩余 ${formatCountdown(remaining)}`}
      </div>
      <div className="bind-code-actions">
        <Button variant="secondary" disabled={expired} onClick={onCopy}>
          复制绑定码
        </Button>
      </div>
    </div>
  );
}

function secondsUntil(expiresAt: string): number {
  const ms = new Date(expiresAt).getTime() - Date.now();
  if (Number.isNaN(ms)) return 0;
  return Math.max(0, Math.floor(ms / 1000));
}

export default Component;
