import { test, expect } from '@playwright/test';

test('Debug dashboard requests', async ({ page }) => {
  let requestCount = 0;
  const requests: string[] = [];
  
  page.on('request', request => {
    const url = request.url();
    if (url.includes('/api/')) {
      requestCount++;
      requests.push(`${request.method()} ${url}`);
    }
  });
  
  await page.goto('http://localhost:8080/login');
  await page.fill('input[name="username"]', 'admin');
  await page.fill('input[name="password"]', 'admin');
  await page.click('button[type="submit"]');
  
  await expect(page).toHaveURL('http://localhost:8080/');
  await page.waitForTimeout(2000);
  
  console.log(`Requests in 2s: ${requestCount}`);
  
  await page.waitForTimeout(3000);
  console.log(`Total requests: ${requestCount}`);
  
  await page.screenshot({ path: 'tests/screenshots/dashboard.png' });
});
