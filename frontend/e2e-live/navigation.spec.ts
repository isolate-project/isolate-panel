import { test, expect } from '@playwright/test'
import { loginViaApi } from './auth'

test.describe('Navigation (live)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
  })

  // Pages that use PageLayout (with <main>)
  const pagesWithMain = [
    { name: 'Dashboard', path: '/' },
    { name: 'Users', path: '/users' },
    { name: 'Inbounds', path: '/inbounds' },
    { name: 'Outbounds', path: '/outbounds' },
    { name: 'Certificates', path: '/certificates' },
    { name: 'Settings', path: '/settings' },
  ]

  // Pages that don't use PageLayout (plain <div> root)
  const pagesWithoutMain = [
    { name: 'Cores', path: '/cores' },
    { name: 'Backups', path: '/backups' },
    { name: 'Notifications', path: '/notifications' },
    { name: 'WARP', path: '/warp' },
    { name: 'Geo Rules', path: '/geo' },
    { name: 'Active Connections', path: '/connections' },
  ]

  for (const pg of pagesWithMain) {
    test(`should load ${pg.name} page`, async ({ page }) => {
      await page.goto(pg.path)
      await page.waitForLoadState('networkidle', { timeout: 15_000 })
      await expect(page.locator('main').first()).toBeVisible({ timeout: 15_000 })
      await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 10_000 })
    })
  }

  for (const pg of pagesWithoutMain) {
    test(`should load ${pg.name} page`, async ({ page }) => {
      await page.goto(pg.path)
      await page.waitForLoadState('networkidle', { timeout: 15_000 })
      // These pages don't have <main> — just check heading renders
      await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 15_000 })
    })
  }

  test('should navigate between pages via sidebar', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('aside')).toBeVisible({ timeout: 15_000 })

    // Navigate to Users
    await page.locator('aside').getByText('Users', { exact: true }).click()
    await page.waitForURL('**/users', { timeout: 10_000 })
    await expect(page.getByRole('heading').first()).toBeVisible()

    // Navigate to Inbounds
    await page.locator('aside').getByText('Inbounds', { exact: true }).click()
    await page.waitForURL('**/inbounds', { timeout: 10_000 })
    await expect(page.getByRole('heading').first()).toBeVisible()

    // Navigate to Settings
    await page.locator('aside').getByText('Settings', { exact: true }).click()
    await page.waitForURL('**/settings', { timeout: 10_000 })
    await expect(page.getByRole('heading').first()).toBeVisible()
  })
})
