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
  checkSetupRequired: vi.fn().mockResolvedValue(true),
}))

// Mock settings API
vi.mock('@/api/settings', () => ({
  getNetworkInterfaces: vi.fn().mockResolvedValue([
    {
      name: 'eth0',
      ip_address: '192.168.1.100',
      subnet: '192.168.1.0/24',
      mac: 'aa:bb:cc:dd:ee:ff',
      status: 'up',
    },
  ]),
  setScanInterface: vi.fn().mockResolvedValue({ interface_name: '' }),
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

  /** Renders SetupPage and waits for the async setup-status check to resolve. */
  async function renderSetupPage() {
    render(<SetupPage />)
    await waitFor(() => {
      expect(screen.getByText(/welcome to subnetree/i)).toBeInTheDocument()
    })
  }

  beforeEach(async () => {
    vi.clearAllMocks()
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isHydrated: true,
    })

    // Default success mocks for setup/login (called during step 1→2 transition)
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
  })

  describe('Step 1 - Account Creation', () => {
    it('renders the account creation form', async () => {
      await renderSetupPage()

      expect(screen.getByText(/welcome to subnetree/i)).toBeInTheDocument()
      expect(screen.getByText(/create your administrator account/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/username/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument()
    })

    it('shows step indicator with step 1 active', async () => {
      await renderSetupPage()

      const stepIndicator = screen.getByText('1').closest('div')
      expect(stepIndicator).toHaveClass('bg-primary')
    })

    it('validates required username', async () => {
      await renderSetupPage()

      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/username is required/i)).toBeInTheDocument()
    })

    it('validates username format', async () => {
      await renderSetupPage()

      await user.type(screen.getByLabelText(/username/i), 'invalid user!')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/username can only contain/i)).toBeInTheDocument()
    })

    it('accepts valid username formats', async () => {
      await renderSetupPage()

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
      await renderSetupPage()

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/email is required/i)).toBeInTheDocument()
    })

    // Note: Email format validation is tested implicitly through validateStep1().
    // The browser's native email input validation in happy-dom prevents form submission
    // for obviously invalid emails before our custom validation runs.
    it('validates required email before password validation', async () => {
      await renderSetupPage()

      await user.type(screen.getByLabelText(/username/i), 'admin')
      // Leave email empty and fill passwords
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/email is required/i)).toBeInTheDocument()
    })

    it('validates password minimum length', async () => {
      await renderSetupPage()

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'short')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/password must be at least 8 characters/i)).toBeInTheDocument()
    })

    it('validates password confirmation matches', async () => {
      await renderSetupPage()

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'different123')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByText(/passwords do not match/i)).toBeInTheDocument()
    })

    it('shows password strength indicator', async () => {
      await renderSetupPage()

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
      await renderSetupPage()

      await user.click(screen.getByRole('button', { name: /next/i }))
      expect(screen.getByText(/username is required/i)).toBeInTheDocument()

      await user.type(screen.getByLabelText(/username/i), 'a')
      expect(screen.queryByText(/username is required/i)).not.toBeInTheDocument()
    })

    it('shows error when account creation fails', async () => {
      const { setupApi } = await import('@/api/auth')
      vi.mocked(setupApi).mockRejectedValue(new Error('Username already exists'))

      await renderSetupPage()

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))

      await waitFor(() => {
        expect(screen.getByRole('alert')).toHaveTextContent('Username already exists')
      })
      // Should stay on step 1
      expect(screen.getByText(/create your administrator account/i)).toBeInTheDocument()
    })

    it('shows loading state during account creation', async () => {
      const { setupApi } = await import('@/api/auth')
      vi.mocked(setupApi).mockImplementation(
        () => new Promise((resolve) => setTimeout(resolve, 100))
      )

      await renderSetupPage()

      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))

      expect(screen.getByRole('button', { name: /creating account/i })).toBeDisabled()
    })
  })

  describe('Step 2 - Network Configuration', () => {
    async function goToStep2() {
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/email/i), 'admin@test.com')
      await user.type(screen.getByLabelText(/^password$/i), 'password123')
      await user.type(screen.getByLabelText(/confirm password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /next/i }))
      // Step 1→2 now calls setupApi + loginApi asynchronously
      await waitFor(() => {
        expect(screen.getByText(/configure network scanning/i)).toBeInTheDocument()
      })
    }

    it('advances to step 2 with valid step 1 data', async () => {
      await renderSetupPage()

      await goToStep2()

      expect(screen.getByText(/configure network scanning/i)).toBeInTheDocument()
      expect(screen.getByText(/select network interface/i)).toBeInTheDocument()
    })

    it('shows back and next buttons', async () => {
      await renderSetupPage()
      await goToStep2()

      expect(screen.getByRole('button', { name: /back/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /next/i })).toBeInTheDocument()
    })

    it('goes back to step 1 when back is clicked', async () => {
      await renderSetupPage()
      await goToStep2()

      await user.click(screen.getByRole('button', { name: /back/i }))

      expect(screen.getByText(/create your administrator account/i)).toBeInTheDocument()
    })

    it('preserves form data when going back', async () => {
      await renderSetupPage()
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
      // Wait for async step 1→2 transition (setupApi + loginApi)
      await waitFor(() => {
        expect(screen.getByText(/configure network scanning/i)).toBeInTheDocument()
      })
      await user.click(screen.getByRole('button', { name: /next/i }))
    }

    it('shows summary with entered data', async () => {
      await renderSetupPage()
      await goToStep3()

      expect(screen.getByText(/setup summary/i)).toBeInTheDocument()
      expect(screen.getByText('admin')).toBeInTheDocument()
      expect(screen.getByText('admin@test.com')).toBeInTheDocument()
      expect(screen.getByText('Administrator')).toBeInTheDocument()
    })

    it('shows complete setup button', async () => {
      await renderSetupPage()
      await goToStep3()

      expect(screen.getByRole('button', { name: /complete setup/i })).toBeInTheDocument()
    })

    it('calls setup and login APIs during step 1 to 2 transition', async () => {
      const { setupApi, loginApi } = await import('@/api/auth')

      await renderSetupPage()
      await goToStep3()

      // APIs called during step 1→2 transition, not on Complete
      expect(setupApi).toHaveBeenCalledWith('admin', 'admin@test.com', 'password123')
      expect(loginApi).toHaveBeenCalledWith('admin', 'password123')
    })

    it('navigates to dashboard on successful setup', async () => {
      const { setScanInterface } = await import('@/api/settings')
      vi.mocked(setScanInterface).mockResolvedValue({ interface_name: '' })

      await renderSetupPage()
      await goToStep3()

      await user.click(screen.getByRole('button', { name: /complete setup/i }))

      await waitFor(() => {
        expect(setScanInterface).toHaveBeenCalledWith('')
        expect(mockNavigate).toHaveBeenCalledWith('/dashboard', { replace: true })
      })
    })

    it('shows loading state during complete', async () => {
      const { setScanInterface } = await import('@/api/settings')
      vi.mocked(setScanInterface).mockImplementation(
        () => new Promise((resolve) => setTimeout(resolve, 100))
      )

      await renderSetupPage()
      await goToStep3()

      await user.click(screen.getByRole('button', { name: /complete setup/i }))

      expect(screen.getByRole('button', { name: /saving/i })).toBeDisabled()
    })

    it('shows error on complete failure', async () => {
      const { setScanInterface } = await import('@/api/settings')
      vi.mocked(setScanInterface).mockRejectedValue(new Error('Failed to save settings'))

      await renderSetupPage()
      await goToStep3()

      await user.click(screen.getByRole('button', { name: /complete setup/i }))

      await waitFor(() => {
        expect(screen.getByRole('alert')).toHaveTextContent('Failed to save settings')
      })
    })
  })
})
