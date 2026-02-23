import { useState } from 'react'
import {
  Download,
  Terminal,
  Copy,
  Check,
  ExternalLink,
  Loader2,
  ChevronDown,
  ChevronRight,
} from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { HelpIcon, HelpPopover } from '@/components/contextual-help'
import { createEnrollmentToken, getInstallScriptUrl } from '@/api/agents'
import { useAuthStore } from '@/stores/auth'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Platform = 'linux' | 'macos' | 'windows'

interface ArchOption {
  label: string
  arch: string
  filename: string
}

interface ShellVariant {
  shell: string
  code: string
}

interface PlatformConfig {
  label: string
  architectures: ArchOption[]
  scriptFilename: string
  runCommand: string
  installVariants: (filename: string) => ShellVariant[]
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const GITHUB_RELEASE_BASE =
  'https://github.com/HerbHall/subnetree/releases/latest/download'

const PLATFORMS: Record<Platform, PlatformConfig> = {
  linux: {
    label: 'Linux',
    architectures: [
      { label: 'x86_64 (amd64)', arch: 'amd64', filename: 'scout_linux_amd64' },
      { label: 'ARM64 (aarch64)', arch: 'arm64', filename: 'scout_linux_arm64' },
    ],
    scriptFilename: 'install-scout.sh',
    runCommand: 'sudo bash install-scout.sh',
    installVariants: (filename: string) => [
      {
        shell: 'bash',
        code: `curl -LO ${GITHUB_RELEASE_BASE}/${filename}\nchmod +x ${filename}\nsudo mv ${filename} /usr/local/bin/scout`,
      },
    ],
  },
  macos: {
    label: 'macOS',
    architectures: [
      { label: 'Apple Silicon (arm64)', arch: 'arm64', filename: 'scout_darwin_arm64' },
      { label: 'Intel (amd64)', arch: 'amd64', filename: 'scout_darwin_amd64' },
    ],
    scriptFilename: 'install-scout.sh',
    runCommand: 'sudo bash install-scout.sh',
    installVariants: (filename: string) => [
      {
        shell: 'bash',
        code: `curl -LO ${GITHUB_RELEASE_BASE}/${filename}\nchmod +x ${filename}\nmv ${filename} /usr/local/bin/scout`,
      },
    ],
  },
  windows: {
    label: 'Windows',
    architectures: [
      { label: 'x86_64 (amd64)', arch: 'amd64', filename: 'scout_windows_amd64.exe' },
    ],
    scriptFilename: 'install-scout.ps1',
    runCommand: 'powershell -ExecutionPolicy Bypass -File install-scout.ps1',
    installVariants: (filename: string) => [
      {
        shell: 'PowerShell',
        code: `Invoke-WebRequest -Uri "${GITHUB_RELEASE_BASE}/${filename}" -OutFile scout.exe\nNew-Item -ItemType Directory -Force -Path "$env:ProgramFiles\\SubNetree" | Out-Null\nMove-Item scout.exe "$env:ProgramFiles\\SubNetree\\scout.exe"`,
      },
      {
        shell: 'CMD',
        code: `curl -LO ${GITHUB_RELEASE_BASE}/${filename}\nmkdir "%ProgramFiles%\\SubNetree" 2>nul\nmove /Y ${filename} "%ProgramFiles%\\SubNetree\\scout.exe"`,
      },
    ],
  },
}

const ENROLLMENT_COMMAND =
  'scout --server <server-ip>:9090 --enroll-token <enrollment-token>'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function detectPlatform(): Platform {
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('mac')) return 'macos'
  if (ua.includes('win')) return 'windows'
  return 'linux'
}

// ---------------------------------------------------------------------------
// Reusable Components
// ---------------------------------------------------------------------------

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={handleCopy}
      className="h-6 w-6 shrink-0"
      title="Copy to clipboard"
    >
      {copied ? (
        <Check className="h-3 w-3 text-green-500" />
      ) : (
        <Copy className="h-3 w-3" />
      )}
    </Button>
  )
}

