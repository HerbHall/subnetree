import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Shield, ShieldCheck, ShieldX, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { MFASetup } from './mfa-setup'

interface UserInfo {
  id: string
  username: string
  totp_enabled: boolean
}

export function MFAStatus() {
  const [showSetup, setShowSetup] = useState(false)
  const [showDisable, setShowDisable] = useState(false)
  const [disableCode, setDisableCode] = useState('')
  const user = useAuthStore((s) => s.user)
  const queryClient = useQueryClient()

  const { data: userInfo, isLoading } = useQuery({
    queryKey: ['user', user?.id],
    queryFn: () => api.get<UserInfo>(`/users/${user?.id}`),
    enabled: !!user?.id,
  })

  const disableMutation = useMutation({
    mutationFn: (totpCode: string) => api.post('/auth/mfa/disable', { totp_code: totpCode }),
    onSuccess: () => {
      toast.success('MFA disabled')
      setShowDisable(false)
      setDisableCode('')
      queryClient.invalidateQueries({ queryKey: ['user', user?.id] })
    },
    onError: () => {
      toast.error('Invalid code. MFA was not disabled.')
    },
  })

  const handleDisableSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      if (disableCode.length === 6) {
        disableMutation.mutate(disableCode)
      }
    },
    [disableCode, disableMutation],
  )

  const handleSetupComplete = useCallback(() => {
    setShowSetup(false)
    queryClient.invalidateQueries({ queryKey: ['user', user?.id] })
  }, [queryClient, user?.id])

  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-8 flex items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  if (showSetup) {
    return <MFASetup onComplete={handleSetupComplete} onCancel={() => setShowSetup(false)} />
  }

  const mfaEnabled = userInfo?.totp_enabled ?? false

  // Disable confirmation form
  if (showDisable) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <ShieldX className="h-4 w-4 text-red-500" />
            Disable Two-Factor Authentication
          </CardTitle>
          <CardDescription>
            Enter your current authentication code to confirm disabling MFA.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleDisableSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="disable-totp">Authentication Code</Label>
              <Input
                id="disable-totp"
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
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
            <div className="flex gap-2">
              <Button
                type="submit"
                variant="destructive"
                disabled={disableCode.length !== 6 || disableMutation.isPending}
                className="gap-2"
              >
                {disableMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {disableMutation.isPending ? 'Disabling...' : 'Disable MFA'}
              </Button>
              <Button
                variant="ghost"
                onClick={() => {
                  setShowDisable(false)
                  setDisableCode('')
                }}
              >
                Cancel
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Shield className="h-4 w-4 text-muted-foreground" />
          Two-Factor Authentication
        </CardTitle>
        <CardDescription>
          Add an extra layer of security to your account with TOTP-based authentication.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {mfaEnabled ? (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-green-500" />
              <span className="text-sm font-medium text-green-600 dark:text-green-400">
                MFA is enabled
              </span>
            </div>
            <p className="text-sm text-muted-foreground">
              Your account is protected with two-factor authentication.
            </p>
            <Button
              variant="outline"
              onClick={() => setShowDisable(true)}
              className="text-red-600 hover:text-red-700 dark:text-red-400"
            >
              Disable MFA
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <ShieldX className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">MFA is not enabled</span>
            </div>
            <p className="text-sm text-muted-foreground">
              Enable two-factor authentication to add an extra layer of security to your account.
            </p>
            <Button onClick={() => setShowSetup(true)}>Enable MFA</Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
