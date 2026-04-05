import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis } from './fixtures'

test.describe('Inbounds Management', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  test('should display inbounds list', async ({ page }) => {
    await page.goto('/inbounds')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    // Should show port numbers or protocol from mock data
    await expect(page.getByText('443').first()).toBeVisible({ timeout: 5000 })
  })

  test('should show Add Inbound button', async ({ page }) => {
    await page.goto('/inbounds')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    const addButton = page.locator('button, a', { hasText: /Add|Create|Создать/i })
    await expect(addButton.first()).toBeVisible({ timeout: 5000 })
  })

  test('should navigate to inbounds page via sidebar', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

    await page.locator('aside').getByText('Inbounds').click()
    await page.waitForURL('**/inbounds')

    await expect(page.getByText('443').first()).toBeVisible({ timeout: 5000 })
  })
})
