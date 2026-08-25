import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { downloadBlob } from './download';

const urlAny = URL as unknown as {
  createObjectURL: (b: Blob) => string;
  revokeObjectURL: (u: string) => void;
};

const originalCreate = urlAny.createObjectURL;
const originalRevoke = urlAny.revokeObjectURL;

describe('downloadBlob', () => {
  beforeEach(() => {
    urlAny.createObjectURL = vi.fn(() => 'blob:mock');
    urlAny.revokeObjectURL = vi.fn();
  });

  afterEach(() => {
    urlAny.createObjectURL = originalCreate;
    urlAny.revokeObjectURL = originalRevoke;
    vi.restoreAllMocks();
  });

  it('创建对象 URL、触发下载并回收 URL', () => {
    const click = vi.fn();
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(click);

    const blob = new Blob(['{"a":1}'], { type: 'application/json' });
    downloadBlob(blob, 'pulse-export.json');

    expect(urlAny.createObjectURL).toHaveBeenCalledWith(blob);
    expect(click).toHaveBeenCalledTimes(1);
    expect(urlAny.revokeObjectURL).toHaveBeenCalledWith('blob:mock');
  });

  it('使用传入的文件名与 blob URL', () => {
    const anchor = document.createElement('a');
    vi.spyOn(document, 'createElement').mockReturnValue(anchor);
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined);

    downloadBlob(new Blob(['x']), 'pulse-export-2024-01-01.json');

    expect(anchor.download).toBe('pulse-export-2024-01-01.json');
    expect(anchor.href).toBe('blob:mock');
  });
});
