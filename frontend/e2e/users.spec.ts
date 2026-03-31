import { test, expect } from '@playwright/test';

test.describe('Users Management', () => {
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

    // Navigate to users page
    await page.click('a:has-text("Users"), a:has-text("Пользователи")');
    await expect(page).toHaveURL(/\/users/);
  });

  test('should display users list', async ({ page }) => {
    // Mock users API
    await page.route('**/api/users', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          users: [
            { id: 1, username: 'user1', email: 'user1@example.com', is_active: true },
            { id: 2, username: 'user2', email: 'user2@example.com', is_active: false },
          ],
          total: 2,
        }),
      });
    });

    await expect(page.locator('text=/user1|user2/i')).toBeVisible();
  });

  test('should create new user', async ({ page }) => {
    // Mock create user API
    await page.route('**/api/users', async route => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            user: {
              id: 3,
              username: 'newuser',
              email: 'newuser@example.com',
              is_active: true,
            },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, users: [], total: 0 }),
        });
      }
    });

    // Click create button
    await page.click('button:has-text("Create"), button:has-text("Создать"), button:has-text("+")');
    
    // Fill form
    await page.fill('input[name="username"], input[placeholder*="username"]', 'newuser');
    await page.fill('input[name="email"], input[type="email"]', 'newuser@example.com');
    await page.fill('input[name="password"], input[type="password"]', 'password123');
    
    // Submit
    await page.click('button[type="submit"]');
    
    // Should show success or redirect
    await expect(page.locator('text=/success|newuser/i')).toBeVisible();
  });

  test('should delete user', async ({ page }) => {
    // Mock delete API
    await page.route('**/api/users/**', async route => {
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
            users: [{ id: 1, username: 'user1', email: 'user1@example.com', is_active: true }],
            total: 1,
          }),
        });
      }
    });

    // Click delete button
    await page.click('[data-testid="delete-button"], button:has-text("Delete"), button:has-text("Удалить")');
    
    // Confirm deletion if there's a confirmation dialog
    try {
      await page.click('button:has-text("Confirm"), button:has-text("Yes")');
    } catch {
      // No confirmation dialog
    }
    
    // Should show success message
    await expect(page.locator('text=/success|deleted/i')).toBeVisible();
  });
});
