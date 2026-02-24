import { vi } from 'vitest'

/**
 * Canonical mock factory for @/api/auth.
 *
 * Usage:
 *   import { authMockFactory } from '@/test/mocks/auth'
 *   vi.mock('@/api/auth', () => authMockFactory())
 *
 * To override specific behavior (e.g., setup.test.tsx where MFA is never triggered):
 *   const { isMFAChallenge } = await import('@/api/auth')
 *   vi.mocked(isMFAChallenge).mockReturnValue(false)
 */
export function authMockFactory() {
  return {
    checkSetupRequired: vi.fn(),
    setupApi: vi.fn(),
    loginApi: vi.fn(),
    refreshApi: vi.fn(),
    logoutApi: vi.fn(),
    verifyMFAApi: vi.fn(),
    verifyMFARecoveryApi: vi.fn(),
    isMFAChallenge: vi.fn((data: unknown) => {
      return typeof data === 'object' && data !== null && 'mfa_required' in data
    }),
  }
}
