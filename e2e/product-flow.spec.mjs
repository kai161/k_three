import { test, expect } from '@playwright/test';

async function completeDay(page, day, ritual = '揉掉它') {
  await page.getByLabel('今天，最消耗你的一件事是什么？').fill(`产品体验第 ${day} 天：今天的工作还留在脑子里`);
  await page.getByRole('button', { name: '为今天收个尾' }).click();
  await expect(page.getByText('今晚，想对你说')).toBeVisible();
  await page.getByRole('button', { name: new RegExp(ritual) }).click();
  await expect(page.getByText('此刻，你更接近')).toBeVisible();
  await page.getByRole('button', { name: '比较平静' }).click();
}

test('public beta user completes seven days and leaves feedback without activation', async ({ page }, testInfo) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '把今天， 轻轻放下。' })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath('01-welcome.png'), fullPage: true, animations: 'disabled' });

  const reflection = page.getByLabel('今天，最消耗你的一件事是什么？');
  await reflection.fill('这句草稿应该在刷新后恢复');
  await page.reload();
  await expect(reflection).toHaveValue('这句草稿应该在刷新后恢复');

  await page.getByRole('button', { name: '为今天收个尾' }).click();
  await expect(page.getByText('今晚，想对你说')).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath('02-reply.png'), fullPage: true, animations: 'disabled' });
  await page.getByRole('button', { name: /揉掉它/ }).click();
  await expect(page.getByText('此刻，你更接近')).toBeVisible();
  await page.getByRole('button', { name: '比较平静' }).click();
  await expect(page.getByRole('heading', { name: '今天到这里， 就很好。' })).toBeVisible();
  await page.getByRole('button', { name: '看看晚安计划' }).click();
  await expect(reflection).toHaveValue('');
  await expect(page.locator('#progress-label')).toHaveText('已留下 1 个夜晚');

  await completeDay(page, 2, '封存它');
  await expect(page.locator('#feedback-day2-form')).toBeVisible();
  await page.getByLabel('这两天的体验，哪一刻对你有一点帮助？').fill('封存烦恼的时候轻松了一点');
  await page.locator('#feedback-day2-form').getByRole('button', { name: '提交反馈' }).click();
  await expect(page.locator('#feedback-day2-form').getByRole('button', { name: '已收到，谢谢你' })).toBeVisible();
  await page.getByRole('button', { name: '看看晚安计划' }).click();
  await expect(reflection).toBeVisible();
  await expect(page.locator('#screen-locked')).toBeHidden();
  await page.screenshot({ path: testInfo.outputPath('03-public-day3.png'), fullPage: true, animations: 'disabled' });

  for (let day = 3; day <= 7; day += 1) {
    await completeDay(page, day);
    if (day < 7) await page.getByRole('button', { name: '看看晚安计划' }).click();
  }
  await expect(page.getByRole('heading', { name: '这七天， 你都在照顾自己。' })).toBeVisible();
  await expect(page.locator('.review-entry')).toHaveCount(7);
  await expect(page.locator('#feedback-day7-form')).toBeVisible();
  await page.getByLabel('这七天里，哪一刻对你有一点帮助？').fill('每天两分钟，让我愿意停下来');
  await page.locator('#feedback-day7-form').getByRole('button', { name: '提交反馈' }).click();
  await expect(page.locator('#feedback-day7-form').getByRole('button', { name: '已收到，谢谢你' })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath('04-review.png'), fullPage: true, animations: 'disabled' });
});

test('high-risk response returns to a safe writing path', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('今天，最消耗你的一件事是什么？').fill('我不想活了');
  await page.getByRole('button', { name: '为今天收个尾' }).click();
  await expect(page.getByRole('heading', { name: '这一刻，不必独自承担' })).toBeVisible();
  await page.getByRole('button', { name: '返回' }).click();
  await expect(page.getByLabel('今天，最消耗你的一件事是什么？')).toBeVisible();
});
