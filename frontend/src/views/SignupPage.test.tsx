import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../context/AuthContext.tsx'
import SignupPage from './SignupPage.tsx'

function renderPage() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <SignupPage />
      </AuthProvider>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('SignupPage', () => {
  it('signs up with entered credentials', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'Authentication required.' }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderPage()

    await waitFor(() => screen.getByLabelText(/email/i))

    await user.type(screen.getByLabelText(/email/i), 'hiker@example.com')
    await user.type(screen.getByLabelText(/password/i), 'hunter2')
    await user.click(screen.getByRole('button', { name: /sign up/i }))

    await waitFor(() => {
      const signupCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/auth/signup'))
      expect(signupCall).toBeTruthy()
    })
  })

  it('shows an error message on failed signup', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'Authentication required.' }) })
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'An account with that email already exists.' }) })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderPage()

    await waitFor(() => screen.getByLabelText(/email/i))

    await user.type(screen.getByLabelText(/email/i), 'hiker@example.com')
    await user.type(screen.getByLabelText(/password/i), 'hunter2')
    await user.click(screen.getByRole('button', { name: /sign up/i }))

    await waitFor(() => {
      expect(screen.getByText(/already exists/i)).toBeInTheDocument()
    })
  })
})
