import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { useToast } from '@/components/common/Toast';
import { deviceService } from '@/services/device.service';
import type { BindDeviceResult } from '@/types/device.types';
import { copyText } from '@/utils/clipboard';

interface BindForm {
  device_id: string;
  name: string;
  bind_code: string;
}

const emptyForm: BindForm = { device_id: '', name: '', bind_code: '' };

export function Component() {
  const navigate = useNavigate();
  const toast = useToast();

  const [form, setForm] = useState<BindForm>(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [result, setResult] = useState<BindDeviceResult | null>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!form.device_id.trim()) {
      setFormError('请填写设备 ID');
      return;
    }
    if (!form.bind_code.trim()) {
      setFormError('请填写绑定码');
      return;
    }

    setSubmitting(true);
    try {
      const res = await deviceService.bind({
        device_id: form.device_id.trim(),
        name: form.name.trim() || undefined,
        bind_code: form.bind_code.trim(),
      });
      setResult(res);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : '绑定失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCopyToken = async () => {
    if (!result) return;
    const ok = await copyText(result.device_token);
    toast[ok ? 'success' : 'error'](ok ? '设备 Token 已复制' : '复制失败，请手动复制');
  };

  if (result) {
    return (
      <div className="page">
        <div className="page-header">
          <h1 className="page-title">绑定成功</h1>
        </div>
        <div className="card">
          <div className="bind-success">
            <div className="bind-success-icon">🎉</div>
            <p>设备「{result.device.name}」已绑定成功。</p>
            <p className="bind-token-hint">
              设备 Token 仅在本次展示一次，请复制并妥善保存，用于硬件侧鉴权。
            </p>
            <div className="bind-token">
              <code className="bind-token-value">{result.device_token}</code>
              <Button variant="secondary" onClick={handleCopyToken}>
                复制 Token
              </Button>
            </div>
            <div className="bind-success-actions">
              <Button variant="secondary" onClick={() => navigate('/devices')}>
                返回设备列表
              </Button>
              <Button onClick={() => navigate(`/devices/${result.device.id}`)}>
                查看设备详情
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">绑定设备</h1>
          <p className="page-subtitle">输入硬件侧设备 ID 与一次性绑定码完成绑定。</p>
        </div>
        <Button variant="secondary" onClick={() => navigate('/devices')}>
          返回设备列表
        </Button>
      </div>

      <div className="card">
        <form onSubmit={handleSubmit} noValidate>
          <div className="form-field">
            <label className="form-label" htmlFor="bind-device-id">
              设备 ID *
            </label>
            <input
              id="bind-device-id"
              className="form-input"
              value={form.device_id}
              onChange={(e) => setForm({ ...form, device_id: e.target.value })}
              placeholder="硬件侧唯一标识"
            />
          </div>
          <div className="form-field">
            <label className="form-label" htmlFor="bind-name">
              设备名称
            </label>
            <input
              id="bind-name"
              className="form-input"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="可选，默认「我的设备」"
            />
          </div>
          <div className="form-field">
            <label className="form-label" htmlFor="bind-code">
              绑定码 *
            </label>
            <input
              id="bind-code"
              className="form-input"
              value={form.bind_code}
              onChange={(e) => setForm({ ...form, bind_code: e.target.value })}
              placeholder="在设备列表页生成"
            />
          </div>
          {formError && <div className="form-error">{formError}</div>}
          <div className="bind-form-actions">
            <Button type="submit" loading={submitting}>
              绑定
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default Component;
