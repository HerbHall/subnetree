import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { render } from '@/test/utils'
import { LoginPage } from './login'
import { useAuthStore } from '@/stores/auth'

// Mock react-router-dom hooks
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useLocation: () => ({ state: null }),
  }
})

// Mock auth API
vi.mock('@/api/auth', async () => {
  const { authMockFactory } = await import('@/test/mocks/auth')
  return authMockFactory()
})

describe('LoginPage', () => {
  const user = userEvent.setup()

  beforeEach(() => {
    vi.clearAllMocks()
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isHydrated: true,
      pendingMFAToken: null,
      mfaRequired: false,
    })
  })

  it('shows loading state while checking setup', async () => {
    const { checkSetupRequired } = await import('@/api/auth')
    let resolveSetupCheck: (value: boolean) => void
    vi.mocked(checkSetupRequired).mockReturnValue(
      new Promise((resolve) => {
        resolveSetupCheck = resolve
      })
    )

    render(<LoginPage />)

    expect(screen.getByText(/checking setup status/i)).toBeInTheDocument()

    resolveSetupCheck!(false)

    await waitFor(() => {
      expect(screen.queryByText(/checking setup status/i)).not.toBeInTheDocument()
    })
  })

  it('redirects to setup page if setup is required', async () => {
    const { checkSetupRequired } = await import('@/api/auth')
    vi.mocked(checkSetupRequired).mockResolvedValue(true)

    render(<LoginPage />)

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/setup', { replace: true })
    })
  })

  it('renders login form after setup check completes', async () => {
    const { checkSetupRequired } = await import('@/api/auth')
    vi.mocked(checkSetupRequired).mockResolvedValue(false)

    render(<LoginPage />)

    await waitFor(() => {
      expect(screen.getByText(/sign in to subnetree/i)).toBeInTheDocument()
    })

    expect(screen.getByLabelText('Username')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('allows entering credentials', async () => {
    const { checkSetupRequired } = await import('@/api/auth')
    vi.mocked(checkSetupRequired).mockResolvedValue(false)

    render(<LoginPage />)

    await waitFor(() => {
      expect(screen.getByLabelText('Username')).toBeInTheDocument()
    })

    const usernameInput = screen.getByLabelText('Username')
    const passwordInput = screen.getByLabelText('Password')

    await user.type(usernameInput, 'testuser')
    await user.type(passwordInput, 'password123')

    expect(usernameInput).toHaveValue('testuser')
    expect(passwordInput).toHaveValue('password123')
  })

  it('shows error message on login failure', async () => {
    const { checkSetupRequired, loginApi } = await import('@/api/auth')
    vi.mocked(checkSetupRequired).mockResolvedValue(false)
    vi.mocked(loginApi).mockRejectedValue(new Error('Invalid credentials'))

    render(<LoginPage />)

    await waitFor(() => {
      expect(screen.getByLabelText('Username')).toBeInTheDocument()
    })

    await user.type(screen.getByLabelText('Username'), 'testuser')
    await user.type(screen.getByLabelText('Password'), 'wrongpass')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Invalid credentials')
    })
  })

  it('shows loading state during login', async () => {
    const { checkSetupRequired, loginApi } = await import('@/api/auth')
    vi.mocked(checkSetupRequired).mockResolvedValue(false)
    vi.mocked(loginApi).mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    )

    render(<LoginPage />)

    await waitFor(() => {
      expect(screen.getByLabelText('Username')).toBeInTheDocument()
    })

    await user.type(screen.getByLabelText('Username'), 'testuser')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(screen.getByRole('button', { name: /signing in/i })).toBeDisabled()
  })

  it('navigates to dashboard on successful login', async () => {
    const { checkSetupRequired, loginApi } = await import('@/api/auth')
    vi.mocked(checkSetupRequired).mockResolvedValue(false)
    vi.mocked(loginApi).mockResolvedValue({
      access_token: 'test-token',
      refresh_token: 'test-refresh',
      expires_in: 900,
    })

    // Mock jwt-decode for the store
    vi.mock('jwt-decode', () => ({
      jwtDecode: () => ({
        uid: 'user-123',
        usr: 'testuser',
        role: 'admin',
        exp: Math.floor(Date.now() / 1000) + 3600,
      }),
    }))

    render(<LoginPage />)

    await waitFor(() => {
      expect(screen.getByLabelText('Username')).toBeInTheDocument()
    })

    await user.type(screen.getByLabelText('Username'), 'testuser')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/dashboard', { replace: true })
    })
  })
})
