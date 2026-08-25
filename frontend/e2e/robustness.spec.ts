import { expect, test } from '@playwright/test';
import { login } from './helpers';

test('深色模式切换并持久化', async ({ page }) => {
  await login(page);
  const html = page.locator('html');
  const before = await html.getAttribute('data-theme');

  await page.getByTitle('切换主题').click();
  const after = await html.getAttribute('data-theme');
  expect(after).not.toBe(before);
  expect(after === 'light' || after === 'dark').toBe(true);

  // 持久化到 localStorage
  const stored = await page.evaluate(() => localStorage.getItem('pulse_theme'));
  expect(stored === 'light' || stored === 'dark').toBe(true);

  // 刷新后主题保持
  await page.reload();
  await expect(html).toHaveAttribute('data-theme', after ?? '');
});

test('access token 失效自动续期并重放请求', async ({ page }) => {
  let identityCalls = 0;
  let refreshCalls = 0;
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname.replace(/^\/api\/v1/, '');
    const method = route.request().method();
    const respond = (data: unknown, code = 0, message = 'ok', status = 200) =>
      route.fulfill({ status, json: { code, message, data } });

    if (method === 'POST' && path === '/auth/login') {
      return respond({ access_token: 'expired', refresh_token: 'refresh-1' });
    }
    if (method === 'GET' && path === '/auth/me') {
      return respond({ id: 'u1', email: 'user@example.com', name: '测试用户' });
    }
    if (method === 'POST' && path === '/auth/refresh') {
      refreshCalls += 1;
      return respond({ access_token: 'fresh', refresh_token: 'refresh-2' });
    }
    if (method === 'GET' && path === '/identities') {
      identityCalls += 1;
      // 首次请求 401，触发续期后重放
      if (identityCalls === 1) {
        return respond(null, 401, 'token expired', 401);
      }
      return respond([
        {
          id: 'i1',
          user_id: 'u1',
          name: '产品经理',
          color: '#3b82f6',
          icon: '🙂',
          is_default: true,
          created_at: '',
          updated_at: '',
        },
      ]);
    }
    return respond(null, 404, 'not found');
  });

  await page.goto('/auth/login');
  await page.getByLabel('邮箱').fill('user@example.com');
  await page.getByLabel('密码').fill('password');
  await page.getByRole('button', { name: '登录' }).click();

  // 身份列表首次请求 401，应自动续期并重放成功（StrictMode 下可能多次拉取，仅要求 >= 2）
  await page.waitForURL('**/identity');
  await expect(page.getByText('产品经理')).toBeVisible();
  expect(refreshCalls).toBe(1);
  expect(identityCalls).toBeGreaterThanOrEqual(2);
});
