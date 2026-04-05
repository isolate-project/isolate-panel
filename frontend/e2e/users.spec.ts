import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis, mockUsers } from './fixtures'

test.describe('Users Management', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  test('should display users list', async ({ page }) => {
    await page.goto('/users')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    // User names from mock data — use .first() because name appears in both desktop table and mobile card
    await expect(page.getByText(mockUsers[0].username).first()).toBeVisible({ timeout: 5000 })
    await expect(page.getByText(mockUsers[1].username).first()).toBeVisible()
  })

  test('should show Add User button', async ({ page }) => {
    await page.goto('/users')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    const addButton = page.locator('button', { hasText: /Add|Create|Создать/i })
    await expect(addButton.first()).toBeVisible({ timeout: 5000 })
  })

  test('should open create user form', async ({ page }) => {
    await page.goto('/users')
    await expect(page.locator('main')).toBeVisible({ timeout: 10000 })

    await page.locator('button', { hasText: /Add|Create|Создать/i }).first().click()

    await expect(page.locator('input[name="username"]')).toBeVisible({ timeout: 5000 })
  })

  test('should navigate to users page via sidebar', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

    await page.locator('aside').getByText('Users').click()
    await page.waitForURL('**/users')

    await expect(page.getByText(mockUsers[0].username).first()).toBeVisible({ timeout: 5000 })
  })
})
