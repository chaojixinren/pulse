import { expect, test, type Page } from '@playwright/test';
import { login } from './helpers';

async function goToStats(page: Page): Promise<void> {
  await login(page);
  await page.getByRole('link', { name: /统计/ }).click();
  await page.waitForURL('**/reports/stats');
}

test('统计页展示汇总、趋势与身份占比', async ({ page }) => {
  await goToStats(page);
  await expect(page.getByRole('heading', { name: '统计' })).toBeVisible();

  const summary = page.locator('.summary-grid');
  await expect(summary.getByText('会话数')).toBeVisible();
  await expect(summary.getByText('42', { exact: true })).toBeVisible();
  await expect(summary.getByText('总时长')).toBeVisible();
  await expect(summary.getByText('15 小时 0 分钟')).toBeVisible();
  await expect(summary.getByText('身份数')).toBeVisible();
  await expect(summary.getByText('2', { exact: true })).toBeVisible();

  // 会话数趋势 + 时长趋势 + 身份占比
  await expect(page.getByRole('img', { name: /每日会话数趋势图/ })).toBeVisible();
  await expect(page.getByRole('img', { name: /每日时长趋势图/ })).toBeVisible();
  await expect(page.getByRole('img', { name: /身份占比图/ })).toBeVisible();
  await expect(page.getByText('产品经理', { exact: true })).toBeVisible();
  await expect(page.getByText('健身教练', { exact: true })).toBeVisible();
});

test('统计页日期区间校验：起始晚于结束时报错', async ({ page }) => {
  await goToStats(page);

  await page.getByLabel('起始日期').fill('2024-06-30');
  await page.getByLabel('结束日期').fill('2024-06-01');
  await page.getByRole('button', { name: '查询' }).click();

  await expect(page.getByText('起始日期不能晚于结束日期')).toBeVisible();
});
