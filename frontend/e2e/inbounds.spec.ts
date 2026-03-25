import { test, expect } from '@playwright/test';

test.describe('Inbounds Management', () => {
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

    // Navigate to inbounds page
    await page.click('a:has-text("Inbounds"), a:has-text("Входящие")');
    await expect(page).toHaveURL(/\/inbounds/);
  });

  test('should display inbounds list', async ({ page }) => {
    // Mock inbounds API
    await page.route('**/api/inbounds', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          inbounds: [
            { id: 1, name: 'VMess-443', protocol: 'vmess', port: 443, is_enabled: true },
            { id: 2, name: 'VLESS-8443', protocol: 'vless', port: 8443, is_enabled: false },
          ],
          total: 2,
        }),
      });
    });

    await expect(page.locator('text=/VMess|VLESS/i')).toBeVisible();
  });

  test('should toggle inbound status', async ({ page }) => {
    // Mock toggle API
    await page.route('**/api/inbounds/**', async route => {
      if (route.request().method() === 'PUT' || route.request().method() === 'PATCH') {
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
            inbounds: [{ id: 1, name: 'VMess-443', protocol: 'vmess', port: 443, is_enabled: true }],
            total: 1,
          }),
        });
      }
    });

    // Click toggle switch
    await page.click('[role="switch"], input[type="checkbox"], button:has-text("Enable"), button:has-text("Disable")');
    
    // Should show success
    await expect(page.locator('text=/success|updated/i')).toBeVisible();
  });

  test('should delete inbound', async ({ page }) => {
    // Mock delete API
    await page.route('**/api/inbounds/**', async route => {
      if (route.request().method() === 'DELETE') {
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
            inbounds: [{ id: 1, name: 'VMess-443', protocol: 'vmess', port: 443, is_enabled: true }],
            total: 1,
          }),
        });
      }
    });

    // Click delete button
    await page.click('[data-testid="delete-button"], button:has-text("Delete"), button:has-text("Удалить")');
    
    // Confirm if needed
    try {
      await page.click('button:has-text("Confirm"), button:has-text("Yes")');
    } catch (e) {
      // No confirmation
    }
    
    // Should show success
    await expect(page.locator('text=/success|deleted/i')).toBeVisible();
  });
});
