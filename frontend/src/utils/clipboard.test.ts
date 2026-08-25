import { afterEach, describe, expect, it, vi } from 'vitest';
import { copyText } from './clipboard';

const originalClipboard = navigator.clipboard;
const originalExecCommand = document.execCommand;

afterEach(() => {
  Object.defineProperty(navigator, 'clipboard', {
    value: originalClipboard,
    configurable: true,
    writable: true,
  });
  document.execCommand = originalExecCommand;
});

describe('copyText', () => {
  it('优先使用 navigator.clipboard.writeText', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    });

    const ok = await copyText('hello');
    expect(writeText).toHaveBeenCalledWith('hello');
    expect(ok).toBe(true);
  });

  it('Clipboard API 失败时回退到 execCommand', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
      configurable: true,
    });
    const execCommand = vi.fn().mockReturnValue(true);
    document.execCommand = execCommand;

    const ok = await copyText('hello');
    expect(execCommand).toHaveBeenCalledWith('copy');
    expect(ok).toBe(true);
  });

  it('无 Clipboard API 时直接使用 execCommand', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: undefined,
      configurable: true,
    });
    const execCommand = vi.fn().mockReturnValue(true);
    document.execCommand = execCommand;

    const ok = await copyText('hello');
    expect(execCommand).toHaveBeenCalledWith('copy');
    expect(ok).toBe(true);
  });
});
