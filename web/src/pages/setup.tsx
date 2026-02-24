import { useState, useMemo, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth'
import { setupApi, loginApi, checkSetupRequired, isMFAChallenge } from '@/api/auth'
import { getNetworkInterfaces, setScanInterface, type NetworkInterface } from '@/api/settings'
import { setActiveTheme } from '@/api/themes'
import { getHealth } from '@/api/system'
import { detectColorScheme } from '@/lib/theme-preference'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Eye, EyeOff, Moon, Sun } from 'lucide-react'

type Step = 1 | 2 | 3 | 4

interface FormData {
  username: string
  email: string
  password: string
  confirmPassword: string
  selectedInterface: string // Empty string means auto-detect
  selectedThemeMode: 'dark' | 'light'
}

interface FormErrors {
  username?: string
  email?: string
  password?: string
  confirmPassword?: string
}

function getPasswordStrength(password: string): {
  score: number
  label: string
  color: string
} {
  let score = 0
  if (password.length >= 8) score++
  if (password.length >= 12) score++
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score++
  if (/\d/.test(password)) score++
  if (/[^a-zA-Z0-9]/.test(password)) score++

  if (score <= 1) return { score, label: 'Weak', color: 'bg-red-500' }
  if (score <= 2) return { score, label: 'Fair', color: 'bg-orange-500' }
  if (score <= 3) return { score, label: 'Good', color: 'bg-yellow-500' }
  if (score <= 4) return { score, label: 'Strong', color: 'bg-green-500' }
  return { score, label: 'Very Strong', color: 'bg-green-600' }
}

