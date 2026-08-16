import { describe, it, expect } from 'vitest'
import { bufferTrail } from './trailGeometry'
import type { TrailPoint } from '../types'

describe('bufferTrail', () => {
  it('returns null for an empty trail', () => {
    expect(bufferTrail([], 46)).toBeNull()
  })

  it('returns a single polygon feature covering a multi-point trail', () => {
    const points: TrailPoint[] = [
      { lat: 37.9235, lng: -122.5965, recordedAt: '2026-08-13T10:00:00Z' },
      { lat: 37.9240, lng: -122.5970, recordedAt: '2026-08-13T10:00:20Z' },
      { lat: 37.9245, lng: -122.5975, recordedAt: '2026-08-13T10:00:40Z' },
    ]
    const result = bufferTrail(points, 46)
    expect(result).not.toBeNull()
    expect(result!.geometry.type).toMatch(/Polygon|MultiPolygon/)
  })

  it('returns a polygon for a single point (buffered circle)', () => {
    const points: TrailPoint[] = [
      { lat: 37.9235, lng: -122.5965, recordedAt: '2026-08-13T10:00:00Z' },
    ]
    const result = bufferTrail(points, 46)
    expect(result).not.toBeNull()
    expect(result!.geometry.type).toMatch(/Polygon|MultiPolygon/)
  })
})
