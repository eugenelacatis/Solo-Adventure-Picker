import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../context/AuthContext.tsx'
import AdventurePage from './AdventurePage.tsx'

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/adventure?region=bay-area']}>
      <AuthProvider>
        <AdventurePage />
      </AuthProvider>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('AdventurePage', () => {
  it('displays the fetched adventure name and XP', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 1, name: 'mount tamalpais', region: 'bay-area', xpValue: 150 }),
    }))

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Mount Tamalpais')).toBeInTheDocument()
    })
    expect(screen.getByText('+150XP')).toBeInTheDocument()
  })

  it('shows an error message when the fetch fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: 'No matching adventure found.', details: 'Womp womp.' }),
    }))

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Womp womp.')).toBeInTheDocument()
    })
  })

  it('submits a journal entry and shows a confirmation', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes('/auth/me')) {
        return Promise.resolve({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
      }
      if (String(url).includes('/random')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 1, name: 'Mount Tamalpais', region: 'bay-area', xpValue: 150 }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({ totalXp: 25, level: 1 }) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Mount Tamalpais')).toBeInTheDocument()
    })

    const textarea = screen.getByPlaceholderText(/write about your adventure/i)
    await user.type(textarea, 'Great hike today!')
    await user.click(screen.getByRole('button', { name: /save journal entry/i }))

    await waitFor(() => {
      expect(screen.getByText(/journal entry saved/i)).toBeInTheDocument()
    })

    const journalCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/journal'))
    expect(journalCall).toBeTruthy()
    const requestBody = JSON.parse(journalCall![1].body)
    expect(requestBody.text).toBe('Great hike today!')
  })

  it('marks an adventure as visited and shows a confirmation', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes('/auth/me')) {
        return Promise.resolve({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
      }
      if (String(url).includes('/random')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 1, name: 'Mount Tamalpais', region: 'bay-area', xpValue: 150, lat: 37.9235, lng: -122.5965 }) })
      }
      if (String(url).includes('/visited')) {
        return Promise.resolve({ ok: true, json: async () => ({ totalXp: 150, level: 2, nextLevelXp: 300, alreadyVisited: false }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({ totalXp: 150, level: 2, nextLevelXp: 300, alreadyVisited: false }) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Mount Tamalpais')).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /mark as visited/i }))

    await waitFor(() => {
      expect(screen.getByText('Marked as visited!')).toBeInTheDocument()
    })

    const visitCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/visited') && call[1]?.method === 'POST')
    expect(visitCall).toBeTruthy()
    const requestBody = JSON.parse(visitCall![1].body)
    expect(requestBody.adventureId).toBe('1')
    expect(requestBody.lat).toBeUndefined()
    expect(requestBody.lng).toBeUndefined()
  })

  it('shows remaining reroll count and disables the button when exhausted', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes('/auth/me')) {
        return Promise.resolve({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
      }
      if (String(url).includes('/random')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 1, name: 'Mount Tamalpais', region: 'bay-area', xpValue: 150 }) })
      }
      if (String(url).includes('/reroll-status')) {
        return Promise.resolve({ ok: true, json: async () => ({ rerollTokens: 0, rerollResetAt: '2026-08-14T00:00:00Z' }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({ totalXp: 25, level: 1 }) })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('0 rerolls left today')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /no rerolls left today/i })).toBeDisabled()
  })

  it('shows an already-visited notice when the backend reports a repeat visit', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (String(url).includes('/auth/me')) {
        return Promise.resolve({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
      }
      if (String(url).includes('/random')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 1, name: 'Mount Tamalpais', region: 'bay-area', xpValue: 150 }) })
      }
      if (String(url).includes('/visited') && init?.method === 'POST') {
        return Promise.resolve({ ok: true, json: async () => ({ totalXp: 150, level: 2, nextLevelXp: 300, alreadyVisited: true }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({ totalXp: 150, level: 2, nextLevelXp: 300, alreadyVisited: false }) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Mount Tamalpais')).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /mark as visited/i }))

    await waitFor(() => {
      expect(screen.getByText(/no new xp/i)).toBeInTheDocument()
    })
  })
})
