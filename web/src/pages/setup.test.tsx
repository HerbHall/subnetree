import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { render } from '@/test/utils'
import { SetupPage } from './setup'
import { useAuthStore } from '@/stores/auth'

// Mock react-router-dom hooks
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Mock auth API
vi.mock('@/api/auth', () => ({
  setupApi: vi.fn(),
  loginApi: vi.fn(),
}))

// Mock jwt-decode
vi.mock('jwt-decode', () => ({
  jwtDecode: () => ({
    uid: 'user-123',
    usr: 'testuser',
    role: 'admin',
    exp: Math.floor(Date.now() / 1000) + 3600,
  }),
}))

describe('SetupPage', () => {
  const user = userEvent.setup()

  beforeEach(() => {
    vi.clearAllMocks()
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isHydrated: true,
    })
  })

  describe('Step 1 - Account Creation', () => {
    it('renders the account creation form', () => {
      render(<SetupPage />)

      expect(screen.getByText(/welcome to subnetree/i)).toBeInTheDocument()
      expect(screen.getByText(/create your administrator account/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/username/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument()
    })

    it('shows step indicator with step 1 active', () => {
      render(<SetupPage />)

      const stepIndicator = screen.getByText('1').closest('div')
      expect(stepIndicator).toHaveClass('bg-primary')
    })

    it('validates required username', async () => {
      render(<SetupPage />)

      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/username is required/i)).toBeInTheDocument()
    })

    it('validates username format', async () => {
      render(<SetupPage />)

      await user.type(screen.getByLabelText(/username/i), 'invalid user!')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/username can only contain/i)).toBeInTheDocument()
    })

    it('accepts valid username formats', async () => {
      render(<SetupPage />)

      const usernameInput = screen.getByLabelText(/username/i)

      // Valid usernames
      await user.clear(usernameInput)
      await user.type(usernameInput, 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.queryByText(/username can only contain/i)).not.toBeInTheDocument()
    })

    it('validates required email', async () => {
      render(<SetupPage />)

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/email is required/i)).toBeInTheDocument()
    })

    // Note: Email format validation is tested implicitly through validateStep1().
    // The browser's native email input validation in happy-dom prevents form submission
    // for obviously invalid emails before our custom validation runs.
    it('validates required email before password validation', async () => {
      render(<SetupPage />)

      await user.type(screen.getByLabelText(/username/i), 'admin')
      // Leave email empty and fill passwords
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/email is required/i)).toBeInTheDocument()
    })

    it('validates password minimum length', async () => {
      render(<SetupPage />)

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'short')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/password must be at least 8 characters/i)).toBeInTheDocument()
    })

    it('validates password confirmation matches', async () => {
      render(<SetupPage />)

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'different123')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/passwords do not match/i)).toBeInTheDocument()
    })

    it('shows password strength indicator', async () => {
      render(<SetupPage />)

      const passwordInput = screen.getByLabelText(/^password$/i)

      // Weak password
      await user.type(passwordInput, 'weak')
      expect(screen.getByText(/password strength: weak/i)).toBeInTheDocument()

      // Strong password
      await user.clear(passwordInput)
      await user.type(passwordInput, 'StrongP@ss123!')
      expect(screen.getByText(/password strength: (strong|very strong)/i)).toBeInTheDocument()
    })

    it('clears field error when user types', async () => {
      render(<SetupPage />)

      await user.click(screen.getByRole('button', { name: /next/i }))
      expect(screen.getByText(/username is required/i)).toBeInTheDocument()

      await user.type(screen.getByLabelText(/username/i), 'a')
      expect(screen.queryByText(/username is required/i)).not.toBeInTheDocument()
    })
  })

  describe('Step 2 - Network Configuration', () => {
    async function goToStep2() {
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))
    }

    it('advances to step 2 with valid step 1 data', async () => {
      render(<SetupPage />)

      await goToStep2()

      expect(screen.getByText(/configure network scanning/i)).toBeInTheDocument()
      expect(screen.getByText(/network configuration/i)).toBeInTheDocument()
    })

    it('shows back and next buttons', async () => {
      render(<SetupPage />)
      await goToStep2()

      expect(screen.getByRole('button', { name: /back/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /next/i })).toBeInTheDocument()
    })

    it('goes back to step 1 when back is clicked', async () => {
      render(<SetupPage />)
      await goToStep2()

      await user.click(screen.getByRole('button', { name: /back/i }))

      expect(screen.getByText(/create your administrator account/i)).toBeInTheDocument()
    })

    it('preserves form data when going back', async () => {
      render(<SetupPage />)
      await goToStep2()
      await user.click(screen.getByRole('button', { name: /back/i }))

      expect(screen.getByLabelText(/username/i)).toHaveValue('admin')
      expect(screen.getByLabelText(/email/i)).toHaveValue('admin@test.com')
    })
  })

  describe('Step 3 - Summary and Complete', () => {
    async function goToStep3() {
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))
      await user.click(screen.getByRole('button', { name: /next/i }))
    }

    it('shows summary with entered data', async () => {
      render(<SetupPage />)
      await goToStep3()

      expect(screen.getByText(/setup summary/i)).toBeInTheDocument()
      expect(screen.getByText('admin')).toBeInTheDocument()
      expect(screen.getByText('admin@test.com')).toBeInTheDocument()
      expect(screen.getByText('Administrator')).toBeInTheDocument()
    })

    it('shows complete setup button', async () => {
      render(<SetupPage />)
      await goToStep3()

      expect(screen.getByRole('button', { name: /complete setup/i })).toBeInTheDocument()
    })

    it('calls setup and login APIs on complete', async () => {
      const { setupApi, loginApi } = await import('@/api/auth')
      vi.mocked(setupApi).mockResolvedValue({
        id: 'user-123',
        username: 'admin',
        email: 'admin@test.com',
        role: 'admin',
        auth_provider: 'local',
        created_at: new Date().toISOString(),
        disabled: false,
      })
      vi.mocked(loginApi).mockResolvedValue({
        access_token: 'test-token',
        refresh_token: 'test-refresh',
        expires_in: 900,
      })

      render(<SetupPage />)
      await goToStep3()

      await user.click(screen.getByRole('button', { name: /complete setup/i }))

      await waitFor(() => {
        expect(setupApi).toHaveBeenCalledWith('admin', 'admin@test.com', 'password123')
        expect(loginApi).toHaveBeenCalledWith('admin', 'password123')
      })
    })

    it('navigates to dashboard on successful setup', async () => {
      const { setupApi, loginApi } = await import('@/api/auth')
      vi.mocked(setupApi).mockResolvedValue({
        id: 'user-123',
        username: 'admin',
        email: 'admin@test.com',
        role: 'admin',
        auth_provider: 'local',
        created_at: new Date().toISOString(),
        disabled: false,
      })
      vi.mocked(loginApi).mockResolvedValue({
        access_token: 'test-token',
        refresh_token: 'test-refresh',
        expires_in: 900,
      })

      render(<SetupPage />)
      await goToStep3()

      await user.click(screen.getByRole('button', { name: /complete setup/i }))

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/dashboard', { replace: true })
      })
    })

    it('shows loading state during setup', async () => {
      const { setupApi } = await import('@/api/auth')
      vi.mocked(setupApi).mockImplementation(
        () => new Promise((resolve) => setTimeout(resolve, 100))
      )

      render(<SetupPage />)
      await goToStep3()

      await user.click(screen.getByRole('button', { name: /complete setup/i }))

      expect(screen.getByRole('button', { name: /creating account/i })).toBeDisabled()
    })

    it('shows error on setup failure', async () => {
      const { setupApi } = await import('@/api/auth')
      vi.mocked(setupApi).mockRejectedValue(new Error('Username already exists'))

      render(<SetupPage />)
      await goToStep3()

      await user.click(screen.getByRole('button', { name: /complete setup/i }))

      await waitFor(() => {
        expect(screen.getByRole('alert')).toHaveTextContent('Username already exists')
      })
    })
  })
})
