import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { Loading } from '@/components/common/Loading';
import { Modal } from '@/components/common/Modal';
import { useToast } from '@/components/common/Toast';
import { deviceService } from '@/services/device.service';
import type { Device, DeviceCommand } from '@/types/device.types';
import { formatDateTime } from '@/utils/date';
import { COMMAND_LABELS, DEVICE_COMMANDS, formatBattery, isDeviceOnline } from '@/utils/device';

export function Component() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();

  const [device, setDevice] = useState<Device | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [command, setCommand] = useState('');
  const [sending, setSending] = useState(false);
  const [lastCommand, setLastCommand] = useState<DeviceCommand | null>(null);

  const [confirmUnbind, setConfirmUnbind] = useState(false);
  const [unbinding, setUnbinding] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const data = await deviceService.get(id);
      setDevice(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  const handleIssue = async () => {
    if (!id || !command) return;
    setSending(true);
    try {
      const cmd = await deviceService.issueCommand(id, command);
      setLastCommand(cmd);
      toast.success('指令已下发');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '指令下发失败');
    } finally {
      setSending(false);
    }
  };

  const handleUnbind = async () => {
    if (!id) return;
    setUnbinding(true);
    try {
      await deviceService.unbind(id);
      toast.success('设备已解绑');
      navigate('/devices');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '解绑失败');
      setUnbinding(false);
      setConfirmUnbind(false);
    }
  };

  if (loading) return <Loading />;
  if (error || !device) {
    return (
      <div className="error-state">
        <div>{error ?? '设备不存在'}</div>
        <Button onClick={load}>重试</Button>
      </div>
    );
  }

  const online = isDeviceOnline(device);

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">{device.name}</h1>
          <p className="page-subtitle">设备详情、指令下发与解绑。</p>
        </div>
        <div className="page-header-actions">
          <Button variant="secondary" onClick={() => navigate('/devices')}>
            返回列表
          </Button>
          <Button variant="danger" onClick={() => setConfirmUnbind(true)}>
            解绑设备
          </Button>
        </div>
      </div>

      <div className="device-detail-grid">
        <div className="card">
          <h2 className="section-title">设备信息</h2>
          <dl className="detail-list">
            <div className="detail-row">
              <dt>设备 ID</dt>
              <dd>{device.device_id}</dd>
            </div>
            <div className="detail-row">
              <dt>设备类型</dt>
              <dd>{device.device_type}</dd>
            </div>
            <div className="detail-row">
              <dt>在线状态</dt>
              <dd>
                <span className={'device-status ' + (online ? 'device-online' : 'device-offline')}>
                  {online ? '在线' : '离线'}
                </span>
              </dd>
            </div>
            <div className="detail-row">
              <dt>电量</dt>
              <dd>{formatBattery(device.battery_level)}</dd>
            </div>
            <div className="detail-row">
              <dt>固件版本</dt>
              <dd>{device.firmware_version || '—'}</dd>
            </div>
            <div className="detail-row">
              <dt>最后活跃</dt>
              <dd>{device.last_seen_at ? formatDateTime(device.last_seen_at) : '—'}</dd>
            </div>
          </dl>
        </div>

        <div className="card">
          <h2 className="section-title">指令下发</h2>
          <p className="device-command-hint">选择预设指令并下发（当前先落库，硬件按需拉取）。</p>
          <div className="command-presets">
            {DEVICE_COMMANDS.map((preset) => (
              <Button
                key={preset.value}
                variant={command === preset.value ? 'primary' : 'secondary'}
                disabled={sending}
                onClick={() => setCommand(preset.value)}
              >
                {preset.label}
              </Button>
            ))}
          </div>
          <div className="command-issue-row">
            <input
              className="form-input"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="start_recording / stop_recording"
              aria-label="指令"
            />
            <Button loading={sending} disabled={!command} onClick={handleIssue}>
              下发
            </Button>
          </div>
          {lastCommand && (
            <div className="command-result">
              已下发指令：
              <strong>{COMMAND_LABELS[lastCommand.command] ?? lastCommand.command}</strong>
              <span className="badge badge-muted">{COMMAND_LABELS[lastCommand.status] ?? lastCommand.status}</span>
            </div>
          )}
        </div>
      </div>

      <Modal
        open={confirmUnbind}
        onClose={() => setConfirmUnbind(false)}
        title="确认解绑"
        footer={
          <>
            <Button variant="secondary" disabled={unbinding} onClick={() => setConfirmUnbind(false)}>
              取消
            </Button>
            <Button variant="danger" loading={unbinding} onClick={handleUnbind}>
              解绑
            </Button>
          </>
        }
      >
        <p>确定要解绑设备「{device.name}」吗？该操作不可撤销。</p>
      </Modal>
    </div>
  );
}

export default Component;
