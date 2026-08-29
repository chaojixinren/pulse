import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/common/Button';
import { Empty } from '@/components/common/Empty';
import { Loading } from '@/components/common/Loading';
import { DeviceCard } from '@/components/business/DeviceCard';
import { deviceService } from '@/services/device.service';
import type { Device } from '@/types/device.types';

export function Component() {
  const navigate = useNavigate();

  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">设备管理</h1>
          <p className="page-subtitle">绑定并管理你的 Pulse 硬件设备，查看在线状态与下发指令。</p>
        </div>
        <div className="page-header-actions">
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
          title="还没有设备"
          description="创建后一次性返回设备 token，手抄到硬件 config.json 即可完成接入。"
          action={<Button onClick={() => navigate('/devices/bind')}>去绑定设备</Button>}
        />
      ) : (
        <div className="device-grid">
          {devices.map((device) => (
            <DeviceCard key={device.id} device={device} onClick={() => navigate(`/devices/${device.id}`)} />
          ))}
        </div>
      )}
    </div>
  );
}

export default Component;
