import { Capacitor } from '@capacitor/core'
import { Geolocation } from '@capacitor/geolocation'
import { API_BASE } from '../api.ts'
import type { Adventure, TrailPoint, TrailUploadResponse } from '../types.ts'

const SAMPLE_INTERVAL_MS = 20_000
const FLUSH_INTERVAL_MS = 60_000
const FLUSH_SIZE_THRESHOLD = 20
const BUFFER_STORAGE_KEY = 'sap-trail-buffer'

let buffer: TrailPoint[] = loadBuffer()
let sampleTimer: ReturnType<typeof setInterval> | null = null
let lastFlushAt = Date.now()
let onNewlyVisited: ((adventures: Adventure[]) => void) | undefined

function loadBuffer(): TrailPoint[] {
  try {
    const raw = localStorage.getItem(BUFFER_STORAGE_KEY)
    return raw ? (JSON.parse(raw) as TrailPoint[]) : []
  } catch {
    return []
  }
}

function saveBuffer(): void {
  localStorage.setItem(BUFFER_STORAGE_KEY, JSON.stringify(buffer))
}

// shouldFlush implements the spec's "60s timer OR 20-point buffer,
// whichever comes first" batch trigger, as a pure function so the
// threshold logic is testable without real timers or geolocation.
export function shouldFlush(bufferSize: number, msSinceLastFlush: number): boolean {
  return bufferSize >= FLUSH_SIZE_THRESHOLD || msSinceLastFlush >= FLUSH_INTERVAL_MS
}

// flushBuffer uploads points to POST /trail and returns the parsed
// response. Does not mutate the caller's buffer — the caller clears it
// only after this resolves successfully, so a failed flush can retry with
// the same points still queued.
export async function flushBuffer(
  points: TrailPoint[],
  apiBase: string
): Promise<TrailUploadResponse> {
  const res = await fetch(`${apiBase}/trail`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ points }),
  })
  if (!res.ok) {
    throw new Error(`POST /trail failed with status ${res.status}`)
  }
  return res.json()
}

// resolveFlushedBuffer computes the buffer that remains after a flush of
// `flushedCount` points succeeds. It splices off only that leading prefix
// rather than clearing the whole buffer, so points pushed onto `current`
// during the flush's in-flight await (which land after `flushedCount`)
// survive instead of being silently discarded.
export function resolveFlushedBuffer(
  current: TrailPoint[],
  flushedCount: number
): TrailPoint[] {
  return current.slice(flushedCount)
}

async function sampleAndMaybeFlush(): Promise<void> {
  try {
    const position = await Geolocation.getCurrentPosition()
    buffer.push({
      lat: position.coords.latitude,
      lng: position.coords.longitude,
      recordedAt: new Date(position.timestamp).toISOString(),
    })
    saveBuffer()
  } catch {
    // GPS read or buffer persist failed; skip this tick and retry next interval.
    return
  }

  if (shouldFlush(buffer.length, Date.now() - lastFlushAt)) {
    const toFlush = buffer.slice()
    try {
      const response = await flushBuffer(toFlush, API_BASE)
      buffer = resolveFlushedBuffer(buffer, toFlush.length)
      lastFlushAt = Date.now()
      saveBuffer()
      if (response.newlyVisited.length > 0) {
        onNewlyVisited?.(response.newlyVisited)
      }
    } catch {
      // Leave buffer intact; next sample's shouldFlush check will retry.
    }
  }
}

// startTracking begins foreground GPS sampling. No-op on the web build —
// only native iOS (Capacitor.isNativePlatform()) actually samples, so
// calling this unconditionally from UI code is always safe. The optional
// onNewlyVisited callback fires after a flush yields newly-visited
// adventures, so callers can surface a discovery notice.
export function startTracking(newlyVisitedCallback?: (adventures: Adventure[]) => void): void {
  if (!Capacitor.isNativePlatform() || sampleTimer !== null) {
    return
  }
  onNewlyVisited = newlyVisitedCallback
  lastFlushAt = Date.now()
  sampleTimer = setInterval(sampleAndMaybeFlush, SAMPLE_INTERVAL_MS)
}

export function stopTracking(): void {
  if (sampleTimer !== null) {
    clearInterval(sampleTimer)
    sampleTimer = null
  }
  onNewlyVisited = undefined
}

export function isTracking(): boolean {
  return sampleTimer !== null
}
