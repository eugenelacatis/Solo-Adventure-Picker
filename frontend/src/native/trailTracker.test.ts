import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { shouldFlush, flushBuffer } from './trailTracker'
import type { TrailPoint } from '../types'

describe('shouldFlush', () => {
  it('is false when neither threshold is met', () => {
    expect(shouldFlush(5, 10_000)).toBe(false)
  })

  it('is true when buffer size reaches 20', () => {
    expect(shouldFlush(20, 0)).toBe(true)
  })

  it('is true when 60 seconds have elapsed', () => {
    expect(shouldFlush(1, 60_000)).toBe(true)
  })

  it('is false just under both thresholds', () => {
    expect(shouldFlush(19, 59_999)).toBe(false)
  })
})

describe('flushBuffer', () => {
  const points: TrailPoint[] = [
    { lat: 37.9235, lng: -122.5965, recordedAt: '2026-08-13T10:00:00Z' },
  ]

  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('POSTs the buffered points to {apiBase}/trail and returns the parsed response', async () => {
    const mockResponse = { newlyVisited: [] }
    ;(fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
      ok: true,
      json: async () => mockResponse,
    })

    const result = await flushBuffer(points, 'http://localhost:8080')

    expect(fetch).toHaveBeenCalledWith(
      'http://localhost:8080/trail',
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        body: JSON.stringify({ points }),
      })
    )
    expect(result).toEqual(mockResponse)
  })

  it('rejects when the server responds with a non-ok status', async () => {
    ;(fetch as ReturnType<typeof vi.fn>).mockResolvedValue({ ok: false, status: 500 })

    await expect(flushBuffer(points, 'http://localhost:8080')).rejects.toThrow()
  })
})
