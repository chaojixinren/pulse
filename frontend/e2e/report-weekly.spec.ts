import { expect, test, type Page } from '@playwright/test';
import { login } from './helpers';

async function goToWeekly(page: Page): Promise<void> {
  await login(page);
  await page.getByRole('link', { name: /周报/ }).click();
  await page.waitForURL('**/reports/weekly');
}

test('周报展示汇总、趋势、身份占比与待办', async ({ page }) => {
  await goToWeekly(page);
  await expect(page.getByRole('heading', { name: '周报' })).toBeVisible();

  // 汇总卡片（作用域限定在 summary-grid，避免与 SVG 柱状图数值文本冲突）
  const summary = page.locator('.summary-grid');
  await expect(summary.getByText('会话数')).toBeVisible();
  await expect(summary.getByText('12', { exact: true })).toBeVisible();
  await expect(summary.getByText('总时长')).toBeVisible();
  await expect(summary.getByText('4 小时 0 分钟')).toBeVisible();
  await expect(summary.getByText('完成承诺')).toBeVisible();
  await expect(summary.getByText('3', { exact: true })).toBeVisible();
  await expect(summary.getByText('身份数')).toBeVisible();
  await expect(summary.getByText('2', { exact: true })).toBeVisible();

  // 趋势图与身份占比图
  await expect(page.getByRole('img', { name: /每日会话数趋势图/ })).toBeVisible();
  await expect(page.getByRole('img', { name: /身份占比图/ })).toBeVisible();
  await expect(page.getByText('产品经理', { exact: true })).toBeVisible();
  await expect(page.getByText('健身教练', { exact: true })).toBeVisible();

  // Top 待办
  await expect(page.getByText('整理产品需求文档')).toBeVisible();
  await expect(page.getByText('安排下周评审')).toBeVisible();
});

test('周报可切换上一周/下一周', async ({ page }) => {
  await goToWeekly(page);
  const dateEl = page.locator('.report-date');
  const before = ((await dateEl.textContent()) ?? '').trim();

  await page.getByRole('button', { name: '← 上一周' }).click();
  await expect(dateEl).not.toHaveText(before);

  await page.getByRole('button', { name: '下一周 →' }).click();
  await expect(dateEl).toHaveText(before);
});
