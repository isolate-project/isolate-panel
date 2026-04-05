import { Page } from '@playwright/test'

// ── Mock data matching real backend API shapes ──

export const mockAdmin = {
  id: 1,
  username: 'admin',
  email: 'admin@example.com',
  is_super_admin: true,
}

export const mockLoginResponse = {
  access_token: 'mock-access-token',
  refresh_token: 'mock-refresh-token',
  expires_in: 86400,
  admin: mockAdmin,
}

export const mockUsers = [
  {
    id: 1, username: 'alice', email: 'alice@example.com', is_active: true,
    traffic_limit_bytes: 10737418240, traffic_used_bytes: 3221225472,
    subscription_token: 'tok-alice', expires_at: '2026-12-31T23:59:59Z',
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 2, username: 'bob', email: 'bob@example.com', is_active: false,
    traffic_limit_bytes: 0, traffic_used_bytes: 0,
    subscription_token: 'tok-bob', expires_at: null,
    created_at: '2026-02-15T00:00:00Z',
  },
]

export const mockCores = [
  { id: 1, name: 'xray', type: 'xray', is_running: true, is_enabled: true, uptime_seconds: 3600, pid: 1234, restart_count: 0, version: '1.8.0' },
  { id: 2, name: 'singbox', type: 'singbox', is_running: false, is_enabled: true, uptime_seconds: 0, pid: 0, restart_count: 2, version: '1.5.0' },
]

export const mockInbounds = [
  { id: 1, tag: 'vmess-443', protocol: 'vmess', port: 443, listen: '0.0.0.0', is_enabled: true, core_id: 1, core_name: 'xray' },
  { id: 2, tag: 'vless-8443', protocol: 'vless', port: 8443, listen: '0.0.0.0', is_enabled: false, core_id: 1, core_name: 'xray' },
]

export const mockBackups = [
  { id: 1, filename: 'backup_20260401.tar.gz', file_size_bytes: 2048576, backup_type: 'manual', status: 'completed', created_at: '2026-04-01T03:00:00Z' },
  { id: 2, filename: 'backup_20260330.tar.gz', file_size_bytes: 1048576, backup_type: 'scheduled', status: 'completed', created_at: '2026-03-30T03:00:00Z' },
]

export const mockNotifications = [
  { id: 1, type: 'warning', title: 'High traffic', message: 'User alice exceeded 80% quota', read: false, created_at: '2026-04-04T12:00:00Z' },
  { id: 2, type: 'info', title: 'Backup complete', message: 'Daily backup finished', read: true, created_at: '2026-04-04T03:00:00Z' },
]

// ── Auth helpers ──

/** Set up authenticated state in localStorage so ProtectedRoute lets us through.
 *  Uses addInitScript so localStorage is set before any page JS runs. */
export async function setupAuth(page: Page) {
  await page.addInitScript((admin) => {
    const state = {
      state: {
        accessToken: 'mock-access-token',
        refreshToken: 'mock-refresh-token',
        user: admin,
        isAuthenticated: true,
        isLoading: false,
      },
      version: 0,
    }
    localStorage.setItem('auth-storage', JSON.stringify(state))
    localStorage.setItem('accessToken', 'mock-access-token')
    localStorage.setItem('refreshToken', 'mock-refresh-token')
  }, mockAdmin)
}

// Helper: fulfill with JSON
function json(data: unknown) {
  return { status: 200, contentType: 'application/json', body: JSON.stringify(data) }
}

/** Register common API mocks for protected pages.
 *  Uses regex patterns so query parameters are matched correctly. */
