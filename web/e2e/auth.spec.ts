import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test('login page renders the sign-in form', async ({ page }) => {
    await page.goto('/login')

    // Verify the heading is visible
    await expect(page.getByText('Sign in to SubNetree')).toBeVisible()

    // Verify form fields exist
    await expect(page.getByLabel('Username')).toBeVisible()
    await expect(page.getByLabel('Password')).toBeVisible()

    // Verify submit button
    await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
  })

  test('login page shows the SubNetree logo', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByAlt('SubNetree')).toBeVisible()
  })

  test('login page shows description text', async ({ page }) => {
    await page.goto('/login')
    await expect(
      page.getByText('Enter your credentials to access the dashboard.')
    ).toBeVisible()
  })

  test('submitting empty form shows browser validation', async ({ page }) => {
    await page.goto('/login')

    // Both fields have the "required" attribute, so the browser prevents
    // submission before our JS handler runs. Verify the fields are required.
    const usernameInput = page.getByLabel('Username')
    const passwordInput = page.getByLabel('Password')

    await expect(usernameInput).toHaveAttribute('required', '')
    await expect(passwordInput).toHaveAttribute('required', '')
  })

  test('invalid credentials show error message', async ({ page }) => {
    await page.goto('/login')

    await page.getByLabel('Username').fill('nonexistent_user')
    await page.getByLabel('Password').fill('wrong_password')
    await page.getByRole('button', { name: 'Sign in' }).click()

    // The login API call should fail and display an error alert
    const errorAlert = page.getByRole('alert')
    await expect(errorAlert).toBeVisible({ timeout: 10_000 })
  })

  test('password visibility toggle works', async ({ page }) => {
    await page.goto('/login')

    const passwordInput = page.getByLabel('Password')
    await passwordInput.fill('secret123')

    // Initially the password is hidden
    await expect(passwordInput).toHaveAttribute('type', 'password')

    // Click the show password button
    await page.getByLabel('Show password').click()
    await expect(passwordInput).toHaveAttribute('type', 'text')

    // Click again to hide
    await page.getByLabel('Hide password').click()
    await expect(passwordInput).toHaveAttribute('type', 'password')
  })
})
