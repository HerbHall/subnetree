import { test, expect } from '@playwright/test'

/**
 * Devices page E2E tests.
 *
 * These run in the "authenticated" project with pre-loaded auth state.
 * With seed data, there should be devices in the list.
 */
test.describe('Devices', () => {
  test('devices page loads with heading', async ({ page }) => {
    await page.goto('/devices')

    await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible()
  })

  test('devices page shows device count', async ({ page }) => {
    await page.goto('/devices')

    // The subtitle shows device count like "20 devices" or "0 devices"
    // Use .first() because subnet grouping shows per-group counts too
    await expect(page.getByText(/\d+ devices?/).first()).toBeVisible({ timeout: 10_000 })
  })

  test('devices page has search input', async ({ page }) => {
    await page.goto('/devices')

    // The search input should be present
    const searchInput = page.getByPlaceholder(/search/i)
    await expect(searchInput).toBeVisible()
  })

  test('devices page has view mode toggles', async ({ page }) => {
    await page.goto('/devices')

    // View mode buttons: grid, list, table (icons from lucide)
    // The page has LayoutGrid, List, and Table2 icons as view mode buttons
    // They are accessible via aria-labels or button roles
    const viewButtons = page.locator('button').filter({ has: page.locator('svg') })
    // At minimum, the scan button and view mode toggles should exist
    expect(await viewButtons.count()).toBeGreaterThan(0)
  })

  test('devices page has scan button', async ({ page }) => {
    await page.goto('/devices')

    // The ScanButton component renders a button with scan functionality
    const scanButton = page.getByRole('button', { name: /scan/i })
    await expect(scanButton).toBeVisible()
  })

  test('sidebar navigation links to devices page', async ({ page }) => {
    await page.goto('/dashboard')

    // Navigate to devices via sidebar
    await page.getByRole('navigation', { name: 'Main navigation' }).getByText('Devices').click()
    await expect(page).toHaveURL(/\/devices/)
    await expect(page.getByRole('heading', { name: 'Devices' })).toBeVisible()
  })
})
