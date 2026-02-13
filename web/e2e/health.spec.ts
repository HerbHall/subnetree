import { test, expect } from '@playwright/test'

test.describe('Health - Smoke tests', () => {
  test('page loads without crashing', async ({ page }) => {
    const response = await page.goto('/')
    expect(response?.status()).toBeLessThan(500)
  })

  test('root redirects to /login when unauthenticated', async ({ page }) => {
    await page.goto('/')
    // The ProtectedRoute component redirects unauthenticated users to /login
    await expect(page).toHaveURL(/\/login/)
  })

  test('page has a title', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveTitle(/.+/)
  })
})
