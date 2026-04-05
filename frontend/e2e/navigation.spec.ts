import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis } from './fixtures'

test.describe('Sidebar Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  const navItems = [
    { label: 'Users', path: '/users' },
    { label: 'Inbounds', path: '/inbounds' },
    { label: 'Outbounds', path: '/outbounds' },
    { label: 'Cores', path: '/cores' },
    { label: 'Certificates', path: '/certificates' },
    { label: 'Connections', path: '/connections' },
    { label: 'WARP Routes', path: '/warp' },
    { label: 'GeoIP/GeoSite', path: '/geo' },
    { label: 'Backups', path: '/backups' },
    { label: 'Notifications', path: '/notifications' },
    { label: 'Settings', path: '/settings' },
  ]

  for (const { label, path } of navItems) {
    test(`should navigate to ${path} via "${label}" link`, async ({ page }) => {
      await page.goto('/')
      await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

      await page.locator('aside').getByText(label, { exact: true }).click()
      await page.waitForURL(`**${path}`)

      // Page should render without crashing — check for heading or page content
      await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 10000 })
    })
  }

  test('should navigate back to dashboard via Dashboard link', async ({ page }) => {
    await page.goto('/users')
    await expect(page.locator('aside')).toBeVisible({ timeout: 10000 })

    await page.locator('aside').getByText('Dashboard', { exact: true }).click()
    await page.waitForURL('**/')

    await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 10000 })
  })
})