function CodeBlock({ code, language }: { code: string; language: string }) {
  return (
    <div className="relative group">
      <div className="absolute right-2 top-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        <CopyButton text={code} />
      </div>
      <div className="rounded-md bg-black/40 border border-[var(--nv-border-subtle)] overflow-hidden">
        <div className="flex items-center gap-1.5 px-3 py-1 border-b border-[var(--nv-border-subtle)] bg-black/20">
          <Terminal className="h-3 w-3 text-muted-foreground/60" />
          <span className="text-[10px] font-mono text-muted-foreground/60 uppercase tracking-wider">{language}</span>
        </div>
        <pre className="p-4 overflow-x-auto">
          <code className="text-xs font-mono text-muted-foreground whitespace-pre">
            {code}
          </code>
        </pre>
      </div>
    </div>
  )
}

function PlatformTab({
  platform,
  isActive,
  onClick,
}: {
  platform: PlatformConfig
  isActive: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors',
        isActive
          ? 'bg-[var(--nv-sidebar-active-bg)] text-[var(--nv-sidebar-active)]'
          : 'text-muted-foreground hover:bg-[var(--nv-bg-hover)] hover:text-foreground'
      )}
    >
      <Terminal className="h-4 w-4" />
      {platform.label}
    </button>
  )
}

function ShellTabs({ variants }: { variants: ShellVariant[] }) {
  const [activeShell, setActiveShell] = useState(0)

  if (variants.length === 1) {
    return <CodeBlock code={variants[0].code} language={variants[0].shell} />
  }

  return (
    <div>
      <div className="flex gap-1 mb-2">
        {variants.map((v, i) => (
          <button
            key={v.shell}
            type="button"
            onClick={() => setActiveShell(i)}
            className={cn(
              'px-3 py-1 text-xs font-medium rounded-md transition-colors',
              activeShell === i
                ? 'bg-[var(--nv-sidebar-active-bg)] text-[var(--nv-sidebar-active)]'
                : 'text-muted-foreground hover:bg-[var(--nv-bg-hover)] hover:text-foreground'
            )}
          >
            {v.shell}
          </button>
        ))}
      </div>
      <CodeBlock code={variants[activeShell].code} language={variants[activeShell].shell} />
    </div>
  )
}

