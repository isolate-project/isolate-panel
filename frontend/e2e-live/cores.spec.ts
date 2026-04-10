import { test, expect } from '@playwright/test'
import { loginViaApi, navigateTo } from './auth'

test.describe('Cores (live)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
  })

  test('should display cores page with all 3 cores', async ({ page }) => {
    await navigateTo(page, '/cores')

    await expect(page.getByRole('heading', { name: /Cores/i }).first()).toBeVisible()

    await expect(page.getByText('xray').first()).toBeVisible({ timeout: 15_000 })
    await expect(page.getByText('singbox').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('mihomo').first()).toBeVisible({ timeout: 10_000 })
  })

  test('should show cores with status badges', async ({ page }) => {
    await navigateTo(page, '/cores')
    await page.waitForTimeout(2_000)

    // Each core should show either "Running" or "Stopped" badge
    const running = await page.getByText('Running').count()
    const stopped = await page.getByText('Stopped').count()
    expect(running + stopped).toBeGreaterThanOrEqual(3)
  })

  test('should have start/stop buttons on cores', async ({ page }) => {
    await navigateTo(page, '/cores')
    await page.waitForTimeout(2_000)

    // At least Start or Stop buttons should be visible
    const startButtons = await page.getByRole('button', { name: /Start/i }).count()
    const stopButtons = await page.getByRole('button', { name: /Stop/i }).count()
    expect(startButtons + stopButtons).toBeGreaterThanOrEqual(1)
  })

  test('should click start on a stopped core', async ({ page }) => {
    await navigateTo(page, '/cores')
    await page.waitForTimeout(2_000)

    // Try to start a core
    const startButton = page.getByRole('button', { name: /Start/i }).first()
    if (await startButton.isVisible().catch(() => false)) {
      await startButton.click()

      // Wait for the API response — UI should react (button might disappear or change)
      await page.waitForTimeout(5_000)

      // After start, the page should still be functional
      await expect(page.getByText('xray').first()).toBeVisible()
    }
  })

  test('should show core version info', async ({ page }) => {
    await navigateTo(page, '/cores')
    await page.waitForTimeout(2_000)

    await expect(page.getByText('Version').first()).toBeVisible({ timeout: 10_000 })
  })

  test('should open core logs', async ({ page }) => {
    await navigateTo(page, '/cores')
    await page.waitForTimeout(2_000)

    const logsButton = page.locator('[title="View Logs"]').first()
    if (await logsButton.isVisible().catch(() => false)) {
      await logsButton.click()
      await expect(page.getByText(/Logs/i)).toBeVisible({ timeout: 5_000 })
    }
  })
})
