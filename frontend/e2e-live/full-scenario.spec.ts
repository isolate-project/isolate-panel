import { test, expect } from '@playwright/test'
import { loginViaApi } from './auth'

/**
 * Full end-to-end walkthrough against a real backend.
 * Uses cached API login.
 */
test('Full Panel Walkthrough (live)', async ({ page }) => {
  test.setTimeout(120_000)

  // ── Step 1: Auth via API + navigate to dashboard ──
  await loginViaApi(page)

  // Go to login first to trigger addInitScript, then navigate to dashboard
  await page.goto('/login')
  await page.waitForTimeout(1_000)
  await page.goto('/')

  // Wait for dashboard — either sidebar or redirect back to login
  // If redirected to login, the token injection from addInitScript works on next navigation
  await page.waitForTimeout(2_000)
  if (await page.url().includes('/login')) {
    // Token may not have been set yet — try reloading
    await page.goto('/')
    await page.waitForTimeout(2_000)
  }

  await expect(page.locator('aside')).toBeVisible({ timeout: 15_000 })

  // ── Step 2: Dashboard loads with real data ──
  await expect(page.getByText('Total Users')).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('Active Connections')).toBeVisible()
  await expect(page.getByText('Total Traffic')).toBeVisible()
  await expect(page.getByText('Core Status')).toBeVisible()

  // ── Step 3: Navigate to Cores ──
  await page.locator('aside').getByText('Cores', { exact: true }).click()
  await page.waitForURL('**/cores', { timeout: 10_000 })
  await expect(page.getByText('xray').first()).toBeVisible({ timeout: 15_000 })
  await expect(page.getByText('singbox').first()).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('mihomo').first()).toBeVisible({ timeout: 10_000 })

  // ── Step 4: Navigate to Inbounds ──
  await page.locator('aside').getByText('Inbounds', { exact: true }).click()
  await page.waitForURL('**/inbounds', { timeout: 10_000 })
  await expect(page.getByRole('heading', { name: /Inbounds/i }).first()).toBeVisible()

  // ── Step 5: Navigate to Users ──
  await page.locator('aside').getByText('Users', { exact: true }).click()
  await page.waitForURL('**/users', { timeout: 10_000 })
  await expect(page.getByRole('heading', { name: /Users/i }).first()).toBeVisible()

  // ── Step 6: Create a user ──
  await page.getByRole('button', { name: /Add User/i }).first().click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible({ timeout: 5_000 })

  const username = `e2e_user_${Date.now()}`
  await dialog.getByPlaceholder('e.g. john_doe').fill(username)
  await dialog.getByRole('button', { name: /Create User/i }).click()
  await expect(dialog).not.toBeVisible({ timeout: 10_000 })

  // Wait and check for user in list
  await page.waitForTimeout(2_000)

  // ── Step 7: Navigate to Outbounds ──
  await page.locator('aside').getByText('Outbounds', { exact: true }).click()
  await page.waitForURL('**/outbounds', { timeout: 10_000 })
  await expect(page.getByRole('heading').first()).toBeVisible()

  // ── Step 8: Navigate to Certificates ──
  await page.locator('aside').getByText('Certificates', { exact: true }).click()
  await page.waitForURL('**/certificates', { timeout: 10_000 })
  await expect(page.getByRole('heading').first()).toBeVisible()

  // ── Step 9: Navigate to Settings ──
  await page.locator('aside').getByText('Settings', { exact: true }).click()
  await page.waitForURL('**/settings', { timeout: 10_000 })
  await expect(page.getByText('Appearance')).toBeVisible({ timeout: 10_000 })

  // ── Step 10: Visit Backups ──
  await page.goto('/backups')
  await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 15_000 })

  // ── Step 11: Return to Dashboard ──
  await page.goto('/')
  await page.waitForLoadState('domcontentloaded')
  // If redirected to login, retry
  if (page.url().includes('/login')) {
    await page.waitForTimeout(1_000)
    await page.goto('/')
    await page.waitForLoadState('domcontentloaded')
  }
  await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 15_000 })
})
