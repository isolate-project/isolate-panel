const { execSync } = require('child_process');
try {
  execSync('npx playwright test e2e-live/users.spec.ts --config playwright.live.config.ts', { stdio: 'pipe' });
} catch (e) {
  console.log("TEST FAILED, listing test-results dir:");
  execSync('ls -la test-results', { stdio: 'inherit' });
}
