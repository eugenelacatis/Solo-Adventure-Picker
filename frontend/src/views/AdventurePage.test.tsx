import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import AdventurePage from './AdventurePage.tsx'

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/adventure?region=bay-area']}>
      <AdventurePage />
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
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 1, name: 'Mount Tamalpais', region: 'bay-area', xpValue: 150 }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ totalXp: 25, level: 1 }),
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

    const journalCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/journal/'))
    expect(journalCall).toBeTruthy()
    const requestBody = JSON.parse(journalCall![1].body)
    expect(requestBody.text).toBe('Great hike today!')
  })

  it('marks an adventure as visited and shows a confirmation', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 1, name: 'Mount Tamalpais', region: 'bay-area', xpValue: 150, lat: 37.9235, lng: -122.5965 }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ totalXp: 150, level: 2 }),
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

    const visitCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/xp/') && String(call[0]).includes('/add'))
    expect(visitCall).toBeTruthy()
    const requestBody = JSON.parse(visitCall![1].body)
    expect(requestBody.adventureId).toBe('1')
    expect(requestBody.lat).toBe(37.9235)
    expect(requestBody.lng).toBe(-122.5965)
  })
})
