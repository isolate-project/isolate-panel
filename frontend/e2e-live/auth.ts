import { Page, expect } from '@playwright/test'
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const TOKEN_FILE = resolve(__dirname, '.auth-tokens.json')
const ADMIN = { username: 'admin', password: 'admin' }

/**
 * Read cached tokens produced by globalSetup (single login, no rate-limit risk).
 */
function getCachedTokens() {
  const raw = readFileSync(TOKEN_FILE, 'utf-8')
  return JSON.parse(raw) as {
    access_token: string
    refresh_token: string
    admin: Record<string, unknown>
  }
}

/**
 * Inject cached admin tokens into the page's localStorage.
 * Does NOT call the login API — tokens were obtained once in globalSetup.
 */
export async function loginViaApi(page: Page): Promise<void> {
  const tokens = getCachedTokens()

  await page.addInitScript(
    ({ token, refresh, admin }) => {
      const state = {
        state: {
          accessToken: token,
          refreshToken: refresh,
          user: admin,
          isAuthenticated: true,
          isLoading: false,
        },
        version: 0,
      }
      localStorage.setItem('auth-storage', JSON.stringify(state))
      localStorage.setItem('accessToken', token)
      localStorage.setItem('refreshToken', refresh)
    },
    { token: tokens.access_token, refresh: tokens.refresh_token, admin: tokens.admin },
  )
}

/**
 * Login as admin through the actual UI form.
 */
export async function loginViaUi(page: Page, creds = ADMIN): Promise<void> {
  await page.goto('/login')
  await expect(page.getByPlaceholder('admin')).toBeVisible({ timeout: 15_000 })

  await page.getByPlaceholder('admin').fill(creds.username)
  await page.getByPlaceholder('••••••••').fill(creds.password)
  await page.getByRole('button', { name: /Sign In/i }).click()

  await expect(page.locator('aside')).toBeVisible({ timeout: 15_000 })

  const token = await page.evaluate(() => localStorage.getItem('accessToken'))
  expect(token).toBeTruthy()
}

/**
 * Navigate to a page with auth.
 * addInitScript sets localStorage before any JS runs on each navigation.
 * If we end up on /login, it means the auth guard fired before localStorage was read —
 * retry once.
 */
export async function navigateTo(page: Page, path: string): Promise<void> {
  await page.goto(path)
  await page.waitForLoadState('domcontentloaded', { timeout: 15_000 })

  // If redirected to login, tokens weren't ready yet — retry
  if (page.url().includes('/login') && path !== '/login') {
    await page.waitForTimeout(1_000)
    await page.goto(path)
    await page.waitForLoadState('domcontentloaded', { timeout: 15_000 })
  }
}

export { ADMIN }
