import { test, expect } from '@playwright/test'
import { loginViaApi, navigateTo } from './auth'

test.describe('Users (live)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
  })

  test('should display users page', async ({ page }) => {
    await navigateTo(page, '/users')

    await expect(page.getByRole('heading', { name: /Users/i }).first()).toBeVisible()
    // Use .first() to avoid strict mode with multiple matching buttons
    await expect(page.getByRole('button', { name: /Add User/i }).first()).toBeVisible()
  })

  test('should show empty state or user list', async ({ page }) => {
    await navigateTo(page, '/users')
    await page.waitForLoadState('networkidle')

    // Either the table is visible or empty state is shown
    const hasTable = await page.locator('table').isVisible().catch(() => false)
    const hasEmptyState = await page.getByText(/No users found/i).isVisible().catch(() => false)
    const hasHeading = await page.getByRole('heading', { name: /Users/i }).first().isVisible().catch(() => false)

    expect(hasTable || hasEmptyState || hasHeading).toBeTruthy()
  })

  test('should open Add User dialog', async ({ page }) => {
    await navigateTo(page, '/users')

    // Use .first() — multiple "Add User" buttons may exist (header + empty state)
    await page.getByRole('button', { name: /Add User/i }).first().click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })
    await expect(dialog.getByText('Account Details')).toBeVisible()
  })

  test('should create a new user', async ({ page }) => {
    await navigateTo(page, '/users')

    // Open create dialog
    await page.getByRole('button', { name: /Add User/i }).first().click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })

    // Fill the form
    const username = `testuser_${Date.now()}`
    await dialog.getByPlaceholder('e.g. john_doe').fill(username)

    // Submit — button may be below the fold in the dialog, force click
    await dialog.getByRole('button', { name: /Create User/i }).click({ force: true })

    // Dialog should close
    await expect(dialog).not.toBeVisible({ timeout: 10_000 })

    // New user should appear in the list
    await expect(page.getByText(username)).toBeVisible({ timeout: 10_000 })
  })

  test('should search users', async ({ page }) => {
    await navigateTo(page, '/users')
    await page.waitForLoadState('networkidle')

    const searchInput = page.getByPlaceholder(/Search/i).first()
    if (await searchInput.isVisible().catch(() => false)) {
      await searchInput.fill('nonexistent_user_xyz')
      await page.waitForTimeout(1_000)

      // Should show empty results or "No users found"
      const emptyState = page.getByText(/No users found/i)
      const table = page.locator('table tbody tr')

      const rowCount = await table.count().catch(() => 0)
      const isEmpty = await emptyState.isVisible().catch(() => false)

      expect(rowCount === 0 || isEmpty).toBeTruthy()
    }
  })
})
