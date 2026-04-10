import { execSync } from 'child_process'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const COMPOSE_FILE = resolve(__dirname, '../../docker/docker-compose.test.yml')

async function globalTeardown(): Promise<void> {
  console.log('\n[global-teardown] Stopping test backend...')
  try {
    execSync(
      `docker compose -f "${COMPOSE_FILE}" down -v --remove-orphans`,
      { stdio: 'inherit', timeout: 60_000 },
    )
  } catch (e) {
    console.error('[global-teardown] Failed to stop containers:', e)
  }
  console.log('[global-teardown] Done.\n')
}

export default globalTeardown
