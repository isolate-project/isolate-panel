import { test, expect } from '@playwright/test';

test.describe('Backups Management', () => {
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

    // Mock backups list
    await page.route('**/api/backups', async route => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: [
              { id: 1, filename: 'backup_2025.tar.gz', file_size_bytes: 1024, backup_type: 'manual', status: 'completed', created_at: '2025-01-01T00:00:00Z' }
            ]
          }),
        });
      } else {
        await route.continue();
      }
    });

    await page.route('**/api/backups/schedule', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { schedule: '0 3 * * *', next_run: '' } }),
      });
    });

    await page.route('**/api/system/settings', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [{ key: 'backup_retention_count', value: '5' }] }),
      });
    });

    // Login
    await page.goto('/');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'password123');
    await page.click('button[type="submit"]');

    // Go to backups page
    await page.goto('/settings/backups'); 
    
    // In case there is no nested routing and it's just /backups
    // A quick hack for robust routing without knowing the exact router tree:
    // If it navigates to 404, fallback to /backups
    const content = await page.content();
    if (content.includes('404')) {
      await page.goto('/backups');
    }
  });

  test('should display list of backups', async ({ page }) => {
    await expect(page.locator('text=backup_2025.tar.gz')).toBeVisible();
  });

  test('should initiate create backup', async ({ page }) => {
    let createCalled = false;
    await page.route('**/api/backups/create', async route => {
      createCalled = true;
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ message: "Backup started" })
      });
    });

    // Mock alert that occurs when backup succeeds
    page.on('dialog', dialog => dialog.accept());

    await page.click('button:has-text("Initiate Secure Backup")');
    await expect(async () => { expect(createCalled).toBe(true); }).toPass();
  });

  test('should restore backup', async ({ page }) => {
    let restoreCalled = false;
    await page.route('**/api/backups/1/restore', async route => {
      restoreCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: "Restore started" })
      });
    });

    // Mock alert that occurs when restore succeeds
    page.on('dialog', dialog => dialog.accept());

    // Click the restore button 
    await page.click('button[title="Restore System"]');
    
    // There is a custom Modal from components/ui/Modal which sets a "Confirm" button
    await page.click('button:has-text("Confirm")');

    await expect(async () => { expect(restoreCalled).toBe(true); }).toPass();
  });
});
