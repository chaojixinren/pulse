import { useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { useToast } from '@/components/common/Toast';
import { deviceService } from '@/services/device.service';
import { copyText } from '@/utils/clipboard';
import type { CreateDeviceResult } from '@/types/device.types';

interface BindForm {
  device_id: string;
  device_name: string;
}

const emptyForm: BindForm = { device_id: '', device_name: '' };

// 将 token 按 4 字符分组展示，便于手抄核对。
function groupToken(token: string): string {
  return token.replace(/(.{4})/g, '$1 ').trim();
}

export function Component() {
  const navigate = useNavigate();
  const toast = useToast();

  const [form, setForm] = useState<BindForm>(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [result, setResult] = useState<CreateDeviceResult | null>(null);

  // 令牌尚未抄录时刷新/关闭页面，给出离开提示。
  useEffect(() => {
    if (!result) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = '';
    };
    window.addEventListener('beforeunload', onBeforeUnload);
    return () => window.removeEventListener('beforeunload', onBeforeUnload);
  }, [result]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);

    const deviceId = form.device_id.trim();
    if (!deviceId) {
      setFormError('请填写设备 ID');
      return;
    }

    setSubmitting(true);
    try {
      const res = await deviceService.create({
        device_id: deviceId,
        name: form.device_name.trim() || undefined,
      });
      setResult(res);
      toast.success('设备已创建，请抄录 token');
    } catch (err) {
      setFormError(err instanceof Error ? err.message : '创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCopy = async () => {
    if (!result) return;
    const ok = await copyText(result.device_token);
    if (ok) {
      toast.success('已复制 token');
    } else {
      toast.error('复制失败，请手动选择复制');
    }
  };

  const handleDone = () => {
    if (result) navigate(`/devices/${result.device.id}`);
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">绑定设备</h1>
          <p className="page-subtitle">
            创建后一次性返回设备 token，手抄到硬件 config.json 即可让硬件接入云端。
          </p>
        </div>
        <Button variant="secondary" onClick={() => navigate('/devices')}>
          返回设备列表
        </Button>
      </div>

      <div className="card">
        {result ? (
          <div className="bind-success">
            <h2>设备已创建</h2>
            <p className="bind-token-hint">
              token 仅显示这一次，请立即抄录到硬件 config.json（cloud.auth_token），关闭后无法再次查看。
            </p>
            <div className="bind-token">
              <code className="bind-token-value" data-testid="device-token">
                {groupToken(result.device_token)}
              </code>
              <Button variant="secondary" onClick={handleCopy}>
                复制 token
              </Button>
            </div>
            <p className="bind-token-hint">
              config.json 中 cloud.device_id 需设为「{result.device.device_id}」，cloud.auth_scheme 设为 Device。
            </p>
            <div className="bind-success-actions">
              <Button onClick={handleDone}>我已抄录，完成</Button>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} noValidate>
            <div className="form-field">
              <label className="form-label" htmlFor="device-id">
                设备 ID *
              </label>
              <input
                id="device-id"
                className="form-input"
                value={form.device_id}
                onChange={(e) => setForm({ ...form, device_id: e.target.value })}
                placeholder="与 config.json 中 cloud.device_id 一致，如 pulse-001"
              />
            </div>
            <div className="form-field">
              <label className="form-label" htmlFor="device-name">
                设备名称
              </label>
              <input
                id="device-name"
                className="form-input"
                value={form.device_name}
                onChange={(e) => setForm({ ...form, device_name: e.target.value })}
                placeholder="可选，默认「我的设备」"
              />
            </div>
            {formError && <div className="form-error">{formError}</div>}
            <div className="bind-form-actions">
              <Button type="submit" loading={submitting}>
                创建并获取 token
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

export default Component;
