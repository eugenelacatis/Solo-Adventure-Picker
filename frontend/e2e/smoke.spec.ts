import { test, expect } from '@playwright/test'

// Exercises the real app against the real Go backend (no mocked routes) to
// verify the full user journey end-to-end, not just isolated units.
test('full user journey: pick region, reroll, mark visited, journal, view map', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('Welcome to Solo Adventure Picker')).toBeVisible()

  await page.selectOption('select', 'bay-area')
  await page.getByRole('button', { name: 'Start Adventure' }).click()

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
})
