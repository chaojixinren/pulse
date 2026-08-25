import { readFileSync } from 'node:fs';
import { expect, test, type Page } from '@playwright/test';
import { login } from './helpers';

async function goToAccount(page: Page): Promise<void> {
  await login(page);
  await page.getByRole('link', { name: /账户设置/ }).click();
  await page.waitForURL('**/account');
}

test('导出个人数据并下载 JSON 文件', async ({ page }) => {
  await goToAccount(page);

  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', { name: '导出数据' }).click();
  const download = await downloadPromise;

  expect(download.suggestedFilename()).toMatch(/^pulse-export-\d{4}-\d{2}-\d{2}\.json$/);
  await expect(page.getByText('数据已导出')).toBeVisible();

  // 校验下载内容包含用户与身份数据
  const path = await download.path();
  const content = readFileSync(path, 'utf8');
  expect(content).toContain('user@example.com');
  expect(content).toContain('产品经理');
});

test('注销账户需邮箱二次确认，注销后跳转注册页', async ({ page }) => {
  await goToAccount(page);

  await page.getByRole('button', { name: '注销账户' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog.getByText(/注销是不可逆操作/)).toBeVisible();

  // 邮箱不匹配时报错，不触发注销
  await dialog.getByLabel('邮箱').fill('wrong@example.com');
  await dialog.getByRole('button', { name: '确认注销' }).click();
  await expect(page.getByText('请输入与账户一致的邮箱以确认注销')).toBeVisible();

  // 输入正确邮箱后注销，跳转注册页
  await dialog.getByLabel('邮箱').fill('user@example.com');
  await dialog.getByRole('button', { name: '确认注销' }).click();
  await page.waitForURL('**/auth/register');
});
