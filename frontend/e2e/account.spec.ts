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

test('账户设置页渲染 ASR/AI 配置并保存', async ({ page }) => {
  await goToAccount(page);

  // 两个配置区块都成功加载（说明 GET /account/asr 与 /account/ai 正常返回）
  await expect(page.getByRole('heading', { name: '语音转写（ASR）' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'AI 分析' })).toBeVisible();

  // 保存 ASR 配置：填写模型后提交
  const asrForm = page.getByTestId('asr-form');
  await asrForm.getByLabel('模型 Model').fill('step-asr');
  await asrForm.getByRole('button', { name: '保存 ASR 配置' }).click();
  await expect(page.getByText('ASR 配置已保存')).toBeVisible();

  // 保存 AI 配置：填写模型后提交
  const aiForm = page.getByTestId('ai-form');
  await aiForm.getByLabel('模型 Model').fill('gpt-4o-mini');
  await aiForm.getByRole('button', { name: '保存 AI 配置' }).click();
  await expect(page.getByText('AI 配置已保存')).toBeVisible();
});

test('ASR 密钥保存后脱敏回显并可清除', async ({ page }) => {
  await goToAccount(page);
  const asrForm = page.getByTestId('asr-form');

  // 初始未配置密钥，不显示清除按钮
  await expect(asrForm.getByRole('button', { name: '清除密钥' })).toHaveCount(0);

  // 输入密钥并保存
  await asrForm.getByLabel('API Key').fill('sk-12345678');
  await asrForm.getByRole('button', { name: '保存 ASR 配置' }).click();
  await expect(page.getByText('ASR 配置已保存')).toBeVisible();

  // 掩码回显（输入框清空，占位符含尾部 4 位）且出现清除按钮
  await expect(asrForm.getByLabel('API Key')).toHaveAttribute('placeholder', /已配置（\*\*\*\*5678）/);
  const clearBtn = asrForm.getByRole('button', { name: '清除密钥' });
  await expect(clearBtn).toBeVisible();

  // 清除密钥后掩码与清除按钮一并消失
  await clearBtn.click();
  await expect(page.getByText('已清除密钥')).toBeVisible();
  await expect(asrForm.getByRole('button', { name: '清除密钥' })).toHaveCount(0);
});
