import { test, expect } from '@playwright/test'
import { mockLoginResponse, mockAdmin } from './fixtures'

test.describe('Authentication', () => {
  test('should display login page with username and password fields', async ({ page }) => {
    await page.goto('/login')

    await expect(page.locator('h1')).toContainText('Isolate Panel')
    await expect(page.locator('input[type="text"]')).toBeVisible()
    await expect(page.locator('input[type="password"]')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toContainText('Sign In')
  })

  test('should redirect unauthenticated user to /login', async ({ page }) => {
    await page.goto('/')
    await page.waitForURL('**/login')
    await expect(page.locator('input[type="text"]')).toBeVisible()
  })

  test('should login with valid credentials and redirect to dashboard', async ({ page }) => {
    await page.route(/\/api\/auth\/login/, route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockLoginResponse) })
    )
    await page.route(/\/api\/me/, route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockAdmin) })
    )
    await page.route(/\/api\/stats\//, route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) })
    )
    await page.route(/\/api\/system\//, route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) })
    )
    await page.route(/\/api\/users/, route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ users: [], total: 0 }) })
    )
    await page.route(/\/api\/cores/, route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) })
    )
    await page.route(/\/api\/ws\/ticket/, route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ticket: 't' }) })
    )

    await page.goto('/login')
    await page.locator('input[type="text"]').fill('admin')
    await page.locator('input[type="password"]').fill('password123')
    await page.locator('button[type="submit"]').click()

    // After login, should navigate away from login page — sidebar appears
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })
  })

  test('should reject invalid credentials and stay on login', async ({ page }) => {
    // Note: The axios interceptor intercepts 401 and redirects to /login (even from login page),
    // causing a full page reload that clears the error state. We verify the form remains usable.
    await page.route(/\/api\/auth\/login/, route =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Invalid username or password' }),
      })
    )

    await page.goto('/login')
    await page.locator('input[type="text"]').fill('admin')
    await page.locator('input[type="password"]').fill('wrong')
    await page.locator('button[type="submit"]').click()

    // After the interceptor reload, login form should still be available
    await expect(page.locator('input[type="text"]')).toBeVisible({ timeout: 10000 })
    await expect(page).toHaveURL(/\/login/)
  })

  test('should show TOTP input when server requires it', async ({ page }) => {
    await page.route(/\/api\/auth\/login/, async route => {
      const body = JSON.parse(route.request().postData() || '{}')
      if (!body.totp_code) {
        await route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ requires_totp: true }),
        })
      } else {
        await route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify(mockLoginResponse),
        })
      }
    })

    await page.goto('/login')
    await page.locator('input[type="text"]').fill('admin')
    await page.locator('input[type="password"]').fill('password123')
    await page.locator('button[type="submit"]').click()

    // TOTP input should appear
    await expect(page.locator('input[placeholder="000000"]')).toBeVisible({ timeout: 5000 })
    // Username/password fields should be hidden
    await expect(page.locator('input[type="password"]')).not.toBeVisible()

    // Back button should return to credentials
    await page.getByText('Back').click()
    await expect(page.locator('input[type="text"]')).toBeVisible()
    await expect(page.locator('input[placeholder="000000"]')).not.toBeVisible()
  })

  test('should not submit empty form (HTML5 required)', async ({ page }) => {
    await page.goto('/login')
    await page.locator('button[type="submit"]').click()
    await expect(page.locator('input[type="text"]')).toBeVisible()
    await expect(page).toHaveURL(/\/login/)
  })
})
