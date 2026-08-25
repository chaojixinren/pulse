import { defineConfig, devices } from '@playwright/test';

// Playwright E2E 配置：
// - 用 Vite 开发服务器作为被测目标（同源代理 /api/v1，避免跨域）；
// - API 在测试内通过 page.route 模拟，因此不依赖真实后端。
//
// 浏览器选择：
// - CI（无 PLAYWRIGHT_CHANNEL）使用 Playwright 下载的 chromium（`npx playwright install --with-deps chromium`）；
// - 本地可 `PLAYWRIGHT_CHANNEL=chrome npm run test:e2e` 复用系统 Chrome，避免下载浏览器。
const channel = process.env.PLAYWRIGHT_CHANNEL;

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], ...(channel ? { channel } : {}) },
    },
  ],
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1 --port 4173',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    env: { VITE_API_BASE_URL: '/api/v1' },
    timeout: 120_000,
  },
});
