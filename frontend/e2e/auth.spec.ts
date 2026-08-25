import { expect, test } from '@playwright/test';
import { login, mockApi } from './helpers';

test('未登录访问受保护页面重定向到登录页', async ({ page }) => {
  await mockApi(page);
  await page.goto('/identity');
  await expect(page).toHaveURL(/\/auth\/login/);
  await expect(page.getByRole('heading', { name: 'Pulse' })).toBeVisible();
});

test('登录成功后进入身份管理', async ({ page }) => {
  await login(page);
  await expect(page).toHaveURL(/\/identity/);
  await expect(page.getByRole('heading', { name: '身份管理' })).toBeVisible();
  await expect(page.getByText('产品经理')).toBeVisible();
});

test('密码错误给出明确提示', async ({ page }) => {
  await mockApi(page);
  await page.goto('/auth/login');
  await page.getByLabel('邮箱').fill('user@example.com');
  await page.getByLabel('密码', { exact: true }).fill('wrong-password');
  await page.getByRole('button', { name: '登录' }).click();
  await expect(page.getByText('邮箱或密码错误')).toBeVisible();
});

test('刷新页面会话保持', async ({ page }) => {
  await login(page);
  await page.reload();
  await expect(page.getByRole('heading', { name: '身份管理' })).toBeVisible();
});

test('登出后跳转登录页', async ({ page }) => {
  await login(page);
  await page.getByRole('button', { name: '退出登录' }).click();
  await expect(page).toHaveURL(/\/auth\/login/);
});

test('注册后自动登录进入身份管理', async ({ page }) => {
  await mockApi(page);
  await page.goto('/auth/register');
  await page.getByLabel('姓名').fill('新用户');
  await page.getByLabel('邮箱').fill('new@example.com');
  await page.getByLabel('密码', { exact: true }).fill('12345678');
  await page.getByLabel('确认密码', { exact: true }).fill('12345678');
  await page.getByRole('button', { name: '注册并登录' }).click();
  await page.waitForURL('**/identity');
  await expect(page.getByRole('heading', { name: '身份管理' })).toBeVisible();
});
