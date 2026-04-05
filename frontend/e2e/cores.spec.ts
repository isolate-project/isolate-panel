import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis } from './fixtures'

test.describe('Cores Management', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  test('should display list of cores', async ({ page }) => {
    await page.goto('/cores')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    // Core names appear in headings
    await expect(page.getByRole('heading', { name: 'xray' })).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('heading', { name: 'singbox' })).toBeVisible()
  })

  test('should show running status for xray', async ({ page }) => {
    await page.goto('/cores')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    await expect(page.getByText(/running|Running/i).first()).toBeVisible({ timeout: 5000 })
  })

  test('should navigate to cores page via sidebar', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

    await page.locator('aside').getByText('Cores').click()
    await page.waitForURL('**/cores')

    await expect(page.getByRole('heading', { name: 'xray' })).toBeVisible({ timeout: 5000 })
  })
})
