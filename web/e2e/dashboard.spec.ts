import { test, expect } from '@playwright/test'

/**
 * Dashboard E2E tests.
 *
 * These run in the "authenticated" project with pre-loaded auth state.
 * The Go server should be running with NV_SEED_DATA=true for meaningful data.
 */
test.describe('Dashboard', () => {
  test('dashboard page loads with heading and description', async ({ page }) => {
    await page.goto('/dashboard')

    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
    await expect(page.getByText('Network overview and quick actions')).toBeVisible()
  })

  test('dashboard displays stat cards', async ({ page }) => {
    await page.goto('/dashboard')

    // The dashboard always renders 4 stat cards regardless of data
    await expect(page.getByText('Total Devices')).toBeVisible()
    await expect(page.getByText('Online')).toBeVisible()
    await expect(page.getByText('Offline')).toBeVisible()
    await expect(page.getByText('Degraded')).toBeVisible()
  })

  test('dashboard has Scan Network button', async ({ page }) => {
    await page.goto('/dashboard')

    const scanButton = page.getByRole('button', { name: /scan network/i })
    await expect(scanButton).toBeVisible()
  })

  test('dashboard displays Scout Agents widget', async ({ page }) => {
    await page.goto('/dashboard')

    await expect(page.getByText('Scout Agents')).toBeVisible()
  })

  test('dashboard displays Active Alerts widget', async ({ page }) => {
    await page.goto('/dashboard')

    await expect(page.getByText('Active Alerts')).toBeVisible()
  })

  test('dashboard has navigation quick links', async ({ page }) => {
    await page.goto('/dashboard')

    // Quick link cards at the bottom of the dashboard
    const devicesLink = page.getByText('View and manage all discovered devices')
    const topologyLink = page.getByText('Visual topology of your network')
    const settingsLink = page.getByText('Configure application settings')

    await expect(devicesLink).toBeVisible()
    await expect(topologyLink).toBeVisible()
    await expect(settingsLink).toBeVisible()
  })

  test('dashboard auto-refresh controls are visible', async ({ page }) => {
    await page.goto('/dashboard')

    // The auto-refresh control shows interval options
    await expect(page.getByRole('button', { name: /15s/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /30s/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /1m/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /5m/ })).toBeVisible()
  })
})
