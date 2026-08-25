import { expect, test, type Page } from '@playwright/test';
import { buildDevice, login } from './helpers';

async function goToDevices(page: Page): Promise<void> {
  await page.getByRole('link', { name: /设备管理/ }).click();
  await page.waitForURL('**/devices');
}

test('设备列表空态展示生成绑定码与绑定入口', async ({ page }) => {
  await login(page);
  await goToDevices(page);

  await expect(page.getByRole('heading', { name: '设备管理' })).toBeVisible();
  await expect(page.getByText('还没有设备')).toBeVisible();
  await expect(page.getByRole('button', { name: '生成绑定码' })).toBeVisible();
  await expect(page.getByRole('button', { name: '去绑定设备' })).toBeVisible();
});

test('生成绑定码显示一次性绑定码与有效期', async ({ page }) => {
  await login(page);
  await goToDevices(page);

  await page.getByRole('button', { name: '生成绑定码' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog.locator('.bind-code-value')).toHaveText('BINDCODE1');
  await expect(dialog.getByText(/有效期剩余/)).toBeVisible();
  await expect(dialog.getByRole('button', { name: '复制绑定码' })).toBeVisible();
});

test('绑定设备全流程：生成码 → 绑定 → token 一次性展示 → 列表出现', async ({ page }) => {
  await login(page);
  await goToDevices(page);

  // 生成绑定码并读取其内容
  await page.getByRole('button', { name: '生成绑定码' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog.locator('.bind-code-value')).toBeVisible();
  const code = ((await dialog.locator('.bind-code-value').textContent()) ?? '').trim();
  expect(code).toBe('BINDCODE1');
  await dialog.locator('.modal-close').click();

  // 用绑定码 + 设备 ID 完成绑定
  await page.getByRole('button', { name: '绑定设备', exact: true }).click();
  await page.waitForURL('**/devices/bind');
  await page.getByLabel('设备 ID *').fill('HW-2024');
  await page.getByLabel('设备名称').fill('办公麦克风');
  await page.getByLabel('绑定码 *').fill(code);
  await page.getByRole('button', { name: '绑定', exact: true }).click();

  // 绑定成功：展示一次性 token 与设备信息
  await expect(page.getByRole('heading', { name: '绑定成功' })).toBeVisible();
  await expect(page.getByText('设备「办公麦克风」已绑定成功。')).toBeVisible();
  await expect(page.locator('.bind-token-value')).toContainText('tok-');
  await expect(page.getByText(/仅在本次展示一次/)).toBeVisible();

  // 返回列表，设备已出现在列表中
  await page.getByRole('button', { name: '返回设备列表' }).click();
  await page.waitForURL('**/devices');
  await expect(page.getByText('办公麦克风')).toBeVisible();
});

test('设备列表展示在线状态、电量与固件', async ({ page }) => {
  await login(page, {
    devices: [
      buildDevice({ id: 'd1', name: '客厅麦克风' }),
      buildDevice({
        id: 'd2',
        name: '书房麦克风',
        device_id: 'HW-002',
        last_seen_at: new Date(Date.now() - 3_600_000).toISOString(),
      }),
    ],
  });
  await goToDevices(page);

  const onlineCard = page.locator('.device-card', { hasText: '客厅麦克风' });
  await expect(onlineCard.getByText('在线')).toBeVisible();
  await expect(onlineCard.getByText('电量 80%')).toBeVisible();
  await expect(onlineCard.getByText('固件 1.2.3')).toBeVisible();
  await expect(onlineCard.getByText(/最后活跃/)).toBeVisible();

  const offlineCard = page.locator('.device-card', { hasText: '书房麦克风' });
  await expect(offlineCard.getByText('离线')).toBeVisible();
});

test('设备详情展示信息并可下发指令', async ({ page }) => {
  await login(page, { devices: [buildDevice({ id: 'd1', name: '客厅麦克风' })] });
  await goToDevices(page);

  await page.locator('.device-card').click();
  await page.waitForURL('**/devices/d1');

  // 设备信息
  await expect(page.getByText('设备 ID')).toBeVisible();
  await expect(page.getByText('HW-001')).toBeVisible();
  await expect(page.getByText('设备类型')).toBeVisible();
  await expect(page.getByText('pulse-mic')).toBeVisible();
  await expect(page.getByText('在线状态')).toBeVisible();
  await expect(page.getByText('在线', { exact: true })).toBeVisible();
  await expect(page.getByText('电量')).toBeVisible();
  await expect(page.getByText('80%')).toBeVisible();
  await expect(page.getByText('固件版本')).toBeVisible();
  await expect(page.getByText('1.2.3')).toBeVisible();

  // 指令下发：选择预设指令并下发
  await page.getByRole('button', { name: '开始录音' }).click();
  await page.getByRole('button', { name: '下发' }).click();

  const result = page.locator('.command-result');
  await expect(result).toBeVisible();
  await expect(result).toContainText('已下发指令');
  await expect(result.getByText('开始录音')).toBeVisible();
  await expect(result.getByText('待下发')).toBeVisible();
});

test('解绑设备需二次确认，解绑后列表更新', async ({ page }) => {
  await login(page, { devices: [buildDevice({ id: 'd1', name: '客厅麦克风' })] });
  await goToDevices(page);

  await page.locator('.device-card').click();
  await page.waitForURL('**/devices/d1');

  await page.getByRole('button', { name: '解绑设备' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog.getByText(/确定要解绑设备「客厅麦克风」吗/)).toBeVisible();
  await dialog.getByRole('button', { name: '解绑', exact: true }).click();

  await page.waitForURL('**/devices');
  await expect(page.getByText('还没有设备')).toBeVisible();
});
