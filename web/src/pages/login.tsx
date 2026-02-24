import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'
import { checkSetupRequired } from '@/api/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Eye, EyeOff } from 'lucide-react'

export function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [checkingSetup, setCheckingSetup] = useState(true)

  // MFA state
  const [totpCode, setTotpCode] = useState('')
  const [recoveryCode, setRecoveryCode] = useState('')
  const [showRecovery, setShowRecovery] = useState(false)

  const login = useAuthStore((s) => s.login)
  const mfaRequired = useAuthStore((s) => s.mfaRequired)
  const completeMFA = useAuthStore((s) => s.completeMFA)
  const completeMFAWithRecovery = useAuthStore((s) => s.completeMFAWithRecovery)
  const navigate = useNavigate()
  const location = useLocation()

  const from = (location.state as { from?: { pathname: string } })?.from?.pathname || '/dashboard'

  useEffect(() => {
    checkSetupRequired()
      .then((required) => {
        if (required) {
          navigate('/setup', { replace: true })
        }
      })
      .finally(() => setCheckingSetup(false))
  }, [navigate])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(username, password)
      // If MFA is not required, login completed and we navigate.
      // If MFA is required, the store sets mfaRequired=true and we show the MFA form.
      if (!useAuthStore.getState().mfaRequired) {
        navigate(from, { replace: true })
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  async function handleMFASubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (showRecovery) {
        await completeMFAWithRecovery(recoveryCode)
      } else {
        await completeMFA(totpCode)
      }
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed')
    } finally {
      setLoading(false)
    }
  }

  if (checkingSetup) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-muted-foreground">
          Checking setup status...
        </CardContent>
      </Card>
    )
  }

  // MFA verification step
  if (mfaRequired) {
    return (
      <Card>
        <CardHeader className="items-center">
          <img src="/favicon.svg" alt="SubNetree" className="mb-2 h-12 w-12" />
          <CardTitle>Two-Factor Authentication</CardTitle>
          <CardDescription>
            {showRecovery
              ? 'Enter a recovery code to sign in.'
              : 'Enter the 6-digit code from your authenticator app.'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleMFASubmit} className="space-y-4">
            {error && (
              <div role="alert" className="rounded-md bg-red-900/20 p-3 text-sm text-red-400">
                {error}
              </div>
            )}
            {showRecovery ? (
              <div className="space-y-2">
                <Label htmlFor="recovery-code">Recovery Code</Label>
                <Input
                  id="recovery-code"
                  value={recoveryCode}
                  onChange={(e) => setRecoveryCode(e.target.value)}
                  placeholder="a1b2c3d4"
                  required
                  autoFocus
                  autoComplete="off"
                />
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="totp-code">Authentication Code</Label>
                <Input
                  id="totp-code"
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  inputMode="numeric"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  placeholder="123456"
                  required
                  autoFocus
                  autoComplete="one-time-code"
                  className="text-center text-lg tracking-widest"
                />
              </div>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? 'Verifying...' : 'Verify'}
            </Button>
            <button
              type="button"
              onClick={() => {
                setShowRecovery(!showRecovery)
                setError('')
                setTotpCode('')
                setRecoveryCode('')
              }}
              className="w-full text-center text-sm text-muted-foreground hover:text-foreground"
            >
              {showRecovery ? 'Use authenticator app instead' : 'Use a recovery code instead'}
            </button>
          </form>
        </CardContent>
      </Card>
    )
  }

  // Normal login form
  return (
    <Card>
      <CardHeader className="items-center">
        <img src="/favicon.svg" alt="SubNetree" className="mb-2 h-12 w-12" />
        <CardTitle>Sign in to SubNetree</CardTitle>
        <CardDescription>Enter your credentials to access the dashboard.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div role="alert" className="rounded-md bg-red-900/20 p-3 text-sm text-red-400">
              {error}
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="username">Username</Label>
            <Input
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                tabIndex={-1}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign in'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
