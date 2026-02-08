import { useQuery } from '@tanstack/react-query'
import {
  Info,
  GitCommit,
  Calendar,
  Code2,
  Monitor,
  Cpu,
  Scale,
  ExternalLink,
  Github,
  MessageSquare,
  Bug,
  BookOpen,
  Heart,
  Coffee,
  Loader2,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { getHealth } from '@/api/system'

export function AboutPage() {
  const {
    data: health,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    staleTime: 60 * 1000,
  })

  const versionInfo = health?.version

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold">About SubNetree</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Version information, licensing, and community links
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Version Info */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Info className="h-4 w-4 text-muted-foreground" />
              Version Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : error ? (
              <p className="text-sm text-red-400 py-4">
                Failed to load version information.
              </p>
            ) : versionInfo ? (
              <div className="space-y-3">
                <VersionRow
                  icon={Monitor}
                  label="Version"
                  value={versionInfo.version}
                />
                <VersionRow
                  icon={GitCommit}
                  label="Commit"
                  value={versionInfo.git_commit.substring(0, 8)}
                  title={versionInfo.git_commit}
                />
                <VersionRow
                  icon={Calendar}
                  label="Build Date"
                  value={formatBuildDate(versionInfo.build_date)}
                  title={versionInfo.build_date}
                />
                <VersionRow
                  icon={Code2}
                  label="Go Version"
                  value={versionInfo.go_version}
                />
                <VersionRow
                  icon={Cpu}
                  label="Platform"
                  value={`${versionInfo.os}/${versionInfo.arch}`}
                />
              </div>
            ) : null}
          </CardContent>
        </Card>

        {/* License */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Scale className="h-4 w-4 text-muted-foreground" />
              License
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <h4 className="text-sm font-medium">Core</h4>
                <p className="text-sm text-muted-foreground mt-1">
                  Business Source License 1.1 (BSL 1.1). Free for personal,
                  homelab, educational, and non-competing production use.
                  Converts to Apache 2.0 after 4 years.
                </p>
              </div>
              <div className="border-t pt-4">
                <h4 className="text-sm font-medium">Plugin SDK</h4>
                <p className="text-sm text-muted-foreground mt-1">
                  Apache License 2.0. Build plugins and integrations with no
                  restrictions.
                </p>
              </div>
              <a
                href="https://github.com/HerbHall/subnetree/blob/main/LICENSING.md"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-green-500 hover:text-green-400 transition-colors mt-2"
              >
                View full licensing details
                <ExternalLink className="h-3.5 w-3.5" />
              </a>
            </div>
          </CardContent>
        </Card>

        {/* Links */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Github className="h-4 w-4 text-muted-foreground" />
              Community & Links
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <LinkRow
                icon={Github}
                label="GitHub Repository"
                href="https://github.com/HerbHall/subnetree"
              />
              <LinkRow
                icon={MessageSquare}
                label="Discussions"
                href="https://github.com/HerbHall/subnetree/discussions"
                description="Questions, ideas, and general chat"
              />
              <LinkRow
                icon={Bug}
                label="Issue Tracker"
                href="https://github.com/HerbHall/subnetree/issues"
                description="Bug reports and feature requests"
              />
              <LinkRow
                icon={BookOpen}
                label="Contributing Guide"
                href="https://github.com/HerbHall/subnetree/blob/main/CONTRIBUTING.md"
                description="How to get involved"
              />
            </div>
          </CardContent>
        </Card>

        {/* Support */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Heart className="h-4 w-4 text-muted-foreground" />
              Support the Project
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">
              SubNetree is free for personal and homelab use. If you find it
              useful, consider supporting development.
            </p>
            <div className="space-y-2">
              <LinkRow
                icon={Github}
                label="GitHub Sponsors"
                href="https://github.com/sponsors/HerbHall"
              />
              <LinkRow
                icon={Coffee}
                label="Ko-fi"
                href="https://ko-fi.com/herbhall"
              />
              <LinkRow
                icon={Coffee}
                label="Buy Me a Coffee"
                href="https://buymeacoffee.com/herbhall"
              />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function VersionRow({
  icon: Icon,
  label,
  value,
  title,
}: {
  icon: React.ElementType
  label: string
  value: string
  title?: string
}) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm text-muted-foreground">{label}</span>
      </div>
      <span className="text-sm font-medium font-mono" title={title}>
        {value}
      </span>
    </div>
  )
}

function LinkRow({
  icon: Icon,
  label,
  href,
  description,
}: {
  icon: React.ElementType
  label: string
  href: string
  description?: string
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50 transition-colors group"
    >
      <Icon className="h-4 w-4 text-muted-foreground" />
      <div className="flex-1 min-w-0">
        <span className="text-sm font-medium group-hover:text-green-500 transition-colors">
          {label}
        </span>
        {description && (
          <p className="text-xs text-muted-foreground">{description}</p>
        )}
      </div>
      <ExternalLink className="h-3.5 w-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
    </a>
  )
}

function formatBuildDate(dateStr: string): string {
  if (dateStr === 'unknown') return 'unknown'
  try {
    const date = new Date(dateStr)
    return date.toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  } catch {
    return dateStr
  }
}
