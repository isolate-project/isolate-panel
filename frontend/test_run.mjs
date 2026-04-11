import { execSync } from 'child_process';
try {
  execSync('npx playwright test e2e-live/users.spec.ts -g "should create a new user" --project chromium', { stdio: 'inherit', env: { ...process.env, DEBUG: 'pw:api' } });
} catch (e) {
  process.exit(1);
}
