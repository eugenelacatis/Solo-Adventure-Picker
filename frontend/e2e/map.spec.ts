import { test, expect } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  // /map is login-gated; mock /auth/me so RequireAuth treats us as signed in
  // without needing a real signup/login round trip for this test.
  await page.route('http://localhost:8080/auth/me', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ userId: 'test-user', email: 'test@example.com' }),
    })
  )
  await page.route('http://localhost:8080/visited', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { adventureId: 'adv-1', lat: 37.9235, lng: -122.5965 },
        { adventureId: 'adv-2', lat: 37.5136, lng: -121.8310 },
      ]),
    })
  )
})

test('renders a real Leaflet map with a marker per visited adventure', async ({ page }) => {
  await page.goto('/map')

  const map = page.locator('.leaflet-container')
  await expect(map).toBeVisible()

  // Real layout math (pane sizing, tile positioning) is exactly what jsdom
  // can't provide, so this assertion only means something in a real browser.
  await expect(page.locator('.leaflet-tile-pane')).toHaveCount(1)

  const markers = page.locator('.leaflet-marker-icon')
  await expect(markers).toHaveCount(2)
})

test('renders the map with zero markers when nothing has been visited', async ({ page }) => {
  await page.unroute('http://localhost:8080/visited')
  await page.route('http://localhost:8080/visited', route =>
    route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
  )

  await page.goto('/map')

  await expect(page.locator('.leaflet-container')).toBeVisible()
  await expect(page.locator('.leaflet-marker-icon')).toHaveCount(0)
})
