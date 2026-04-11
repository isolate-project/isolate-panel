import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright config for live E2E tests against a real backend.
 *
 * The backend runs in Docker (docker/docker-compose.test.yml) and is started
 * automatically via globalSetup. Vite dev server proxies /api to localhost:8080.
 *
 * Run:  npm run test:e2e:live
 * UI:   npm run test:e2e:live:ui
 */
export default defineConfig({
  testDir: './e2e-live',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 1,
  workers: 1,
  reporter: [['html', { outputFolder: 'playwright-report-live' }]],

  globalSetup: './e2e-live/global-setup.ts',
  globalTeardown: './e2e-live/global-teardown.ts',

  use: {
    baseURL: 'http://127.0.0.1:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    actionTimeout: 15_000,
    navigationTimeout: 15_000,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    command: 'npm run dev',
    url: 'http://127.0.0.1:5173',
    reuseExistingServer: true,
    timeout: 30_000,
  },

  /* Increase global test timeout — real backend is slower than mocks */
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
})
