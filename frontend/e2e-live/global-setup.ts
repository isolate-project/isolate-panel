import { execSync } from 'child_process'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'
import { writeFileSync } from 'fs'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const COMPOSE_FILE = resolve(__dirname, '../../docker/docker-compose.test.yml')
const BACKEND_URL = 'http://localhost:8080/health'
const LOGIN_URL = 'http://localhost:8080/api/auth/login'
const TOKEN_FILE = resolve(__dirname, '.auth-tokens.json')
const MAX_WAIT_MS = 120_000
const POLL_INTERVAL_MS = 2_000

async function waitForHealthy(): Promise<void> {
  const start = Date.now()
  while (Date.now() - start < MAX_WAIT_MS) {
    try {
      const res = await fetch(BACKEND_URL)
      if (res.ok) {
        const body = await res.json()
        if (body.status === 'healthy') {
          return
        }
      }
    } catch {
      // Backend not ready yet
    }
    await new Promise(r => setTimeout(r, POLL_INTERVAL_MS))
  }
  throw new Error(`Backend did not become healthy within ${MAX_WAIT_MS / 1000}s`)
}

async function loginOnce(): Promise<void> {
  const res = await fetch(LOGIN_URL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: 'admin' }),
  })

  if (!res.ok) {
    throw new Error(`Login failed in globalSetup: ${res.status} ${await res.text()}`)
  }

  const data = await res.json()
  // Write tokens to a temp file that tests can read
  writeFileSync(TOKEN_FILE, JSON.stringify(data), 'utf-8')
  console.log('[global-setup] Admin login successful, tokens cached.')
}

async function globalSetup(): Promise<void> {
  console.log('\n[global-setup] Starting test backend via Docker Compose...')

  try {
    execSync(
      `docker compose -f "${COMPOSE_FILE}" up -d --build --wait`,
      { stdio: 'inherit', timeout: 300_000 },
    )
  } catch {
    console.log('[global-setup] docker compose --wait exited, polling health manually...')
  }

  console.log('[global-setup] Waiting for backend health check...')
  await waitForHealthy()
  console.log('[global-setup] Backend is healthy!')

  // Login once and cache tokens — avoids rate-limiting in tests
  await loginOnce()
  console.log('')
}

export default globalSetup
