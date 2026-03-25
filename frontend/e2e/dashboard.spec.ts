import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Mock authentication
    await page.route('**/api/auth/login', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          token: 'mock-jwt-token',
          user: { id: 1, username: 'admin', email: 'admin@example.com', is_admin: true },
        }),
      });
    });

    // Login
    await page.goto('/');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'password123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('should display dashboard stats', async ({ page }) => {
    // Mock stats API
    await page.route('**/api/stats/dashboard', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          total_users: 150,
          active_users: 120,
          online_users: 45,
          total_inbounds: 25,
          cores_running: 2,
          cores_total: 3,
        }),
      });
    });

    // Wait for stats to load
    await expect(page.locator('text=/150|Total Users/i')).toBeVisible();
  });

  test('should display cores status', async ({ page }) => {
    // Mock cores API
    await page.route('**/api/cores', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          cores: [
            { id: 1, name: 'xray', type: 'xray', status: 'running' },
            { id: 2, name: 'singbox', type: 'singbox', status: 'running' },
            { id: 3, name: 'mihomo', type: 'mihomo', status: 'stopped' },
          ],
        }),
      });
    });

    await expect(page.locator('text=/xray|singbox|mihomo/i')).toBeVisible();
  });

  test('should refresh stats', async ({ page }) => {
    let callCount = 0;
    
    await page.route('**/api/stats/dashboard', async route => {
      callCount++;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          total_users: 150 + callCount,
          active_users: 120,
          online_users: 45,
        }),
      });
    });

    // Click refresh button
    await page.click('[data-testid="refresh-button"], button:has-text("Refresh")');
    
    // API should be called again
    expect(callCount).toBeGreaterThan(1);
  });
});
