import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis } from './fixtures'

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  test('should display settings page', async ({ page }) => {
    await page.goto('/settings')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    // Settings page should have settings-related content
    await expect(page.getByText(/Settings|Настройки/i).first()).toBeVisible({ timeout: 5000 })
  })

  test('should navigate to settings page via sidebar', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

    await page.locator('aside').getByText('Settings').click()
    await page.waitForURL('**/settings')

    await expect(page.getByText(/Settings|Настройки/i).first()).toBeVisible({ timeout: 5000 })
  })
})
