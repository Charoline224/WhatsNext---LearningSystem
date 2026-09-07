import { test, expect } from '@playwright/test'

test('logs in and creates an exam learning space', async ({ page }) => {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: '登录 WhatsNext' })).toBeVisible()
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page.getByRole('heading', { name: '你的学习空间' })).toBeVisible()

  await page.getByRole('button', { name: '创建学习空间' }).click()
  await page.getByLabel('学习空间名称').fill('操作系统期末复习')
  await page.getByLabel('我的目标').fill('期末考试达到 85 分')
  await page.getByLabel('考试日期').fill('2027-01-10')
  await page.getByRole('button', { name: '创建并继续' }).click()
  await expect(page.getByRole('heading', { name: '操作系统期末复习' })).toBeVisible()
})
