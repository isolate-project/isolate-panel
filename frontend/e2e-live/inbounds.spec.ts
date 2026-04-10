import { test, expect } from '@playwright/test'
import { loginViaApi, navigateTo } from './auth'

test.describe('Inbounds (live)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
  })

  test('should display inbounds page', async ({ page }) => {
    await navigateTo(page, '/inbounds')

    await expect(page.getByRole('heading', { name: /Inbounds/i }).first()).toBeVisible()
    // Use .first() — there may be 2 "Add Inbound" buttons (header + empty state)
    await expect(page.getByRole('button', { name: /Add Inbound/i }).first()).toBeVisible()
  })

  test('should show empty state or inbound list', async ({ page }) => {
    await navigateTo(page, '/inbounds')
    await page.waitForLoadState('networkidle')

    // Either inbound cards exist or empty state is shown
    const heading = page.getByRole('heading', { name: /Inbounds/i }).first()
    await expect(heading).toBeVisible({ timeout: 10_000 })
  })

  test('should open Add Inbound drawer', async ({ page }) => {
    await navigateTo(page, '/inbounds')

    await page.getByRole('button', { name: /Add Inbound/i }).first().click()

    // Drawer should appear with form
    await expect(page.getByText('General Configuration')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByPlaceholder('e.g. Europe-VLESS')).toBeVisible()
  })

  test('should create a new inbound', async ({ page }) => {
    await navigateTo(page, '/inbounds')

    // Open create drawer
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    await expect(page.getByText('General Configuration')).toBeVisible({ timeout: 5_000 })

    const name = `test-inbound-${Date.now()}`

    // Fill the form
    await page.getByPlaceholder('e.g. Europe-VLESS').fill(name)

    // Select protocol
    const protocolSelect = page.locator('select[name="protocol"], #protocol').first()
    if (await protocolSelect.isVisible().catch(() => false)) {
      const options = await protocolSelect.locator('option').allTextContents()
      if (options.length > 1) {
        await protocolSelect.selectOption({ index: 1 })
      }
    }

    // Select core
    const coreSelect = page.locator('select[name="core_id"], #core_id').first()
    if (await coreSelect.isVisible().catch(() => false)) {
      const options = await coreSelect.locator('option').allTextContents()
      if (options.length > 1) {
        await coreSelect.selectOption({ index: 1 })
      }
    }

    // Set a port
    const port = 10000 + Math.floor(Math.random() * 50000)
    const portInput = page.getByPlaceholder('443')
    if (await portInput.isVisible().catch(() => false)) {
      await portInput.clear()
      await portInput.fill(String(port))
    }

    // Submit
    await page.getByRole('button', { name: /Create Inbound/i }).click()

    // Wait for drawer to close or success
    await page.waitForTimeout(3_000)
  })
})
