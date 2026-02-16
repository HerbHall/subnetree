import { test, expect } from '@playwright/test'

/**
 * Dashboard E2E tests.
 *
 * These run in the "authenticated" project with pre-loaded auth state.
 * The Go server should be running with NV_SEED_DATA=true for meaningful data.
 */
test.describe('Dashboard', () => {
  test('dashboard page loads with heading', async ({ page }) => {
    await page.goto('/dashboard')

    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  })

  test('dashboard displays device statistics', async ({ page }) => {
    await page.goto('/dashboard')

    // At minimum, the dashboard should show device count info
    await expect(page.getByText('Total Devices')).toBeVisible()
  })

  test('dashboard has Scan Network button', async ({ page }) => {
    await page.goto('/dashboard')

    const scanButton = page.getByRole('button', { name: /scan network/i })
    await expect(scanButton).toBeVisible()
  })

  test('dashboard has navigation links', async ({ page }) => {
    await page.goto('/dashboard')

    // The sidebar should have navigation to key pages
    await expect(page.getByRole('link', { name: /devices/i })).toBeVisible()
    await expect(page.getByRole('link', { name: /topology/i })).toBeVisible()
  })
})
