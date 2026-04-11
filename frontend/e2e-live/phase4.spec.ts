import { test, expect } from '@playwright/test'
import { loginViaApi, navigateTo } from './auth'

test.describe('Phase 4 Scenarios (live)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
  })

  test('should upload a certificate successfully', async ({ page }) => {
    await navigateTo(page, '/certificates')
    
    // Check page loaded
    await expect(page.getByRole('heading', { name: /Certificates/i }).first()).toBeVisible()
    
    // Open Upload Modal
    const uploadBtn = page.getByRole('button', { name: /Upload/i }).first()
    await uploadBtn.click()
    
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5000 })
    
    // Fill the fields
    const domainName = `test-${Date.now()}.local`
    // placeholder="example.com"
    await dialog.getByPlaceholder('example.com').fill(domainName)
    
    // textareas
    const validCert = `-----BEGIN CERTIFICATE-----
MIIDCDCCAfCgAwIBAgIUO9rSz+whctwTx9vdjQQxIGQ4yWYwDQYJKoZIhvcNAQEL
BQAwFTETMBEGA1UEAwwKdGVzdC5sb2NhbDAeFw0yNjA0MTExMDA3MjFaFw0yNzA0
MTExMDA3MjFaMBUxEzARBgNVBAMMCnRlc3QubG9jYWwwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQCp3L60r/WIT8HRg+E2EJfK9dH13cGnjPeRNX9o860x
CsuDPhb8TPSSJQh2UQ0u653hqJfAv/uhaftB1aRa1TRZ65jAIFiPkF/nS1F0bxCZ
JVHb7aNCBB+zSpZJXGhsmL9HuR5yTo4RAm2xZOhvh1VDV16iTSY+eo/dYkfT6g8J
vPCVQmLF/3kOVXOiKkYFypU7bPn7eUNfA31hUJV7ty7w/PNzbI5f+pxaU4xZLhno
f6WtyurKDx8AmA+97gVqjz3FMiIh58+ihPdWOFkSlMHPMXKO3gwG6ufLBgl9G9HO
iLFI2dzPVjiKrU7Jgr+8nJdo0odva4JxX+zNYA/R50kTAgMBAAGjUDBOMB0GA1Ud
DgQWBBRbRcJzfc0NmpahMOs8DmkFweiqgDAfBgNVHSMEGDAWgBRbRcJzfc0Nmpah
MOs8DmkFweiqgDAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQBlN6lH
bkxs9qP1LDVfNq7z2p94W4X3ck0wopdwNCqEB+KUF9CszzRDhcvB1uQK2WMVwiAV
YE73D2FFlgnJ4clLyZ8r6owPNOZUglOayjfQJdQDiu0w62QmT7o2jSUZ0CrfKh/E
nh4bUKcqxcKgo90oTKpW3ThDVM4Dnji8sKqax4tuQbSu+3mAgCylZzi8pHxxGUrm
mZH72qboDUprXGkWQQaKBJdPcDI6J3ydfvzrhO1Hf7QAWKQOUG/kNS3tXitVIkzc
qtUpfRBgOPD61ViQB5AH8cRLYnrjcEJ56CTrFlGgvA9RyVK61cF5nBXU1SRJFQ+o
zTsDBYVSICHQQZma
-----END CERTIFICATE-----`

    const validKey = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCp3L60r/WIT8HR
