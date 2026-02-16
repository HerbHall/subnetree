import { chromium, type FullConfig } from '@playwright/test'
import { TEST_USER, AUTH_STATE_PATH, completeSetup } from './helpers'

/**
 * Playwright global setup: runs once before all test projects.
 *
 * 1. Checks if the server needs initial setup (GET /api/v1/auth/setup/status)
 * 2. If setup is required, completes the setup wizard (creates admin account)
 * 3. Logs in and saves the browser storage state (localStorage with JWT tokens)
 *
 * Authenticated test projects use the saved storage state so each test
 * starts already logged in.
 */
async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0]?.use?.baseURL || 'http://localhost:5173'

  const browser = await chromium.launch()
  const context = await browser.newContext({ baseURL })
  const page = await context.newPage()

  try {
    // Check if setup is required
    const setupStatusResponse = await page.request.get('/api/v1/auth/setup/status')

    let needsSetup = false
    if (setupStatusResponse.ok()) {
      const status = await setupStatusResponse.json()
      needsSetup = status.setup_required === true
    }

    if (needsSetup) {
      // Complete the setup wizard (creates account + auto-logs in)
      await completeSetup(page)
    } else {
      // Setup already done -- just log in
      await page.goto('/login')

      // Wait for setup status check
      await page.waitForSelector('text=Sign in to SubNetree', { timeout: 15_000 })

      await page.getByLabel('Username').fill(TEST_USER.username)
      await page.locator('#password').fill(TEST_USER.password)
      await page.getByRole('button', { name: 'Sign in' }).click()
      await page.waitForURL(/\/dashboard/, { timeout: 10_000 })
    }

    // Save the authenticated storage state (localStorage with JWT tokens)
    await context.storageState({ path: AUTH_STATE_PATH })
  } finally {
    await browser.close()
  }
}

export default globalSetup
