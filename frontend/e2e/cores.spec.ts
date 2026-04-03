import { test, expect } from '@playwright/test';

test.describe('Cores Management', () => {
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

    // Mock Cores
    await page.route('**/api/cores', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: 1, name: 'xray', type: 'xray', is_running: true, is_enabled: true, uptime_seconds: 120, pid: 1234, restart_count: 0 },
          { id: 2, name: 'singbox', type: 'singbox', is_running: false, is_enabled: false, uptime_seconds: 0, pid: 0, restart_count: 0 },
        ]),
      });
    });

    // Login
    await page.goto('/');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'password123');
    await page.click('button[type="submit"]');

    // Go to cores page
    await page.goto('/cores');
  });

  test('should display list of cores', async ({ page }) => {
    await expect(page.locator('text=/xray/i')).toBeVisible();
    await expect(page.locator('text=/singbox/i')).toBeVisible();
  });

  test('should stop a running core', async ({ page }) => {
    let stopCalled = false;
    await page.route('**/api/cores/xray/stop', async route => {
      stopCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ message: 'stopped' }) });
    });

    const xrayCard = page.locator('.p-6', { hasText: 'xray' }).first();
    const stopButton = xrayCard.locator('button', { has: page.locator('svg.lucide-square') });
    
    // Mock browser confirm dialog
    page.on('dialog', dialog => dialog.accept());
    
    await stopButton.click();
    await expect(async () => { expect(stopCalled).toBe(true); }).toPass();
  });

  test('should start a stopped core', async ({ page }) => {
    let startCalled = false;
    await page.route('**/api/cores/singbox/start', async route => {
      startCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ message: 'started' }) });
    });

    const singboxCard = page.locator('.p-6', { hasText: 'singbox' }).first();
    const startButton = singboxCard.locator('button', { has: page.locator('svg.lucide-play') });

    await startButton.click();
    await expect(async () => { expect(startCalled).toBe(true); }).toPass();
  });
});
