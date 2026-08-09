import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../context/AuthContext.tsx'
import LoginPage from './LoginPage.tsx'

function renderPage() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <LoginPage />
      </AuthProvider>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('LoginPage', () => {
  it('logs in with entered credentials', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'Authentication required.' }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderPage()

    await waitFor(() => screen.getByLabelText(/email/i))

    await user.type(screen.getByLabelText(/email/i), 'hiker@example.com')
    await user.type(screen.getByLabelText(/password/i), 'hunter2')
    await user.click(screen.getByRole('button', { name: /log in/i }))

    await waitFor(() => {
      const loginCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/auth/login'))
      expect(loginCall).toBeTruthy()
    })
  })

  it('shows an error message on failed login', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'Authentication required.' }) })
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'Invalid email or password.' }) })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderPage()

    await waitFor(() => screen.getByLabelText(/email/i))

    await user.type(screen.getByLabelText(/email/i), 'hiker@example.com')
    await user.type(screen.getByLabelText(/password/i), 'wrong')
    await user.click(screen.getByRole('button', { name: /log in/i }))

    await waitFor(() => {
      expect(screen.getByText(/invalid email or password/i)).toBeInTheDocument()
    })
  })
})
