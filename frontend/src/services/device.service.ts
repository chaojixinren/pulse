import {
  Device,
  DeviceCommand,
  CreateDeviceInput,
  CreateDeviceResult,
} from '@/types/device.types';
import { http } from './api';

export const deviceService = {
  create(data: CreateDeviceInput): Promise<CreateDeviceResult> {
    return http.post<CreateDeviceResult>('/devices', data);
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