function ArchitectureSection({
  archOption,
  platformConfig,
}: {
  archOption: ArchOption
  platformConfig: PlatformConfig
}) {
  const variants = platformConfig.installVariants(archOption.filename)
  const downloadUrl = `${GITHUB_RELEASE_BASE}/${archOption.filename}`

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h4 className="text-sm font-medium">{archOption.label}</h4>
          <p className="text-xs text-muted-foreground mt-0.5">
            {archOption.filename}
          </p>
        </div>
        <a
          href={downloadUrl}
          className="inline-flex items-center gap-1.5 text-sm text-green-500 hover:text-green-400 transition-colors"
        >
          <Download className="h-3.5 w-3.5" />
          Download
        </a>
      </div>
      <ShellTabs variants={variants} />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function AgentSetupPage() {
  const [activePlatform, setActivePlatform] = useState<Platform>(detectPlatform)
  const [selectedArch, setSelectedArch] = useState(0)
  const [downloaded, setDownloaded] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const accessToken = useAuthStore((s) => s.accessToken)
  const serverHost = window.location.hostname

  const platformConfig = PLATFORMS[activePlatform]
  const currentArch = platformConfig.architectures[selectedArch] ?? platformConfig.architectures[0]

  const enrollmentCommand = ENROLLMENT_COMMAND.replace(
    '<server-ip>',
    serverHost || '<server-ip>'
  )

  // Reset arch selection and download state when platform changes
  const handlePlatformChange = (platform: Platform) => {
    setActivePlatform(platform)
    setSelectedArch(0)
    setDownloaded(false)
  }

  const installMutation = useMutation({
    mutationFn: async () => {
      // Step 1: Create a short-lived enrollment token
      const tokenResp = await createEnrollmentToken({
        description: `One-click install (${activePlatform}/${currentArch.arch})`,
        max_uses: 1,
        expires_in: '1h',
      })

      // Step 2: Fetch the install script with auth
      const url = getInstallScriptUrl(activePlatform, currentArch.arch, tokenResp.token)
      const resp = await fetch(url, {
        headers: { Authorization: `Bearer ${accessToken}` },
      })

      if (!resp.ok) {
        throw new Error(`Download failed (${resp.status})`)
      }

      const blob = await resp.blob()

      // Step 3: Trigger browser download
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = platformConfig.scriptFilename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(a.href)
    },
    onSuccess: () => {
      setDownloaded(true)
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to generate installer')
    },
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold">Deploy Scout Agent</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Install the Scout agent on a device to start monitoring it with SubNetree
        </p>
      </div>

      {/* Primary Install Card */}
      <Card className="border-[var(--nv-accent)]/30">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Download className="h-4 w-4 text-[var(--nv-accent)]" />
            One-Click Install
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-5">
          {/* Platform tabs */}
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-2 block">
              Platform
              <HelpIcon content="Choose the operating system where Scout will be installed. The installer is pre-configured for the selected platform." />
            </label>
            <div className="flex gap-2 rounded-lg border border-[var(--nv-border-subtle)] bg-card p-1 w-fit">
              {(Object.entries(PLATFORMS) as [Platform, PlatformConfig][]).map(
                ([key, config]) => (
                  <PlatformTab
                    key={key}
                    platform={config}
                    isActive={activePlatform === key}
                    onClick={() => handlePlatformChange(key)}
                  />
                )
              )}
            </div>
          </div>

          {/* Architecture selector */}
          {platformConfig.architectures.length > 1 && (
            <div>
              <label className="text-xs font-medium text-muted-foreground mb-2 block">
                Architecture
              </label>
              <div className="flex gap-2">
                {platformConfig.architectures.map((archOpt, idx) => (
                  <button
                    key={archOpt.arch}
                    type="button"
                    onClick={() => { setSelectedArch(idx); setDownloaded(false) }}
                    className={cn(
                      'px-4 py-2 text-sm font-medium rounded-md border transition-colors',
                      selectedArch === idx
                        ? 'border-[var(--nv-accent)] bg-[var(--nv-accent)]/10 text-[var(--nv-accent)]'
                        : 'border-[var(--nv-border-subtle)] text-muted-foreground hover:bg-[var(--nv-bg-hover)] hover:text-foreground'
                    )}
                  >
                    {archOpt.label}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Install button */}
          <div>
            <Button
              size="lg"
              className="gap-2 w-full sm:w-auto"
              onClick={() => installMutation.mutate()}
              disabled={installMutation.isPending}
            >
              {installMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              {installMutation.isPending
                ? 'Generating installer...'
                : `Install Scout on ${platformConfig.label}`}
            </Button>
            <p className="text-xs text-muted-foreground mt-2">
              Downloads a pre-configured install script with an embedded enrollment token.
              The token expires in 1 hour and can only be used once.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Post-Download Card */}
      {downloaded && (
        <Card className="border-green-500/30 bg-green-500/5">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2 text-green-500">
              <Check className="h-4 w-4" />
              Next Step: Run the Installer
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Open a terminal on the target device and run the downloaded script:
            </p>
            <div className="relative group">
              <div className="absolute right-2 top-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity z-10">
                <CopyButton text={platformConfig.runCommand} />
              </div>
              <div className="rounded-md bg-black/40 border border-green-500/20 overflow-hidden">
                <pre className="p-4 overflow-x-auto">
                  <code className="text-sm font-mono text-green-400 whitespace-pre">
                    {platformConfig.runCommand}
                  </code>
                </pre>
              </div>
            </div>
            <p className="text-xs text-muted-foreground">
              The script will download the Scout binary, install it as a system service,
              and automatically enroll with this SubNetree server.
            </p>
          </CardContent>
        </Card>
      )}

      {/* Advanced Section (collapsible) */}
      <div className="border border-[var(--nv-border-subtle)] rounded-lg">
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex items-center justify-between w-full px-4 py-3 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
        >
          <span>Advanced: Manual Installation</span>
          {showAdvanced ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </button>

        {showAdvanced && (
          <div className="px-4 pb-4 space-y-6 border-t border-[var(--nv-border-subtle)] pt-4">
            {/* Windows Installer - Primary option */}
            {activePlatform === 'windows' && (
              <Card className="border-[var(--nv-accent)]/30 bg-[var(--nv-accent)]/5">
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium flex items-center gap-2">
                    <Download className="h-4 w-4" />
                    Recommended: Windows Installer
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <p className="text-sm text-muted-foreground">
                    The installer configures Scout as a Windows service with automatic startup,
                    Start Menu entries, and PATH registration.
                  </p>
                  <div className="flex items-center gap-3">
                    <a
                      href={`${GITHUB_RELEASE_BASE}/SubNetreeScout-setup.exe`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2"
                    >
                      <Button size="sm">
                        <Download className="h-4 w-4 mr-1" />
                        Download Installer (.exe)
                      </Button>
                    </a>
                    <span className="text-xs text-muted-foreground">
                      Requires Windows 10+ (64-bit)
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground italic">
                    Or use the manual installation below for more control.
                  </p>
                </CardContent>
              </Card>
            )}

            {/* Manual download instructions per architecture */}
            <div>
              <h3 className="text-sm font-medium mb-4 flex items-center gap-2">
                <Download className="h-4 w-4 text-muted-foreground" />
                Download and Install Scout for {platformConfig.label}
              </h3>
              <div className="space-y-6">
                {platformConfig.architectures.map((archOption) => (
                  <ArchitectureSection
                    key={archOption.arch}
                    archOption={archOption}
                    platformConfig={platformConfig}
                  />
                ))}
              </div>
            </div>

            {/* Enrollment */}
            <div className="border-t border-[var(--nv-border-subtle)] pt-4">
              <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
                <Terminal className="h-4 w-4 text-muted-foreground" />
                Enroll Agent
                <HelpPopover title="Enrollment Token">
                  <p className="text-xs text-muted-foreground">
                    A one-time token that authorizes a new Scout agent to register with this SubNetree server.
                    Generate a token from the Settings page, then pass it to the agent's enroll command.
                  </p>
                </HelpPopover>
              </h3>
              <div className="space-y-4">
                <p className="text-sm text-muted-foreground">
                  After installing Scout, enroll it with your SubNetree server.
                  Replace <code className="text-xs bg-black/30 rounded px-1 py-0.5">&lt;enrollment-token&gt;</code> with
                  the token from the Agents page.
                </p>
                <CodeBlock code={enrollmentCommand} language={activePlatform === 'windows' ? 'PowerShell / CMD' : 'bash'} />
                <p className="text-xs text-muted-foreground">
                  The agent will establish a secure mTLS connection to the server
                  on port 9090 and begin reporting system metrics.
                </p>
              </div>
            </div>

            {/* Quick reference */}
            <div className="border-t border-[var(--nv-border-subtle)] pt-4">
              <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
                <Terminal className="h-4 w-4 text-muted-foreground" />
                Quick Reference
              </h3>
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-medium">Verify Installation</h4>
                  <div className="mt-2">
                    <CodeBlock code="scout --version" language={activePlatform === 'windows' ? 'PowerShell / CMD' : 'bash'} />
                  </div>
                </div>
                <div className="border-t border-[var(--nv-border-subtle)] pt-4">
                  <h4 className="text-sm font-medium">Check Agent Status</h4>
                  <div className="mt-2">
                    <CodeBlock code="scout status" language={activePlatform === 'windows' ? 'PowerShell / CMD' : 'bash'} />
                  </div>
                </div>
                <div className="border-t border-[var(--nv-border-subtle)] pt-4">
                  <h4 className="text-sm font-medium">System Requirements</h4>
                  <ul className="mt-2 space-y-1 text-sm text-muted-foreground">
                    <li>- Minimal resource usage (5-15 MB RAM)</li>
                    <li>- Runs as a background service</li>
                    <li>- Linux, macOS, and Windows supported</li>
                  </ul>
                </div>
                <a
                  href="https://github.com/HerbHall/subnetree/releases"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-sm text-green-500 hover:text-green-400 transition-colors mt-2"
                >
                  View all releases
                  <ExternalLink className="h-3.5 w-3.5" />
                </a>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
