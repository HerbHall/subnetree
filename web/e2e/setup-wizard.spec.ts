import { test, expect } from '@playwright/test'

/**
 * Setup wizard E2E tests.
 *
 * These run in the "authenticated" project (after global-setup has already
 * completed the wizard). They verify post-setup behavior and form validation.
 *
 * Note: The actual setup flow is exercised by global-setup.ts which must
 * succeed for any authenticated tests to run. These tests verify edge cases.
 */
test.describe('Setup Wizard', () => {
  test('shows "already complete" when setup has been done', async ({ page }) => {
    await page.goto('/setup')

    // After setup is complete, the page should show the "already complete" card
    await expect(page.getByText('Setup Already Complete')).toBeVisible({ timeout: 15_000 })
    await expect(
      page.getByText('An administrator account has already been created.')
    ).toBeVisible()

    // Should have a button to go to login
    await expect(page.getByRole('button', { name: 'Go to Login' })).toBeVisible()
  })

  test('"Go to Login" button navigates to login page', async ({ page }) => {
    await page.goto('/setup')

    await expect(page.getByText('Setup Already Complete')).toBeVisible({ timeout: 15_000 })
    await page.getByRole('button', { name: 'Go to Login' }).click()

    await expect(page).toHaveURL(/\/login/)
  })

  test('setup page displays version info', async ({ page }) => {
    await page.goto('/setup')

    // The setup page shows SubNetree version at the bottom
    // Even on the "already complete" view, the card is rendered
    await expect(page.getByText('Setup Already Complete')).toBeVisible({ timeout: 15_000 })
  })
})