export async function mockProtectedApis(page: Page) {
  // /me — called by ProtectedRoute to verify auth
  await page.route(/\/api\/me(\?|$)/, route => route.fulfill(json(mockAdmin)))

  // Dashboard stats
  await page.route(/\/api\/stats\/dashboard/, route => route.fulfill(json({
    total_users: 150, active_users: 120, online_users: 45,
    total_inbounds: 25, cores_running: 2, cores_total: 3,
    total_traffic_bytes: 107374182400,
  })))

  // Stats — traffic overview, top users, connections
  await page.route(/\/api\/stats\/traffic/, route => route.fulfill(json({ data: [] })))
  await page.route(/\/api\/stats\/connections/, route => route.fulfill(json({ connections: [], total: 0 })))
  await page.route(/\/api\/stats\/user\//, route => route.fulfill(json({ data: [] })))

  // System resources
  await page.route(/\/api\/system\/resources/, route => route.fulfill(json({
    cpu_percent: 25, memory_percent: 40, memory_used: 1073741824, memory_total: 4294967296, disk_percent: 55,
  })))

  // System connections
  await page.route(/\/api\/system\/connections/, route => route.fulfill(json({ count: 12 })))

  // WS ticket
  await page.route(/\/api\/ws\/ticket/, route => route.fulfill(json({ ticket: 'mock-ticket' })))

  // Users — backend returns { success, users, total } — match /api/users with optional query params
  await page.route(/\/api\/users(\?|$)/, route => route.fulfill(json({
    success: true, users: mockUsers, total: mockUsers.length, page: 1, page_size: 20, pages: 1,
  })))
  // Individual user endpoints (/api/users/1, /api/users/1/inbounds etc)
  await page.route(/\/api\/users\/\d+/, route => route.fulfill(json({ data: {} })))

  // Cores — backend returns raw array
  await page.route(/\/api\/cores(\?|$)/, route => route.fulfill(json(mockCores)))
  // Individual core endpoints
  await page.route(/\/api\/cores\/\w+/, route => route.fulfill(json({ name: 'xray', is_running: true })))

  // Inbounds — backend returns raw array
  await page.route(/\/api\/inbounds(\?|$)/, route => route.fulfill(json(mockInbounds)))
  await page.route(/\/api\/inbounds\/\d+/, route => route.fulfill(json(mockInbounds[0])))
  await page.route(/\/api\/inbounds\/check-port/, route => route.fulfill(json({ available: true })))

  // Backups — page reads backupsRes.data.data
  await page.route(/\/api\/backups(\?|$)/, route => route.fulfill(json({ data: mockBackups })))
  await page.route(/\/api\/backups\/schedule/, route => route.fulfill(json({ data: { schedule: '', next_run: '' } })))
  await page.route(/\/api\/backups\/\d+/, route => route.fulfill(json({ data: mockBackups[0] })))

  // Notifications
  await page.route(/\/api\/notifications(\?|$)/, route => route.fulfill(json({ data: mockNotifications })))
  await page.route(/\/api\/notifications\/settings/, route => route.fulfill(json({ data: {} })))

  // Settings — Backups page reads settingsRes.data.data as array of {key, value}
  await page.route(/\/api\/settings(\?|$)/, route => route.fulfill(json({
    data: [
      { key: 'panel_name', value: 'Isolate Panel' },
      { key: 'panel_url', value: 'https://panel.example.com' },
      { key: 'backup_retention_count', value: '3' },
    ],
  })))
  await page.route(/\/api\/settings\/monitoring/, route => route.fulfill(json({ data: { mode: 'lite' } })))
  await page.route(/\/api\/settings\/traffic-reset/, route => route.fulfill(json({ data: { schedule: 'disabled' } })))

  // Certificates — page expects { certificates: [], total: 0 }
  await page.route(/\/api\/certificates/, route => route.fulfill(json({ certificates: [], total: 0 })))

  // Outbounds — raw array
  await page.route(/\/api\/outbounds/, route => route.fulfill(json([])))

  // Protocols
  await page.route(/\/api\/protocols/, route => route.fulfill(json([])))

  // WARP
  await page.route(/\/api\/warp\/routes/, route => route.fulfill(json({ data: [] })))
  await page.route(/\/api\/warp\/status/, route => route.fulfill(json({ data: null })))
  await page.route(/\/api\/warp\/presets/, route => route.fulfill(json({ data: {} })))

  // Geo
  await page.route(/\/api\/geo\/rules/, route => route.fulfill(json({ data: [] })))
  await page.route(/\/api\/geo\/countries/, route => route.fulfill(json({ data: [] })))
  await page.route(/\/api\/geo\/categories/, route => route.fulfill(json({ data: [] })))
  await page.route(/\/api\/geo\/databases/, route => route.fulfill(json({ data: [] })))

  // Health
  await page.route(/\/api\/health/, route => route.fulfill(json({ status: 'ok' })))

  // TOTP status
  await page.route(/\/api\/auth\/totp\/status/, route => route.fulfill(json({ totp_enabled: false })))
}
