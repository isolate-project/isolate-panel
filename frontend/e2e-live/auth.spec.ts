import { test, expect } from '@playwright/test'
import { loginViaApi, ADMIN } from './auth'

test.describe('Authentication (live)', () => {
  test.describe.configure({ mode: 'serial' })

  test('should redirect unauthenticated user to login', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/login/, { timeout: 15_000 })
  })

  test('should not submit with empty form', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByPlaceholder('admin')).toBeVisible()

    await page.getByRole('button', { name: /Sign In/i }).click()

    // Should stay on login page
    await expect(page).toHaveURL(/\/login/)
  })

  test('should login with valid credentials', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByPlaceholder('admin')).toBeVisible()

    await page.getByPlaceholder('admin').fill(ADMIN.username)
    await page.getByPlaceholder('••••••••').fill(ADMIN.password)
    await page.getByRole('button', { name: /Sign In/i }).click()

    // Should redirect to dashboard
    await expect(page.locator('aside')).toBeVisible({ timeout: 15_000 })

    // Token should be real, not mocked
    const token = await page.evaluate(() => localStorage.getItem('accessToken'))
    expect(token).toBeTruthy()
    expect(token).not.toBe('mock-access-token')
  })

  test('should reject invalid credentials', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByPlaceholder('admin')).toBeVisible()

    await page.getByPlaceholder('admin').fill('wronguser')
    await page.getByPlaceholder('••••••••').fill('wrongpassword')
    await page.getByRole('button', { name: /Sign In/i }).click()

    // Wait for the API response to complete
    await page.waitForTimeout(3_000)

    // Should NOT redirect — still on login page (no valid token obtained)
    await expect(page).toHaveURL(/\/login/)

    // No token should be stored
    const token = await page.evaluate(() => localStorage.getItem('accessToken'))
    expect(token).toBeFalsy()
  })

  test('should persist auth across navigation', async ({ page }) => {
    // Use cached API login to avoid rate limiting
    await loginViaApi(page)

    await page.goto('/users')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10_000 })

    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10_000 })

    const token = await page.evaluate(() => localStorage.getItem('accessToken'))
    expect(token).toBeTruthy()
  })
})
