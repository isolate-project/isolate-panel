import { chromium } from 'playwright';
(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  const loginRes = await page.request.post('http://127.0.0.1:8080/api/auth/login', {
    data: { username: process.env.TEST_ADMIN_USER || 'admin', password: process.env.TEST_ADMIN_PASS || 'admin' }
  });
  const data = await loginRes.json();
  const accessToken = data.access_token;
  const postRes = await page.request.post('http://127.0.0.1:8080/api/users', {
    data: { username: 'testuser_ABC', email: '', is_active: true, unlimited: true, inbound_ids: [] },
    headers: { 'Authorization': `Bearer ${accessToken}` }
  });
  const resText = await postRes.text();
  console.log("POST Response Status:", postRes.status());
  console.log("POST Response Body:", resText);
  await browser.close();
})();
