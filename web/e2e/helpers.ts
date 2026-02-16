import { type Page, expect } from '@playwright/test'

/** Test user credentials for the admin account created during setup. */
export const TEST_USER = {
  username: 'admin',
  email: 'admin@test.local',
  password: 'TestPass123!',
}

/** Auth state storage file used by Playwright to persist login across tests. */
export const AUTH_STATE_PATH = 'e2e/.auth/state.json'

/**
 * Complete the setup wizard to create the initial admin account.
 * This must run once before any authenticated tests.
 *
 * The setup wizard has 4 steps:
 * 1. Account creation (username, email, password, confirm password)
 * 2. Network interface selection (auto-detect default)
 * 3. Theme preference (dark/light)
 * 4. Summary & complete
 *
 * Note: The account is actually created and logged in at the step 1 -> 2
 * transition (POST /api/v1/auth/setup + POST /api/v1/auth/login).
 */
export async function completeSetup(page: Page): Promise<void> {
  await page.goto('/setup')

  // Wait for the setup status check to complete
  await expect(page.getByText('Welcome to SubNetree')).toBeVisible({ timeout: 15_000 })

  // Step 1: Account creation
  await page.getByLabel('Username').fill(TEST_USER.username)
  await page.getByLabel('Email').fill(TEST_USER.email)
  // Password field has id="password", Confirm Password has id="confirmPassword"
  await page.locator('#password').fill(TEST_USER.password)
  await page.locator('#confirmPassword').fill(TEST_USER.password)
  await page.getByRole('button', { name: 'Next' }).click()

  // Step 2: Network interface -- accept default (auto-detect) and proceed
  // Wait for step 2 to load (the heading changes)
  await expect(page.getByText('Select Network Interface')).toBeVisible({ timeout: 15_000 })
  await page.getByRole('button', { name: 'Next' }).click()

  // Step 3: Theme preference -- accept default and proceed
  await expect(page.getByText('Theme Preference')).toBeVisible()
  await page.getByRole('button', { name: 'Next' }).click()

  // Step 4: Summary -- complete setup
  await expect(page.getByText('Setup Summary')).toBeVisible()
  await page.getByRole('button', { name: 'Complete Setup' }).click()

  // Should redirect to dashboard
  await page.waitForURL(/\/dashboard/, { timeout: 15_000 })
}

/**
 * Log in with test credentials.
 * Assumes setup has already been completed.
 */
export async function login(page: Page): Promise<void> {
  await page.goto('/login')

  // Wait for setup status check to complete (login page checks if setup is needed)
  await expect(page.getByText('Sign in to SubNetree')).toBeVisible({ timeout: 15_000 })

  await page.getByLabel('Username').fill(TEST_USER.username)
  await page.getByLabel('Password').fill(TEST_USER.password)
  await page.getByRole('button', { name: 'Sign in' }).click()

  // Wait for successful redirect to dashboard
  await page.waitForURL(/\/dashboard/, { timeout: 10_000 })
}
