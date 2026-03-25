import { test, expect } from '@playwright/test';

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    // Mock authentication and login
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

    await page.goto('/');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'password123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/dashboard/);

    // Navigate to settings page
    await page.click('a:has-text("Settings"), a:has-text("Настройки")');
    await expect(page).toHaveURL(/\/settings/);
  });

  test('should display settings page', async ({ page }) => {
    // Mock settings API
    await page.route('**/api/settings', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          settings: {
            panel_name: 'Isolate Panel',
            monitoring_mode: 'lite',
          },
        }),
      });
    });

    await expect(page.locator('text=/Settings|Настройки/i')).toBeVisible();
  });

  test('should update monitoring mode', async ({ page }) => {
    // Mock settings API
    await page.route('**/api/settings', async route => {
      if (route.request().method() === 'PUT' || route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            settings: { panel_name: 'Test Panel', monitoring_mode: 'lite' },
          }),
        });
      }
    });

    // Change monitoring mode
    await page.selectOption('select[name="monitoring_mode"], select:has-text("lite")', 'full');
    
    // Save
    await page.click('button[type="submit"], button:has-text("Save"), button:has-text("Сохранить")');
    
    // Should show success
    await expect(page.locator('text=/success|saved/i')).toBeVisible();
  });

  test('should update panel name', async ({ page }) => {
    await page.route('**/api/settings', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    });

    // Fill panel name
    await page.fill('input[name="panel_name"], input[placeholder*="panel"]', 'New Panel Name');
    
    // Save
    await page.click('button[type="submit"]');
    
    // Should show success
    await expect(page.locator('text=/success|saved/i')).toBeVisible();
  });
});
