import { describe, it, expect } from 'vitest'
import { isWithinRevealRadius } from './geo.ts'

describe('isWithinRevealRadius', () => {
  const visited = [
    { lat: 37.9235, lng: -122.5965 }, // Mount Tam
  ]

  it('returns true for a point at a visited location', () => {
    expect(isWithinRevealRadius(37.9235, -122.5965, visited, 5)).toBe(true)
  })

  it('returns true for a point within the reveal radius', () => {
    // ~1km north of Mount Tam
    expect(isWithinRevealRadius(37.933, -122.5965, visited, 5)).toBe(true)
  })

  it('returns false for a point far outside the reveal radius', () => {
    // San Francisco, ~20km away
    expect(isWithinRevealRadius(37.7749, -122.4194, visited, 5)).toBe(false)
  })

  it('returns false when there are no visited locations', () => {
    expect(isWithinRevealRadius(37.9235, -122.5965, [], 5)).toBe(false)
  })
})
