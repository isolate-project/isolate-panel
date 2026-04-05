import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis } from './fixtures'

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  test('should display dashboard page with stat cards', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    // Dashboard renders stat cards — check for translated labels
    await expect(page.getByText(/Total Users|Users/i).first()).toBeVisible({ timeout: 5000 })
  })

  test('should show system resource info', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    await expect(page.getByText(/CPU|Memory|RAM/i).first()).toBeVisible({ timeout: 5000 })
  })

  test('should display sidebar navigation', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

    await expect(page.locator('aside').getByText('Dashboard')).toBeVisible()
    await expect(page.locator('aside').getByText('Users')).toBeVisible()
    await expect(page.locator('aside').getByText('Cores')).toBeVisible()
    await expect(page.locator('aside').getByText('Inbounds')).toBeVisible()
    await expect(page.locator('aside').getByText('Backups')).toBeVisible()
    await expect(page.locator('aside').getByText('Settings')).toBeVisible()
  })
})
