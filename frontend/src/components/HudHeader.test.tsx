import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../context/AuthContext.tsx'
import HudHeader from './HudHeader.tsx'
import type { XpResponse } from '../types.ts'

function renderHud(xp: XpResponse | null) {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <HudHeader xp={xp} />
      </AuthProvider>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('HudHeader', () => {
  it('renders level and XP from the xp prop', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => [] }))

    renderHud({ totalXp: 150, level: 2, nextLevelXp: 300, alreadyVisited: false })

    expect(screen.getByText('Level 2')).toBeInTheDocument()
    expect(screen.getByText('150 / 300 XP')).toBeInTheDocument()
  })

  it('shows default level 1 with no XP data yet', () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => [] }))

    renderHud(null)

    expect(screen.getByText('Level 1')).toBeInTheDocument()
  })

  // Regression test: a brand-new user's /achievements response used to
  // encode as JSON `null` (a nil Go slice), not `[]`, and calling .length
  // on the decoded null crashed the whole page on signup.
  it('does not crash when /achievements responds with null', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => null }))

    renderHud({ totalXp: 0, level: 1, nextLevelXp: 100, alreadyVisited: false })

    await waitFor(() => {
      expect(screen.getByText('Level 1')).toBeInTheDocument()
    })
  })

  it('renders fetched achievement badges', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'first-adventure', name: 'First Adventure' }],
    }))

    renderHud({ totalXp: 100, level: 2, nextLevelXp: 300, alreadyVisited: false })

    await waitFor(() => {
      expect(screen.getByText('First Adventure')).toBeInTheDocument()
    })
  })

  it('updates displayed XP when the xp prop changes', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => [] }))

    const { rerender } = renderHud({ totalXp: 0, level: 1, nextLevelXp: 100, alreadyVisited: false })
    expect(screen.getByText('0 / 100 XP')).toBeInTheDocument()

    rerender(
      <MemoryRouter>
        <AuthProvider>
          <HudHeader xp={{ totalXp: 150, level: 2, nextLevelXp: 300, alreadyVisited: false }} />
        </AuthProvider>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('150 / 300 XP')).toBeInTheDocument()
    })
  })
})
