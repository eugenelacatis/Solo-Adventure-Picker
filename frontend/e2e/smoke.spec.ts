import { test, expect } from '@playwright/test'

// Exercises the real app against the real Go backend (no mocked routes) to
// verify the full user journey end-to-end, not just isolated units: demo
// browsing while logged out, signup, mark visited, journal, logout, then
// login again to confirm progress actually persisted under the account.
test('full user journey: demo, sign up, mark visited, journal, log out, log back in', async ({ page }) => {
  const email = `smoke-${Date.now()}@example.com`

  await page.goto('/')
  await expect(page.getByText('Welcome to Solo Adventure Picker')).toBeVisible()

  // Anonymous demo: reroll works with no backend call and no login.
  const demoHeading = page.getByRole('heading', { level: 2 })
  await expect(demoHeading).toBeVisible()
  await page.getByRole('button', { name: 'Reroll' }).click()
  await expect(demoHeading).toBeVisible()

  // The real app is gated: going straight to /adventure without a session
  // redirects to /login.
  await page.goto('/adventure')
  await expect(page).toHaveURL(/\/login$/)

  // Sign up.
  await page.goto('/signup')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill('hunter2')
  await page.getByRole('button', { name: 'Sign Up' }).click()
  await expect(page).toHaveURL(/\/adventure$/)

  await expect(page.getByRole('heading', { level: 2 })).toBeVisible({ timeout: 10000 })

  await page.getByRole('button', { name: 'Show More' }).click()
  await expect(page.getByPlaceholder('Write about your adventure...')).toBeVisible()

  await page.getByRole('button', { name: 'Mark as Visited' }).click()
  await expect(page.getByText('Marked as visited!')).toBeVisible()

  await page.getByPlaceholder('Write about your adventure...').fill('What a great smoke test hike.')
  await page.getByRole('button', { name: 'Save Journal Entry' }).click()
  await expect(page.getByText('Journal entry saved!')).toBeVisible()

  await page.goto('/map')
  await expect(page.locator('.leaflet-container')).toBeVisible()
  await expect(page.locator('.leaflet-marker-icon')).toHaveCount(1, { timeout: 10000 })

  // Log out via the UI, confirm gated routes are inaccessible again.
  await page.goto('/adventure')
  await page.getByRole('button', { name: 'Log Out' }).click()
  await page.goto('/map')
  await expect(page).toHaveURL(/\/login$/)

  // Log back in and confirm the visited pin persisted under the account.
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill('hunter2')
  await page.getByRole('button', { name: 'Log In' }).click()
  await expect(page).toHaveURL(/\/adventure$/)

  await page.goto('/map')
  await expect(page.locator('.leaflet-marker-icon')).toHaveCount(1, { timeout: 10000 })
})
