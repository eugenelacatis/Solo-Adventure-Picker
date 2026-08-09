import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { AuthProvider } from './AuthContext.tsx'
import RequireAuth from './RequireAuth.tsx'

function renderProtected() {
  return render(
    <MemoryRouter initialEntries={['/protected']}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<div>login page</div>} />
          <Route
            path="/protected"
            element={
              <RequireAuth>
                <div>secret content</div>
              </RequireAuth>
            }
          />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('RequireAuth', () => {
  it('redirects to /login when there is no session', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: 'Authentication required.' }),
    }))

    renderProtected()

    await waitFor(() => {
      expect(screen.getByText('login page')).toBeInTheDocument()
    })
  })

  it('renders children when a session exists', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ userId: 'abc', email: 'hiker@example.com' }),
    }))

    renderProtected()

    await waitFor(() => {
      expect(screen.getByText('secret content')).toBeInTheDocument()
    })
  })
})
