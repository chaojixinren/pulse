import { expect, test, type Page } from '@playwright/test';
import { buildDevice, login } from './helpers';

async function goToDevices(page: Page): Promise<void> {
  await page.getByRole('link', { name: /设备管理/ }).click();
  await page.waitForURL('**/devices');
}

test('设备列表空态展示绑定入口', async ({ page }) => {
  await login(page);
  await goToDevices(page);

  await expect(page.getByRole('heading', { name: '设备管理' })).toBeVisible();
  await expect(page.getByText('还没有设备')).toBeVisible();
  await expect(page.getByRole('button', { name: '去绑定设备' })).toBeVisible();
  await expect(page.getByRole('button', { name: '绑定设备', exact: true })).toBeVisible();
});

test('绑定设备全流程：填写设备 ID → 创建成功 → 抄录 token → 列表出现设备', async ({ page }) => {
  await login(page);
  await goToDevices(page);

  // 进入绑定页，填写设备 ID + 设备名称
  await page.getByRole('button', { name: '绑定设备', exact: true }).click();
  await page.waitForURL('**/devices/bind');
  await page.getByLabel('设备 ID *').fill('pulse-001');
  await page.getByLabel('设备名称').fill('办公麦克风');
  await page.getByRole('button', { name: '创建并获取 token', exact: true }).click();

  // 创建成功后展示一次性 token
  await expect(page.getByRole('heading', { name: '设备已创建' })).toBeVisible();
  await expect(page.getByTestId('device-token')).toBeVisible();

  // 抄录完成后进入设备详情
  await page.getByRole('button', { name: '我已抄录，完成' }).click();
  await page.waitForURL(/\/devices\/d-/);
  await expect(page.getByRole('heading', { name: '办公麦克风' })).toBeVisible();

  // 返回列表，设备已出现在列表中
  await page.getByRole('button', { name: '返回列表' }).click();
  await page.waitForURL('**/devices');
  await expect(page.getByText('办公麦克风')).toBeVisible();
});

test('设备已被占用时展示后端错误', async ({ page }) => {
  await login(page);
  await goToDevices(page);

  await page.getByRole('button', { name: '绑定设备', exact: true }).click();
  await page.waitForURL('**/devices/bind');
  await page.getByLabel('设备 ID *').fill('taken');
  await page.getByRole('button', { name: '创建并获取 token', exact: true }).click();

  await expect(page.getByText('该设备已被其他用户绑定')).toBeVisible();
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
