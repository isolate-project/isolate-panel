import { chromium } from 'playwright';
import fs from 'fs';

const TOKEN_FILE = 'e2e-live/.auth-tokens.json';

async function run() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ baseURL: 'http://127.0.0.1:5173' });
  
  // start vite in background
  const { spawn } = await import('child_process');
  const vite = spawn('npm', ['run', 'dev'], { detached: true });
  await new Promise(r => setTimeout(r, 3000));
  
  const page = await context.newPage();
  
  // setup auth
  const raw = fs.readFileSync(TOKEN_FILE, 'utf-8');
  const tokens = JSON.parse(raw);
  await page.addInitScript(
    ({ token, refresh, admin }) => {
      const state = {
        state: { accessToken: token, refreshToken: refresh, user: admin, isAuthenticated: true, isLoading: false },
        version: 0,
      };
      localStorage.setItem('auth-storage', JSON.stringify(state));
      localStorage.setItem('accessToken', token);
      localStorage.setItem('refreshToken', refresh);
    },
    { token: tokens.access_token, refresh: tokens.refresh_token, admin: tokens.admin },
  );

  page.on('console', msg => console.log('BROWSER CONSOLE:', msg.text()));
  page.on('pageerror', err => console.log('BROWSER ERROR:', err));

  const response = await page.goto('/users');
  console.log('Navigated to /users. Status:', response?.status());
  
  // wait and try to get html
  await page.waitForTimeout(5000);
  
  const bodyHTML = await page.evaluate(() => document.body.innerHTML);
  fs.writeFileSync('page_body.html', bodyHTML);
  console.log("Body length:", bodyHTML.length);
  
  if (bodyHTML.includes('Users')) {
    console.log("Found 'Users' in HTML!");
  } else {
    console.log("Did not find 'Users' in HTML!");
  }
  
  await browser.close();
  process.kill(-vite.pid);
}

run().catch(console.error);
