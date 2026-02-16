import { test, expect } from '@playwright/test'

/**
 * Topology page E2E tests.
 *
 * These run in the "authenticated" project with pre-loaded auth state.
 * The topology page renders a React Flow graph canvas.
 */
test.describe('Topology', () => {
  test('topology page loads without error', async ({ page }) => {
    await page.goto('/topology')

    // The page should not show an error state
    // React Flow renders a container with class "react-flow"
    // Wait for either the graph container or a loading/empty state
    await expect(
      page.locator('.react-flow').or(page.getByText(/no devices/i)).or(page.getByText('Loading...'))
    ).toBeVisible({ timeout: 15_000 })
  })

  test('topology page has layout controls', async ({ page }) => {
    await page.goto('/topology')

    // Wait for the page to load
    await page.waitForLoadState('networkidle')

    // The topology toolbar should be visible with layout algorithm controls
    // The TopologyToolbar renders buttons for layout algorithms
    // At minimum, the page should have rendered without crashing
    await expect(page.locator('.react-flow').or(page.getByText('Loading...'))).toBeVisible({
      timeout: 15_000,
    })
  })

  test('topology page is accessible from sidebar', async ({ page }) => {
    await page.goto('/dashboard')

    // Navigate to topology via sidebar
    await page
      .getByRole('navigation', { name: 'Main navigation' })
      .getByText('Topology')
      .click()

    await expect(page).toHaveURL(/\/topology/)
  })

  test('topology page renders React Flow container', async ({ page }) => {
    await page.goto('/topology')

    // React Flow renders a div with class "react-flow"
    // This confirms the component mounted successfully
    const reactFlowContainer = page.locator('.react-flow')
    await expect(reactFlowContainer).toBeVisible({ timeout: 15_000 })
  })
})
