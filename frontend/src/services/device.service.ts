import {
  BindDeviceInput,
  BindDeviceResult,
  Device,
  DeviceBindCode,
  DeviceCommand,
} from '@/types/device.types';
import { http } from './api';

export const deviceService = {
  generateBindCode(): Promise<DeviceBindCode> {
    return http.post<DeviceBindCode>('/devices/bind-code');
  },
  bind(data: BindDeviceInput): Promise<BindDeviceResult> {
    return http.post<BindDeviceResult>('/devices/bind', data);
  },
  list(): Promise<Device[]> {
    return http.get<Device[]>('/devices');
  },
  get(id: string): Promise<Device> {
    return http.get<Device>(`/devices/${id}`);
  },
  unbind(id: string): Promise<void> {
    return http.delete(`/devices/${id}`);
  },
  issueCommand(id: string, command: string): Promise<DeviceCommand> {
    return http.post<DeviceCommand>(`/devices/${id}/command`, { command });
  },
};
