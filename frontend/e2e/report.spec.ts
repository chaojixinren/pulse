import { expect, test, type Page } from '@playwright/test';
import { login } from './helpers';

async function goToReport(page: Page): Promise<void> {
  await login(page);
  await page.getByRole('link', { name: /日报/ }).click();
  await page.waitForURL('**/reports/daily');
}

test('日报展示汇总、身份拆分与 AI 提取', async ({ page }) => {
  await goToReport(page);
  await expect(page.getByRole('heading', { name: '日报' })).toBeVisible();

  // 汇总卡片
  await expect(page.getByText('会话数')).toBeVisible();
  await expect(page.getByText('2', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('总时长')).toBeVisible();
  await expect(page.getByText('1 小时 1 分钟', { exact: true })).toBeVisible();

  // 按身份拆分
  await expect(page.getByText('产品经理')).toBeVisible();
  await expect(page.getByText('健身教练')).toBeVisible();

  // AI 提取的待办与笔记
  await expect(page.getByText('整理产品需求文档')).toBeVisible();
  await expect(page.getByText('讨论了产品路线图')).toBeVisible();
});

test('切换日期加载对应日报', async ({ page }) => {
  await goToReport(page);
  const dateInput = page.locator('input[type="date"]');
  const before = await dateInput.inputValue();

  await page.getByRole('button', { name: '← 前一天' }).click();
  await expect(dateInput).not.toHaveValue(before);

  await page.getByRole('button', { name: '后一天 →' }).click();
  await expect(dateInput).toHaveValue(before);
});
