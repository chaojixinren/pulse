import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

// Node.js >= 22 暴露实验性的全局 localStorage / sessionStorage，其 API 不完整
// （`localStorage.clear is not a function`），且 vitest 的 jsdom 环境默认不会把
// jsdom 的 Storage 覆盖到全局，导致被测代码里的裸 `localStorage` 指向 Node 的残缺实现。
// 这里安装一个确定性的内存 Storage，保证单元/组件测试的存储行为可预期。
class MemoryStorage implements Storage {
  private store = new Map<string, string>();

  get length(): number {
    return this.store.size;
  }

  clear(): void {
    this.store.clear();
  }

  getItem(key: string): string | null {
    return this.store.has(key) ? (this.store.get(key) as string) : null;
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null;
  }

  removeItem(key: string): void {
    this.store.delete(key);
  }

  setItem(key: string, value: string): void {
    this.store.set(key, String(value));
  }
}

function installStorage(): void {
  const localStorage = new MemoryStorage();
  const sessionStorage = new MemoryStorage();
  for (const [key, value] of [
    ['localStorage', localStorage],
    ['sessionStorage', sessionStorage],
  ] as const) {
    try {
      Object.defineProperty(globalThis, key, {
        value,
        writable: true,
        configurable: true,
      });
    } catch {
      // 兜底：直接赋值（某些环境下 defineProperty 可能失败）。
      (globalThis as unknown as Record<string, unknown>)[key] = value;
    }
  }
}

installStorage();

// 每个测试用例结束后卸载组件、清理存储，避免跨用例污染。
afterEach(() => {
  cleanup();
  try {
    globalThis.localStorage.clear();
    globalThis.sessionStorage.clear();
  } catch {
    // 忽略清理失败，不影响测试结果。
  }
});
