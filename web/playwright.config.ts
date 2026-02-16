import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright E2E test configuration for SubNetree Dashboard.
 *
 * Two test projects:
 * - "unauthenticated": Tests that run without login (auth.spec.ts, health.spec.ts, navigation.spec.ts)
 * - "authenticated":   Tests that require a logged-in session (dashboard, devices, topology, etc.)
 *
 * The global setup completes the setup wizard (if needed) and saves
 * the authenticated storage state for reuse by the authenticated project.
 *
 * Run tests:   pnpm test:e2e
 * UI mode:     pnpm test:e2e:ui
 * Debug:       npx playwright test --debug
 */
export default defineConfig({
  testDir: './e2e',
  outputDir: './test-results',

  /* Global setup: complete wizard + save auth state */
  globalSetup: './e2e/global-setup.ts',

  /* Fail the build on CI if you accidentally left test.only in the source code */
  forbidOnly: !!process.env.CI,

  /* Retry once in CI, never locally */
  retries: process.env.CI ? 1 : 0,

  /* Single worker in CI for stability, parallel locally */
  workers: process.env.CI ? 1 : undefined,

  /* Reporter: list for CI, html for local debugging */
  reporter: process.env.CI ? 'list' : 'html',

  /* Shared settings for all tests */
  use: {
    baseURL: 'http://localhost:5173',

    /* Collect trace on first retry for debugging CI failures */
    trace: 'on-first-retry',

    /* Screenshot on failure */
    screenshot: 'only-on-failure',

    /* Action timeout: 10 seconds */
    actionTimeout: 10_000,
  },

  /* Test timeout: 30 seconds */
  timeout: 30_000,

  projects: [
    {
      name: 'unauthenticated',
      use: { ...devices['Desktop Chrome'] },
      testMatch: /\/(auth|health|navigation)\.spec\.ts$/,
    },
    {
      name: 'authenticated',
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'e2e/.auth/state.json',
      },
      testMatch: /\/(setup-wizard|dashboard|devices|topology)\.spec\.ts$/,
    },
  ],

  /* Start Vite dev server before running tests */
  webServer: {
    command: 'pnpm dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
})
