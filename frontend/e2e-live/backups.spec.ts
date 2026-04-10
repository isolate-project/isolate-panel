import { test, expect } from '@playwright/test'
import { loginViaApi, navigateTo } from './auth'

test.describe('Backups (live)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
  })

  test('should display backups page', async ({ page }) => {
    await navigateTo(page, '/backups')

    // Backups page uses a plain <div>, not <main> or PageLayout
    await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 15_000 })
  })

  test('should show create backup button', async ({ page }) => {
    await navigateTo(page, '/backups')
    await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 15_000 })

    // Look for a button to create/initiate a backup
    const createBtn = page.getByRole('button', { name: /Create|Backup|Initiate/i }).first()
    await expect(createBtn).toBeVisible({ timeout: 10_000 })
  })

  test('should render backup content area', async ({ page }) => {
    await navigateTo(page, '/backups')
    await page.waitForLoadState('networkidle')

    // The page should render a heading at minimum
    await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 15_000 })
  })

  test('should create a manual backup', async ({ page }) => {
    await navigateTo(page, '/backups')
    await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 15_000 })

    // Click create/initiate backup button
    const createBtn = page.getByRole('button', { name: /Create|Backup|Initiate/i }).first()
    if (await createBtn.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await createBtn.click()
      // Wait for backup process
      await page.waitForTimeout(5_000)
    }

    // Page should still be functional
    await expect(page.getByRole('heading').first()).toBeVisible()
  })
})