function StepIndicator({ currentStep }: { currentStep: Step }) {
  const steps = [
    { num: 1, label: 'Account' },
    { num: 2, label: 'Network' },
    { num: 3, label: 'Theme' },
    { num: 4, label: 'Complete' },
  ]

  return (
    <div className="flex items-center justify-center gap-1 sm:gap-2 mb-6 overflow-hidden">
      {steps.map((step, idx) => (
        <div key={step.num} className="flex items-center">
          <div
            className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors ${
              currentStep >= step.num
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted text-muted-foreground'
            }`}
          >
            {step.num}
          </div>
          <span
            className={`ml-2 text-sm hidden sm:inline ${
              currentStep >= step.num ? 'text-foreground' : 'text-muted-foreground'
            }`}
          >
            {step.label}
          </span>
          {idx < steps.length - 1 && (
            <div
              className={`mx-1 sm:mx-3 h-px w-6 sm:w-12 ${
                currentStep > step.num ? 'bg-primary' : 'bg-muted'
              }`}
            />
          )}
        </div>
      ))}
    </div>
  )
}

export function SetupPage() {
  const [step, setStep] = useState<Step>(1)
  const [formData, setFormData] = useState<FormData>({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    selectedInterface: '',
    selectedThemeMode: detectColorScheme(),
  })
  const [errors, setErrors] = useState<FormErrors>({})
  const [apiError, setApiError] = useState('')
  const [loading, setLoading] = useState(false)
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([])
  const [interfacesLoading, setInterfacesLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const setTokens = useAuthStore((s) => s.setTokens)
  const navigate = useNavigate()
  const [setupComplete, setSetupComplete] = useState(false)
  const [checkingStatus, setCheckingStatus] = useState(true)

  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    staleTime: 60 * 1000,
  })
  const version = health?.version

  // Guard: check if setup is already complete
  useEffect(() => {
    checkSetupRequired()
      .then((required) => {
        if (!required) {
          setSetupComplete(true)
        }
      })
      .finally(() => setCheckingStatus(false))
  }, [])

  // Fetch network interfaces when entering step 2
  useEffect(() => {
    if (step === 2 && interfaces.length === 0) {
      setInterfacesLoading(true)
      getNetworkInterfaces()
        .then((data) => {
          setInterfaces(data)
        })
        .catch((err) => {
          setApiError(err instanceof Error ? err.message : 'Failed to load network interfaces')
        })
        .finally(() => {
          setInterfacesLoading(false)
        })
    }
  }, [step, interfaces.length])

  const passwordStrength = useMemo(
    () => getPasswordStrength(formData.password),
    [formData.password]
  )

  function updateField<K extends keyof FormData>(key: K, value: FormData[K]) {
    setFormData((prev) => ({ ...prev, [key]: value }))
    // Only clear error for fields that have form errors
    if (key in errors) {
      const errorKey = key as keyof FormErrors
      if (errors[errorKey]) {
        setErrors((prev) => ({ ...prev, [errorKey]: undefined }))
      }
    }
  }

  function validateStep1(): boolean {
    const newErrors: FormErrors = {}

    if (!formData.username.trim()) {
      newErrors.username = 'Username is required'
    } else if (!/^[a-zA-Z0-9_-]+$/.test(formData.username)) {
      newErrors.username = 'Username can only contain letters, numbers, underscores, and dashes'
    }

    if (!formData.email.trim()) {
      newErrors.email = 'Email is required'
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Please enter a valid email address'
    }

    if (!formData.password) {
      newErrors.password = 'Password is required'
    } else if (formData.password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters'
    }

    if (!formData.confirmPassword) {
      newErrors.confirmPassword = 'Please confirm your password'
    } else if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = 'Passwords do not match'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  async function handleNext() {
    setApiError('')
    if (step === 1 && validateStep1()) {
      // Create account and log in before moving to step 2,
      // so the auth token is available for the network interfaces API
      setLoading(true)
      try {
        await setupApi(formData.username, formData.email, formData.password)
        const result = await loginApi(formData.username, formData.password)
        if (isMFAChallenge(result)) {
          throw new Error('MFA should not be enabled during initial setup')
        }
        setTokens(result.access_token, result.refresh_token)
        setStep(2)
      } catch (err) {
        setApiError(err instanceof Error ? err.message : 'Account creation failed')
      } finally {
        setLoading(false)
      }
    } else if (step === 2) {
      setStep(3)
    } else if (step === 3) {
      setStep(4)
    }
  }

  function handleBack() {
    setApiError('')
    if (step === 2) setStep(1)
    else if (step === 3) setStep(2)
    else if (step === 4) setStep(3)
  }

  async function handleComplete() {
    setApiError('')
    setLoading(true)
    try {
      // Account was already created and logged in during step 1 -> 2 transition
      // Save the selected interface (empty string means auto-detect)
      await setScanInterface(formData.selectedInterface)
      // Save the selected theme preference
      const themeId = formData.selectedThemeMode === 'dark'
        ? 'builtin-forest-dark'
        : 'builtin-forest-light'
      await setActiveTheme(themeId)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setApiError(err instanceof Error ? err.message : 'Setup failed')
    } finally {
      setLoading(false)
    }
  }

  if (checkingStatus) {
    return (
      <Card className="w-full max-w-lg">
        <CardContent className="py-8 text-center text-muted-foreground">
          Checking setup status...
        </CardContent>
      </Card>
    )
  }

  if (setupComplete) {
    return (
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>Setup Already Complete</CardTitle>
          <CardDescription>
            An administrator account has already been created.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="mb-4 text-sm text-muted-foreground">
            If you need to access SubNetree, sign in with your existing credentials.
            If you have forgotten your password, contact your system administrator.
          </p>
          <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
            Go to Login
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-lg">
      <CardHeader>
        <CardTitle>Welcome to SubNetree</CardTitle>
        <CardDescription>
          {step === 1 && 'Create your administrator account to get started.'}
          {step === 2 && 'Configure network scanning settings.'}
          {step === 3 && 'Choose your preferred appearance.'}
          {step === 4 && 'Review your settings and complete setup.'}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <StepIndicator currentStep={step} />

        {apiError && (
          <div
            role="alert"
            className="mb-4 rounded-md bg-red-900/20 p-3 text-sm text-red-400"
          >
            {apiError}
          </div>
        )}

        {step === 1 && (
          <form
            onSubmit={(e) => {
              e.preventDefault()
              handleNext()
            }}
            className="space-y-4"
          >
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                name="username"
                value={formData.username}
                onChange={(e) => updateField('username', e.target.value)}
                placeholder="admin"
                autoFocus
                autoComplete="username"
              />
              {errors.username && (
                <p className="text-sm text-red-400">{errors.username}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                name="email"
                type="email"
                value={formData.email}
                onChange={(e) => updateField('email', e.target.value)}
                placeholder="admin@example.com"
                autoComplete="email"
              />
              {errors.email && (
                <p className="text-sm text-red-400">{errors.email}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  id="password"
                  name="new-password"
                  type={showPassword ? 'text' : 'password'}
                  value={formData.password}
                  onChange={(e) => updateField('password', e.target.value)}
                  autoComplete="new-password"
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
              {formData.password && (
                <div className="space-y-1">
                  <div className="flex gap-1">
                    {[1, 2, 3, 4, 5].map((i) => (
                      <div
                        key={i}
                        className={`h-1 flex-1 rounded-full transition-colors ${
                          i <= passwordStrength.score
                            ? passwordStrength.color
                            : 'bg-muted'
                        }`}
                      />
                    ))}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Password strength: {passwordStrength.label}
                  </p>
                </div>
              )}
              {errors.password && (
                <p className="text-sm text-red-400">{errors.password}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Confirm Password</Label>
              <div className="relative">
                <Input
                  id="confirmPassword"
                  name="confirm-password"
                  type={showPassword ? 'text' : 'password'}
                  value={formData.confirmPassword}
                  onChange={(e) => updateField('confirmPassword', e.target.value)}
                  autoComplete="new-password"
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
              {errors.confirmPassword && (
                <p className="text-sm text-red-400">{errors.confirmPassword}</p>
              )}
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? 'Creating account...' : 'Next'}
            </Button>
          </form>
        )}

        {step === 2 && (
          <div className="space-y-4">
            <div className="space-y-3">
              <h3 className="font-medium text-foreground">Select Network Interface</h3>
              <p className="text-sm text-muted-foreground">
                Choose which network interface SubNetree should use for scanning.
              </p>

              {interfacesLoading ? (
                <div className="space-y-2">
                  {[1, 2, 3].map((i) => (
                    <div
                      key={i}
                      className="h-16 animate-pulse rounded-lg bg-muted"
                    />
                  ))}
                </div>
              ) : (
                <div className="space-y-2">
                  {/* Auto-detect option */}
                  <label
                    htmlFor="interface-auto"
                    className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                      formData.selectedInterface === ''
                        ? 'border-primary bg-primary/5'
                        : 'border-muted hover:border-muted-foreground/50'
                    }`}
                  >
                    <input
                      type="radio"
                      id="interface-auto"
                      name="interface"
                      value=""
                      checked={formData.selectedInterface === ''}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          selectedInterface: e.target.value,
                        }))
                      }
                      className="mt-1"
                    />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium">Auto-detect</span>
                        <span className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                          Recommended
                        </span>
                      </div>
                      <p className="mt-0.5 text-sm text-muted-foreground">
                        SubNetree will automatically select the best available interface
                      </p>
                    </div>
                  </label>

                  {/* Network interfaces */}
                  {interfaces.map((iface) => (
                    <label
                      key={iface.name}
                      htmlFor={`interface-${iface.name}`}
                      className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                        formData.selectedInterface === iface.name
                          ? 'border-primary bg-primary/5'
                          : 'border-muted hover:border-muted-foreground/50'
                      } ${iface.status === 'down' ? 'opacity-60' : ''}`}
                    >
                      <input
                        type="radio"
                        id={`interface-${iface.name}`}
                        name="interface"
                        value={iface.name}
                        checked={formData.selectedInterface === iface.name}
                        onChange={(e) =>
                          setFormData((prev) => ({
                            ...prev,
                            selectedInterface: e.target.value,
                          }))
                        }
                        className="mt-1"
                        disabled={iface.status === 'down'}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium truncate">{iface.name}</span>
                          <span
                            className={`rounded px-1.5 py-0.5 text-xs ${
                              iface.status === 'up'
                                ? 'bg-green-500/20 text-green-400'
                                : 'bg-red-500/20 text-red-400'
                            }`}
                          >
                            {iface.status}
                          </span>
                        </div>
                        <div className="mt-1 space-y-0.5 text-sm text-muted-foreground">
                          <p className="truncate">
                            <span className="font-medium">IP:</span> {iface.ip_address}
                          </p>
                          <p className="truncate">
                            <span className="font-medium">Subnet:</span> {iface.subnet}
                          </p>
                          {iface.mac && (
                            <p className="truncate">
                              <span className="font-medium">MAC:</span> {iface.mac}
                            </p>
                          )}
                        </div>
                      </div>
                    </label>
                  ))}

                  {interfaces.length === 0 && !interfacesLoading && (
                    <div className="rounded-lg border border-dashed border-muted-foreground/25 p-4 text-center">
                      <p className="text-sm text-muted-foreground">
                        No network interfaces found. SubNetree will use auto-detection.
                      </p>
                    </div>
                  )}
                </div>
              )}
            </div>

            <p className="text-sm text-muted-foreground">
              You can change this later in{' '}
              <span className="font-medium text-foreground">Settings → Network</span>.
            </p>

            <div className="flex gap-3">
              <Button type="button" variant="outline" onClick={handleBack} className="flex-1">
                Back
              </Button>
              <Button type="button" onClick={handleNext} className="flex-1">
                Next
              </Button>
            </div>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-4">
            <div className="space-y-3">
              <h3 className="font-medium text-foreground">Theme Preference</h3>
              <p className="text-sm text-muted-foreground">
                Select a color scheme that matches your preference. This is based on your
                system setting, but you can change it anytime.
              </p>

              <div className="grid grid-cols-2 gap-3">
                {/* Dark mode card */}
                <button
                  type="button"
                  onClick={() => updateField('selectedThemeMode', 'dark')}
                  className={`flex flex-col items-center gap-3 rounded-lg border p-4 transition-colors ${
                    formData.selectedThemeMode === 'dark'
                      ? 'border-primary bg-primary/5'
                      : 'border-muted hover:border-muted-foreground/50'
                  }`}
                >
                  <Moon className="h-8 w-8 text-muted-foreground" />
                  <div className="text-center">
                    <p className="font-medium text-foreground">Dark</p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      Easy on the eyes
                    </p>
                  </div>
                  {/* Color swatch preview */}
                  <div className="flex gap-1.5">
                    <div className="h-4 w-4 rounded-full" style={{ backgroundColor: '#0c1a0e' }} title="Background" />
                    <div className="h-4 w-4 rounded-full" style={{ backgroundColor: '#1a2e1c' }} title="Card" />
                    <div className="h-4 w-4 rounded-full" style={{ backgroundColor: '#4ade80' }} title="Accent" />
                    <div className="h-4 w-4 rounded-full" style={{ backgroundColor: '#f5f0e8' }} title="Text" />
                  </div>
                </button>

                {/* Light mode card */}
                <button
                  type="button"
                  onClick={() => updateField('selectedThemeMode', 'light')}
                  className={`flex flex-col items-center gap-3 rounded-lg border p-4 transition-colors ${
                    formData.selectedThemeMode === 'light'
                      ? 'border-primary bg-primary/5'
                      : 'border-muted hover:border-muted-foreground/50'
                  }`}
                >
                  <Sun className="h-8 w-8 text-muted-foreground" />
                  <div className="text-center">
                    <p className="font-medium text-foreground">Light</p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      Clean and bright
                    </p>
                  </div>
                  {/* Color swatch preview */}
                  <div className="flex gap-1.5">
                    <div className="h-4 w-4 rounded-full border border-muted" style={{ backgroundColor: '#f5f5f0' }} title="Background" />
                    <div className="h-4 w-4 rounded-full border border-muted" style={{ backgroundColor: '#ffffff' }} title="Card" />
                    <div className="h-4 w-4 rounded-full" style={{ backgroundColor: '#16a34a' }} title="Accent" />
                    <div className="h-4 w-4 rounded-full" style={{ backgroundColor: '#1a2e1c' }} title="Text" />
                  </div>
                </button>
              </div>
            </div>

            <p className="text-sm text-muted-foreground">
              You can change this later in{' '}
              <span className="font-medium text-foreground">Settings → Themes</span>.
            </p>

            <div className="flex gap-3">
              <Button type="button" variant="outline" onClick={handleBack} className="flex-1">
                Back
              </Button>
              <Button type="button" onClick={handleNext} className="flex-1">
                Next
              </Button>
            </div>
          </div>
        )}

        {step === 4 && (
          <div className="space-y-4">
            <div className="rounded-lg bg-muted/50 p-4 space-y-3">
              <h3 className="font-medium text-foreground">Setup Summary</h3>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Username</span>
                  <span className="font-medium">{formData.username}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Email</span>
                  <span className="font-medium">{formData.email}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Role</span>
                  <span className="font-medium">Administrator</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Network Interface</span>
                  <span className="font-medium">
                    {formData.selectedInterface || 'Auto-detect'}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Theme</span>
                  <span className="font-medium capitalize">
                    {formData.selectedThemeMode}
                  </span>
                </div>
              </div>
            </div>

            <p className="text-sm text-muted-foreground">
              Click Complete to save your settings and start using SubNetree.
            </p>

            <div className="flex gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={handleBack}
                className="flex-1"
                disabled={loading}
              >
                Back
              </Button>
              <Button
                type="button"
                onClick={handleComplete}
                className="flex-1"
                disabled={loading}
              >
                {loading ? 'Saving...' : 'Complete Setup'}
              </Button>
            </div>
          </div>
        )}

        <p className="text-xs text-muted-foreground text-center mt-4">
          SubNetree {version?.version || ''}
        </p>
      </CardContent>
    </Card>
  )
}
