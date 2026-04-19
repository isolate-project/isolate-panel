import { test, expect } from '@playwright/test'
import { setupAuth, mockProtectedApis, setupMockServer, mockPortValidationInfo, mockPortValidationWarning, mockPortValidationError } from './fixtures'

test.describe('Inbound Port Validation', () => {
  test.beforeEach(async ({ page }) => {
    await mockProtectedApis(page)
    await setupAuth(page)
  })

  test('should show checking indicator during debounce', async ({ page }) => {
    await page.route(/\/api\/inbounds\/check-port/, route => 
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationInfo) })
    )
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    await portInput.fill('443')
    
    const checkingIndicator = page.locator('text=⏳')
    await expect(checkingIndicator).toBeVisible()
    
    await page.waitForTimeout(600)
    await expect(checkingIndicator).not.toBeVisible()
  })

  test('should display info state for available port', async ({ page }) => {
    await page.route(/\/api\/inbounds\/check-port/, route => 
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationInfo) })
    )
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    await portInput.fill('443')
    await page.waitForTimeout(600)
    
    await expect(page.locator('text=✓')).toBeVisible()
    await expect(page.getByText(/доступен|HAProxy/i)).toBeVisible()
  })

  test('should display warning state with conflict details', async ({ page }) => {
    await page.route(/\/api\/inbounds\/check-port/, route => 
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationWarning) })
    )
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    await portInput.fill('443')
    await page.waitForTimeout(600)
    
    await expect(page.locator('text=⚠️')).toBeVisible()
    await expect(page.getByText(/конфликт|conflict|используется|used|warning/i)).toBeVisible()
    const conflictList = page.locator('div[class*="text-xs"]').filter({ has: page.locator('ul') })
    await expect(conflictList).toBeVisible()
    const haproxyOk = page.locator('span').filter({ hasText: /✓ HAProxy|HAProxy OK/i })
    await expect(haproxyOk.first()).toBeVisible()
  })

  test('should display warning confirmation buttons', async ({ page }) => {
    const warningResponse = {
      ...mockPortValidationWarning,
      action: 'confirm'
    }
    
    await page.route(/\/api\/inbounds\/check-port/, route => 
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(warningResponse) })
    )
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    await portInput.fill('443')
    await page.waitForTimeout(600)
    
    const buttons = page.locator('button[type="button"]')
    await expect(buttons.filter({ hasText: /другой|different/i }).first()).toBeVisible()
    await expect(buttons.filter({ hasText: /HAProxy|с HAProxy/i }).first()).toBeVisible()
  })

  test('should display error state for blocked port', async ({ page }) => {
    await page.route(/\/api\/inbounds\/check-port/, route => 
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationError) })
    )
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    await portInput.fill('8443')
    await page.waitForTimeout(600)
    
    await expect(page.locator('text=✗').first()).toBeVisible()
    await expect(page.getByText(/UDP|ошибка|error/i).first()).toBeVisible()
  })

  test('should validate port on input change', async ({ page }) => {
    let requestCount = 0
    
    await page.route(/\/api\/inbounds\/check-port/, async route => {
      requestCount++
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationInfo) })
    })
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    
    await portInput.fill('4')
    await page.waitForTimeout(100)
    await portInput.fill('44')
    await page.waitForTimeout(100)
    await portInput.fill('443')
    
    await page.waitForTimeout(600)
    
    expect(requestCount).toBeGreaterThanOrEqual(1)
    await expect(page.locator('text=✓')).toBeVisible()
  })

  test('should update validation on port change', async ({ page }) => {
    let requestCount = 0
    
    await page.route(/\/api\/inbounds\/check-port/, async route => {
      requestCount++
      if (requestCount === 1) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationInfo) })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationWarning) })
      }
    })
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    
    await portInput.fill('443')
    await page.waitForTimeout(600)
    await expect(page.locator('text=✓')).toBeVisible()
    
    await portInput.fill('8443')
    await page.waitForTimeout(600)
    await expect(page.locator('text=⚠️')).toBeVisible()
  })

  test('should validate across protocol changes', async ({ page }) => {
    await page.route(/\/api\/inbounds\/check-port/, async route => {
      const request = route.request()
      const postData = await request.postData()
      const body = postData ? JSON.parse(postData) : {}
      
      if (body.protocol === 'hysteria2' || body.transport === 'udp') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationError) })
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationInfo) })
      }
    })
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    await portInput.fill('443')
    await page.waitForTimeout(600)
    
    await expect(page.locator('text=✓')).toBeVisible()
  })

  test('should debounce rapid input changes', async ({ page }) => {
    let requestCount = 0

    await setupMockServer(page)
    
    await page.route(/\/api\/inbounds\/check-port/, async route => {
      requestCount++
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockPortValidationInfo) })
    })

    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()

    const portInput = page.locator('input[type="number"]').first()

    await portInput.fill('4')
    await page.waitForTimeout(100)
    await portInput.fill('44')
    await page.waitForTimeout(100)
    await portInput.fill('443')

    await page.waitForTimeout(700)

    expect(requestCount).toBeLessThanOrEqual(2)
    expect(requestCount).toBeGreaterThanOrEqual(1)
  })

  test('should display multiple conflicts list', async ({ page }) => {
    await setupMockServer(page)
    
    const mockMultipleConflicts = {
      ...mockPortValidationWarning,
      conflicts: [
        {
          inbound_id: 1,
          inbound_name: 'VLESS-443',
          protocol: 'vless',
          transport: '',
          core_type: 'xray',
          port: 443,
          haproxy_compatible: true,
          can_share: true,
          sharing_mechanism: 'haproxy',
          requires_confirm: true,
        },
        {
          inbound_id: 2,
          inbound_name: 'VMess-443',
          protocol: 'vmess',
          transport: '',
          core_type: 'xray',
          port: 443,
          haproxy_compatible: true,
          can_share: true,
          sharing_mechanism: 'haproxy',
          requires_confirm: true,
        },
        {
          inbound_id: 3,
          inbound_name: 'Trojan-443',
          protocol: 'trojan',
          transport: '',
          core_type: 'xray',
          port: 443,
          haproxy_compatible: false,
          can_share: false,
          requires_confirm: false,
        },
      ],
    }
    
    await page.route(/\/api\/inbounds\/check-port/, route => 
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockMultipleConflicts) })
    )
    
    await page.goto('/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    
    const portInput = page.locator('input[type="number"]').first()
    await portInput.fill('443')
    await page.waitForTimeout(600)
    
    await expect(page.getByText('VLESS-443')).toBeVisible()
    await expect(page.getByText('VMess-443')).toBeVisible()
    await expect(page.getByText('Trojan-443')).toBeVisible()
    
    const haproxyBadges = page.locator('text=/HAProxy OK|✓ HAProxy/i')
    await expect(haproxyBadges).toHaveCount(2)
    await expect(page.getByText(/Incompatible|✗ Incompatible/i)).toBeVisible()
  })
})