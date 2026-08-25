import { expect, test, type Page } from '@playwright/test';
import { login } from './helpers';

async function goToTimeline(page: Page): Promise<void> {
  await login(page);
  await page.getByRole('link', { name: /时间线/ }).click();
  await page.waitForURL('**/timeline');
}

test('时间线列表展示转写文本与状态', async ({ page }) => {
  await goToTimeline(page);
  await expect(page.getByText('第 1 条会话转写文本')).toBeVisible();
  await expect(page.locator('.badge-status-completed').first()).toBeVisible();
  await expect(page.getByText(/共 25 条/)).toBeVisible();
});

test('按状态过滤生效', async ({ page }) => {
  await goToTimeline(page);
  await page.getByLabel('状态').selectOption('completed');
  // 过滤后仅剩 completed 的 10 条
  await expect(page.getByText('第 1 条会话转写文本')).toBeVisible();
  await expect(page.getByText('第 11 条会话转写文本')).toHaveCount(0);
  await expect(page.getByText(/共 10 条/)).toBeVisible();
});

test('分页正常', async ({ page }) => {
  await goToTimeline(page);
  const next = page.getByRole('button', { name: '下一页' });
  await expect(next).toBeEnabled();
  await next.click();
  await expect(page.getByText('第 21 条会话转写文本')).toBeVisible();
  await expect(page.getByRole('button', { name: '上一页' })).toBeEnabled();
});

test('空态正常', async ({ page }) => {
  await goToTimeline(page);
  // 选择「失败」状态（mock 无 failed 数据）
  await page.getByLabel('状态').selectOption('failed');
  await expect(page.getByText('暂无时间线记录')).toBeVisible();
});
