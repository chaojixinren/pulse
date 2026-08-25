import { expect, test } from '@playwright/test';
import { login } from './helpers';

test('身份列表展示默认标识', async ({ page }) => {
  await login(page);
  await expect(page.getByText('产品经理')).toBeVisible();
  await expect(page.getByText('健身教练')).toBeVisible();
  await expect(page.getByText('默认', { exact: true })).toBeVisible();
});

test('创建身份后立即出现在列表', async ({ page }) => {
  await login(page);
  await page.getByRole('button', { name: '+ 创建身份' }).click();
  await page.getByPlaceholder('例如：产品经理、健身教练').fill('新身份');
  await page.getByRole('button', { name: '保存' }).click();

  await expect(page.getByText('身份已创建')).toBeVisible();
  await expect(page.getByText('新身份')).toBeVisible();
});

test('编辑身份字段生效', async ({ page }) => {
  await login(page);
  const card = page.locator('.identity-card', { hasText: '健身教练' });
  await card.getByRole('button', { name: '编辑' }).click();

  const nameInput = page.getByPlaceholder('例如：产品经理、健身教练');
  await nameInput.fill('资深健身教练');
  await page.getByRole('button', { name: '保存' }).click();

  await expect(page.getByText('身份已更新')).toBeVisible();
  await expect(page.getByText('资深健身教练')).toBeVisible();
});

test('删除身份需二次确认', async ({ page }) => {
  await login(page);
  const card = page.locator('.identity-card', { hasText: '健身教练' });
  await card.getByRole('button', { name: '删除' }).click();

  const dialog = page.getByRole('dialog');
  await expect(dialog.getByText(/确定要删除身份「健身教练」吗/)).toBeVisible();
  await dialog.getByRole('button', { name: '删除' }).click();

  await expect(page.getByText('身份已删除')).toBeVisible();
  await expect(page.getByText('健身教练')).toHaveCount(0);
});

test('设为默认后显示默认标识', async ({ page }) => {
  await login(page);
  const card = page.locator('.identity-card', { hasText: '健身教练' });
  await card.getByRole('button', { name: '设为默认' }).click();

  await expect(page.getByText('已设为默认身份')).toBeVisible();
  // 健身教练卡片应显示「默认」，产品经理卡片不再显示
  await expect(card.getByText('默认', { exact: true })).toBeVisible();
  await expect(
    page.locator('.identity-card', { hasText: '产品经理' }).getByText('默认', { exact: true }),
  ).toHaveCount(0);
});
