import { useState } from 'react'
import { Download, Terminal, Copy, Check, ExternalLink } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Platform = 'linux' | 'macos' | 'windows'

interface ArchOption {
  label: string
  arch: string
  filename: string
}

interface PlatformConfig {
  label: string
  icon: string
  architectures: ArchOption[]
  installCommands: (filename: string) => string
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const GITHUB_RELEASE_BASE =
  'https://github.com/HerbHall/subnetree/releases/latest/download'

const PLATFORMS: Record<Platform, PlatformConfig> = {
  linux: {
    label: 'Linux',
    icon: 'Terminal',
    architectures: [
      { label: 'x86_64 (amd64)', arch: 'amd64', filename: 'scout_linux_amd64' },
      { label: 'ARM64 (aarch64)', arch: 'arm64', filename: 'scout_linux_arm64' },
    ],
    installCommands: (filename: string) =>
      `curl -LO ${GITHUB_RELEASE_BASE}/${filename}\nchmod +x ${filename}\nsudo mv ${filename} /usr/local/bin/scout`,
  },
  macos: {
    label: 'macOS',
    icon: 'Apple',
    architectures: [
      { label: 'Apple Silicon (arm64)', arch: 'arm64', filename: 'scout_darwin_arm64' },
      { label: 'Intel (amd64)', arch: 'amd64', filename: 'scout_darwin_amd64' },
    ],
    installCommands: (filename: string) =>
      `curl -LO ${GITHUB_RELEASE_BASE}/${filename}\nchmod +x ${filename}\nmv ${filename} /usr/local/bin/scout`,
  },
  windows: {
    label: 'Windows',
    icon: 'Monitor',
    architectures: [
      { label: 'x86_64 (amd64)', arch: 'amd64', filename: 'scout_windows_amd64.exe' },
    ],
    installCommands: (filename: string) =>
      `Invoke-WebRequest -Uri "${GITHUB_RELEASE_BASE}/${filename}" -OutFile scout.exe\nNew-Item -ItemType Directory -Force -Path "$env:ProgramFiles\\SubNetree" | Out-Null\nMove-Item scout.exe "$env:ProgramFiles\\SubNetree\\scout.exe"`,
  },
}

const ENROLLMENT_COMMAND =
  'scout enroll --server https://<server-ip>:9090 --token <enrollment-token>'

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
// Components
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
      <div className="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity">
        <CopyButton text={code} />
      </div>
      <pre className="rounded-md bg-black/40 border border-[var(--nv-border-subtle)] p-4 overflow-x-auto">
        <code className="text-xs font-mono text-muted-foreground whitespace-pre">
          <span className="sr-only">{language}</span>
          {code}
        </code>
      </pre>
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

function ArchitectureSection({
  archOption,
  platformConfig,
}: {
  archOption: ArchOption
  platformConfig: PlatformConfig
}) {
  const installCommands = platformConfig.installCommands(archOption.filename)
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
      <CodeBlock
        code={installCommands}
        language={platformConfig.label === 'Windows' ? 'powershell' : 'bash'}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function AgentSetupPage() {
  const [activePlatform, setActivePlatform] = useState<Platform>(detectPlatform)
  const serverHost = window.location.hostname

  const platformConfig = PLATFORMS[activePlatform]
  const enrollmentCommand = ENROLLMENT_COMMAND.replace(
    '<server-ip>',
    serverHost || '<server-ip>'
  )

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold">Agent Setup</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Download and install the Scout agent on devices you want to monitor
        </p>
      </div>

      {/* Platform selector */}
      <div className="flex gap-2 rounded-lg border border-[var(--nv-border-subtle)] bg-card p-1 w-fit">
        {(Object.entries(PLATFORMS) as [Platform, PlatformConfig][]).map(
          ([key, config]) => (
            <PlatformTab
              key={key}
              platform={config}
              isActive={activePlatform === key}
              onClick={() => setActivePlatform(key)}
            />
          )
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Install instructions */}
        <Card className="lg:col-span-2">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Download className="h-4 w-4 text-muted-foreground" />
              Install Scout for {platformConfig.label}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              {platformConfig.architectures.map((archOption) => (
                <ArchitectureSection
                  key={archOption.arch}
                  archOption={archOption}
                  platformConfig={platformConfig}
                />
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Enrollment */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Terminal className="h-4 w-4 text-muted-foreground" />
              Enroll Agent
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                After installing Scout, enroll it with your SubNetree server.
                Replace <code className="text-xs bg-black/30 rounded px-1 py-0.5">&lt;enrollment-token&gt;</code> with
                the token from the Agents page.
              </p>
              <CodeBlock code={enrollmentCommand} language="bash" />
              <p className="text-xs text-muted-foreground">
                The agent will establish a secure mTLS connection to the server
                on port 9090 and begin reporting system metrics.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Quick reference */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Terminal className="h-4 w-4 text-muted-foreground" />
              Quick Reference
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <h4 className="text-sm font-medium">Verify Installation</h4>
                <div className="mt-2">
                  <CodeBlock code="scout --version" language="bash" />
                </div>
              </div>
              <div className="border-t border-[var(--nv-border-subtle)] pt-4">
                <h4 className="text-sm font-medium">Check Agent Status</h4>
                <div className="mt-2">
                  <CodeBlock code="scout status" language="bash" />
                </div>
              </div>
              <div className="border-t border-[var(--nv-border-subtle)] pt-4">
                <h4 className="text-sm font-medium">
                  System Requirements
                </h4>
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
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