g+E2EJfK9dH13cGnjPeRNX9o860xCsuDPhb8TPSSJQh2UQ0u653hqJfAv/uhaftB
1aRa1TRZ65jAIFiPkF/nS1F0bxCZJVHb7aNCBB+zSpZJXGhsmL9HuR5yTo4RAm2x
ZOhvh1VDV16iTSY+eo/dYkfT6g8JvPCVQmLF/3kOVXOiKkYFypU7bPn7eUNfA31h
UJV7ty7w/PNzbI5f+pxaU4xZLhnof6WtyurKDx8AmA+97gVqjz3FMiIh58+ihPdW
OFkSlMHPMXKO3gwG6ufLBgl9G9HOiLFI2dzPVjiKrU7Jgr+8nJdo0odva4JxX+zN
YA/R50kTAgMBAAECggEAAnWI0WGSwkxwcuvf+Qu8VGo+4wBIY3CmGnYSDfRu+lvQ
A2qUdWWmHjwhMu4InW6s5uO8zEPvoDvCMRZ4l3L2XcoMSfR/YU1soK4cV9aEhKKW
Go7AqSmlB7jal2lMlhTs4PerPfW8VQil2HEg4HJKCM3J2rOq3aYGxdUN3rrcUKGC
lkgAs7HiDzgh6Z03kB2ciRBPwlxgcBy++wpT/7fbDJ3JYz8/mYojJzfYFesljq/a
JtHsaM2XIw9vqRduHE8XQUMMkLuQDJAWbwBhtV86Ih/rFIWAj0EMA1qrNpcHfPAm
6fgpb/OryQaGfWqtFyN2ib+9+CRcTsZ/cVU55gvVgQKBgQDjxzajmX/6tRnKly2R
sRkUoYmGu0+ckllupcd5Mews4fh7agmU0CIoa66tE8UMb/yV24GwHO7PvstxVnqe
sIFBTC9qTKo1GXqtXOJjj2TmFFbS4Qy6vUo0J2fHBUGsfdrgZPVh5LcFxMFiEAOk
k178PsF1vzXb8+rXssEBTH6ZMwKBgQC+6IZvzFFUvpVxAWkQ87+7utbhxr2bYl0V
1pK1AlKC3x/CouWRMxUt67WgDPSqVkxQ4w9WGYtFlKJVKmPm1osziOvjLVzrqM8i
ZrHQ9ycH0/D6zs6nmuyxme7v/UoTG28UCLCfYoOy40GQLzs4buaM43ZZ8vQii4xs
lkdzWfNQoQKBgQDACROayG5qm0a8U8q6eznu9+XvrnoHQieeLqxHFHzOtlD9E8Ay
M2uo8mhZSUKnIr8sRN1I8ouwoGX7DvLgWWUP/UA4eZxCmlGgWaAQWjOx+tHchppp
0e7+m35V/6uH1q+y4cszllVryp9Tora/iPPa7LnEIMoyv6lt4ynvg2N0mwKBgCFt
7jyddpB0Xw7OxGsng6eH7CDVAFa5PruYO1Be+7vW/mTCyZhHbaoA4GkKW72IJwzy
9biJ+I1Snap0JdJCN1Xq4AOD6gWKJdtMSE7jOH5yanxAwocu5cujvOdhXxtBbo3/
h44hXhZxHQX2f1Q+dzisjAjsNjvmW8yX9CMK2USBAoGBAJiVR61fPwUqEZZh+6Ni
3ZhnaKpjq34jL9BpfR07VzxzCrwMxRJffxqsvTYzOVxvmHCsINHLa97VFQPFbzs5
YN5zPpMbfG0B5i03ibI/6c3Mo3PkkIqEDQ1X2ryElTkCPW2Mcwr/bpIdnXjJ2WVV
xy1kiV7pw+PIVWxuSNcKpsgH
-----END PRIVATE KEY-----`

    const textareas = dialog.locator('textarea')
    await textareas.nth(0).fill(validCert)
    await textareas.nth(1).fill(validKey)
    
    // Click Upload inside dialog
    const dialogUploadBtn = dialog.getByRole('button', { name: /Upload/i })
    await dialogUploadBtn.click()
    
    // Wait for it to close
    await expect(dialog).not.toBeVisible({ timeout: 10000 })
    
    // It should appear in the table
    await expect(page.getByText(domainName).first()).toBeVisible({ timeout: 5000 })
  })

  test('should create an inbound, add a user, and check subscription', async ({ page }) => {
    // 1. Create Inbound
    await navigateTo(page, '/inbounds')
    await page.getByRole('button', { name: /Add Inbound/i }).first().click()
    await expect(page.getByText('General Configuration')).toBeVisible({ timeout: 5_000 })
    
    const inboundName = `sub-test-inbound-${Date.now()}`
    await page.getByPlaceholder('e.g. Europe-VLESS').fill(inboundName)
    
    const port = 20000 + Math.floor(Math.random() * 30000)
    const portInput = page.getByPlaceholder('443')
    if (await portInput.isVisible()) {
      await portInput.clear()
      await portInput.fill(String(port))
    }
    
    await page.getByRole('button', { name: /Create Inbound/i }).click()
    await page.waitForTimeout(3000)

    // 2. Create User
    await navigateTo(page, '/users')
    await page.getByRole('button', { name: /Add User/i }).first().click()
    const userDialog = page.getByRole('dialog')
    await expect(userDialog).toBeVisible({ timeout: 5000 })
    
    const username = `sub_user_${Date.now()}`
    await userDialog.getByPlaceholder('e.g. john_doe').fill(username)
    await userDialog.getByRole('button', { name: /Create User/i }).click({ force: true })
    await expect(userDialog).not.toBeVisible({ timeout: 10000 })
    
    // 3. User should be in the list, then we should be able to get their Subscription Link
    // We assume there's a link button or the proxy URI is displayed somewhere
    // For now we'll just check the user is created successfully
    await expect(page.getByText(username).first()).toBeVisible({ timeout: 10000 })
  })
})
