import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthProvider, useAuth } from './AuthContext.tsx'

function Probe() {
  const { user, isLoading, login, signup, logout } = useAuth()
  return (
    <div>
      <div data-testid="loading">{String(isLoading)}</div>
      <div data-testid="user">{user ? user.email : 'none'}</div>
      <button onClick={() => login('hiker@example.com', 'hunter2')}>login</button>
      <button onClick={() => signup('hiker@example.com', 'hunter2')}>signup</button>
      <button onClick={() => logout()}>logout</button>
    </div>
  )
}

function renderProbe() {
  return render(
    <AuthProvider>
      <Probe />
    </AuthProvider>
  )
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('AuthContext', () => {
  it('restores the session from /auth/me on mount', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ userId: 'abc', email: 'hiker@example.com' }),
    }))

    renderProbe()

    expect(screen.getByTestId('loading')).toHaveTextContent('true')

    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('hiker@example.com')
    })
    expect(screen.getByTestId('loading')).toHaveTextContent('false')
  })

  it('leaves user null when /auth/me returns unauthenticated', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: 'Authentication required.' }),
    }))

    renderProbe()

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false')
    })
    expect(screen.getByTestId('user')).toHaveTextContent('none')
  })

  it('login sets the user on success', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'Authentication required.' }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderProbe()

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false')
    })

    await user.click(screen.getByRole('button', { name: 'login' }))

    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('hiker@example.com')
    })
  })

  it('logout clears the user', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ userId: 'abc', email: 'hiker@example.com' }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ ok: true }) })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderProbe()

    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('hiker@example.com')
    })

    await user.click(screen.getByRole('button', { name: 'logout' }))

    await waitFor(() => {
      expect(screen.getByTestId('user')).toHaveTextContent('none')
    })
  })
})
