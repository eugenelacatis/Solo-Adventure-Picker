import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import HomePage from './HomePage.tsx'
import { demoAdventures } from '../data/demoAdventures.ts'

function renderPage() {
  return render(
    <MemoryRouter>
      <HomePage />
    </MemoryRouter>
  )
}

describe('HomePage demo mode', () => {
  it('shows a demo adventure card with no backend call', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    expect(fetchMock).not.toHaveBeenCalled()
    const namesOnPage = demoAdventures.map(a => a.name)
    const heading = screen.getByRole('heading', { level: 2 })
    expect(namesOnPage.some(name => heading.textContent?.includes(name))).toBe(true)
  })

  it('rerolling only cycles through demo adventures matching the selected region', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.selectOptions(screen.getByRole('combobox'), 'north-bay')

    const northBayNames = demoAdventures
      .filter(a => a.region === 'north-bay')
      .map(a => a.name)

    for (let i = 0; i < 10; i++) {
      await user.click(screen.getByRole('button', { name: /reroll/i }))
      const heading = screen.getByRole('heading', { level: 2 })
      expect(northBayNames.some(name => heading.textContent?.includes(name))).toBe(true)
    }
  })

  it('links to signup and login', () => {
    renderPage()
    expect(screen.getByRole('link', { name: /sign up/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /log in/i })).toBeInTheDocument()
  })
})
