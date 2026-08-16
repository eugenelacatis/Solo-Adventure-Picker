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
        { adventureId: 'adv-1', name: 'Mount Tam', lat: 37.9235, lng: -122.5965 },
        { adventureId: 'adv-2', name: 'Sunol Regional Wilderness', lat: 37.5136, lng: -121.8310 },
      ]),
    })
  )
  // MapPage now also renders the HUD, which fetches XP and achievements.
  await page.route('http://localhost:8080/xp', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ totalXp: 250, level: 2, nextLevelXp: 300, alreadyVisited: false }),
    })
  )
  await page.route('http://localhost:8080/achievements', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([{ id: 'first-adventure', name: 'First Adventure' }]),
    })
  )
  await page.route('http://localhost:8080/trail', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { lat: 37.9235, lng: -122.5965, recordedAt: '2026-08-13T10:00:00Z' },
        { lat: 37.9240, lng: -122.5970, recordedAt: '2026-08-13T10:00:20Z' },
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

test('renders a fog-reveal polygon from trail points', async ({ page }) => {
  await page.goto('/map')

  await expect(page.locator('.leaflet-container')).toBeVisible()
  await expect(page.locator('path.leaflet-interactive')).toHaveCount(1)
})
