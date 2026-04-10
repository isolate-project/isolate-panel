import { test, expect } from '@playwright/test'
import { loginViaApi, navigateTo } from './auth'

test.describe('Settings (live)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
  })

  test('should display settings page', async ({ page }) => {
    await navigateTo(page, '/settings')

    await expect(page.getByRole('heading', { name: /Settings/i }).first()).toBeVisible()
  })

  test('should show appearance section', async ({ page }) => {
    await navigateTo(page, '/settings')

    await expect(page.getByText('Appearance')).toBeVisible({ timeout: 10_000 })

    // Use exact: true to avoid matching "Theme and language settings..."
    await expect(page.getByText('Theme', { exact: true }).first()).toBeVisible()

    // Theme buttons should be present
    await expect(page.getByRole('button', { name: /Light Mode/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /Dark Mode/i })).toBeVisible()
  })

  test('should show general settings', async ({ page }) => {
    await navigateTo(page, '/settings')

    await expect(page.getByText('General')).toBeVisible({ timeout: 10_000 })

    // Panel Name input
    const panelNameInput = page.locator('input[name="panel_name"]')
    await expect(panelNameInput).toBeVisible()
  })

  test('should show monitoring mode section', async ({ page }) => {
    await navigateTo(page, '/settings')

    await expect(page.getByText('Monitoring Mode')).toBeVisible({ timeout: 10_000 })
  })

  test('should toggle theme', async ({ page }) => {
    await navigateTo(page, '/settings')

    await page.getByRole('button', { name: /Dark Mode/i }).click()
    await page.waitForTimeout(500)

    await page.getByRole('button', { name: /Light Mode/i }).click()
    await page.waitForTimeout(500)
  })

  test('should show security section', async ({ page }) => {
    await navigateTo(page, '/settings')

    // Scroll down to find Security section
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(500)

    await expect(page.getByText('Security')).toBeVisible({ timeout: 10_000 })
  })
})
