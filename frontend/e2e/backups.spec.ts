import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis, mockBackups } from './fixtures'

test.describe('Backups Management', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  test('should display backups list', async ({ page }) => {
    await page.goto('/backups')

    // Backup filenames from mock data
    await expect(page.getByText(mockBackups[0].filename)).toBeVisible({ timeout: 10000 })
    await expect(page.getByText(mockBackups[1].filename)).toBeVisible()
  })

  test('should show create backup button', async ({ page }) => {
    await page.goto('/backups')

    await expect(page.getByText('Initiate Secure Backup')).toBeVisible({ timeout: 10000 })
  })

  test('should navigate to backups page via sidebar', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

    await page.locator('aside').getByText('Backups').click()
    await page.waitForURL('**/backups')

    await expect(page.getByText(mockBackups[0].filename)).toBeVisible({ timeout: 10000 })
  })
})
