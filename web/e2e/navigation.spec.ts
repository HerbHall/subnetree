import { test, expect } from '@playwright/test'

test.describe('Navigation - Unauthenticated', () => {
  test('accessing /dashboard redirects to /login', async ({ page }) => {
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/login/)
  })

  test('accessing /devices redirects to /login', async ({ page }) => {
    await page.goto('/devices')
    await expect(page).toHaveURL(/\/login/)
  })

  test('accessing /topology redirects to /login', async ({ page }) => {
    await page.goto('/topology')
    await expect(page).toHaveURL(/\/login/)
  })

  test('accessing /settings redirects to /login', async ({ page }) => {
    await page.goto('/settings')
    await expect(page).toHaveURL(/\/login/)
  })

  test('accessing /about redirects to /login', async ({ page }) => {
    await page.goto('/about')
    await expect(page).toHaveURL(/\/login/)
  })

  test('non-existent route shows not-found page', async ({ page }) => {
    await page.goto('/this-page-does-not-exist')
    // The NotFoundPage should render for unmatched routes
    // Checking that we did NOT get redirected to /login (it's a catch-all route, not protected)
    await expect(page).not.toHaveURL(/\/login/)
  })
})

test.describe('Navigation - Login page', () => {
  test('login page is accessible without authentication', async ({ page }) => {
    const response = await page.goto('/login')
    expect(response?.status()).toBe(200)
  })

  test('setup page is accessible without authentication', async ({ page }) => {
    const response = await page.goto('/setup')
    expect(response?.status()).toBe(200)
  })
})
