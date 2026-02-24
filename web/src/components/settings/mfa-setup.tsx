import { useState, useCallback } from 'react'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Copy, Check, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/api/client'

interface MFASetupResponse {
  otpauth_url: string
  recovery_codes: string[]
}

interface MFASetupProps {
  onComplete: () => void
  onCancel: () => void
}

export function MFASetup({ onComplete, onCancel }: MFASetupProps) {
  const [step, setStep] = useState<'init' | 'verify'>('init')
  const [setupData, setSetupData] = useState<MFASetupResponse | null>(null)
  const [verifyCode, setVerifyCode] = useState('')
  const [copied, setCopied] = useState(false)

  const setupMutation = useMutation({
    mutationFn: () => api.post<MFASetupResponse>('/auth/mfa/setup'),
    onSuccess: (data) => {
      setSetupData(data)
      setStep('init')
    },
    onError: () => {
      toast.error('Failed to initiate MFA setup')
    },
  })

  const verifyMutation = useMutation({
    mutationFn: (totpCode: string) => api.post('/auth/mfa/verify-setup', { totp_code: totpCode }),
    onSuccess: () => {
      toast.success('MFA enabled successfully')
      onComplete()
    },
    onError: () => {
      toast.error('Invalid code. Please try again.')
    },
  })

  const handleCopyRecoveryCodes = useCallback(() => {
    if (!setupData) return
    const text = setupData.recovery_codes.join('\n')
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [setupData])

  const handleDownloadRecoveryCodes = useCallback(() => {
    if (!setupData) return
    const text = `SubNetree Recovery Codes\n${'='.repeat(30)}\n\nStore these codes in a safe place.\nEach code can only be used once.\n\n${setupData.recovery_codes.join('\n')}\n`
    const blob = new Blob([text], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'subnetree-recovery-codes.txt'
    a.click()
    URL.revokeObjectURL(url)
  }, [setupData])

  const handleVerifySubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      if (verifyCode.length === 6) {
        verifyMutation.mutate(verifyCode)
      }
    },
    [verifyCode, verifyMutation],
  )

  // Initial state: start setup
  if (!setupData) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Enable Two-Factor Authentication</CardTitle>
          <CardDescription>
            Add an extra layer of security to your account using a TOTP authenticator app.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            You will need an authenticator app such as Google Authenticator, Authy, or 1Password.
          </p>
          <div className="flex gap-2">
            <Button
              onClick={() => setupMutation.mutate()}
              disabled={setupMutation.isPending}
              className="gap-2"
            >
              {setupMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              {setupMutation.isPending ? 'Setting up...' : 'Begin Setup'}
            </Button>
            <Button variant="ghost" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  // Step 1: Show secret and recovery codes
  if (step === 'init') {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Step 1: Configure Authenticator</CardTitle>
          <CardDescription>
            Add this account to your authenticator app using the URL below.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* OTPAuth URL */}
          <div className="space-y-2">
            <Label>Setup URL</Label>
            <p className="text-xs text-muted-foreground">
              Copy this URL into your authenticator app, or manually enter the secret.
            </p>
            <Input readOnly value={setupData.otpauth_url} className="font-mono text-xs" />
          </div>

          {/* Recovery Codes */}
          <div className="space-y-2">
            <Label>Recovery Codes</Label>
            <p className="text-xs text-muted-foreground">
              Save these codes in a safe place. Each can only be used once to sign in if you lose
              access to your authenticator app.
            </p>
            <div className="rounded-md border bg-muted/30 p-3">
              <div className="grid grid-cols-2 gap-1 font-mono text-sm">
                {setupData.recovery_codes.map((code) => (
                  <span key={code}>{code}</span>
                ))}
              </div>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={handleCopyRecoveryCodes} className="gap-2">
                {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                {copied ? 'Copied' : 'Copy'}
              </Button>
              <Button variant="outline" size="sm" onClick={handleDownloadRecoveryCodes}>
                Download
              </Button>
            </div>
          </div>

          <Button onClick={() => setStep('verify')}>
            I have saved my recovery codes
          </Button>
        </CardContent>
      </Card>
    )
  }

  // Step 2: Verify TOTP code
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Step 2: Verify Setup</CardTitle>
        <CardDescription>
          Enter the 6-digit code from your authenticator app to complete setup.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleVerifySubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="verify-totp">Authentication Code</Label>
            <Input
              id="verify-totp"
              value={verifyCode}
              onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
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
              disabled={verifyCode.length !== 6 || verifyMutation.isPending}
              className="gap-2"
            >
              {verifyMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              {verifyMutation.isPending ? 'Verifying...' : 'Verify and Enable'}
            </Button>
            <Button variant="ghost" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
