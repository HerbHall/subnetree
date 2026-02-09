import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function ThemePreview() {
  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-muted-foreground">Live Preview</h3>

      {/* Sample card */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm">Sample Card</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-foreground">
            Primary text with <span className="text-accent">accent</span> color.
          </p>
          <p className="text-xs text-muted-foreground">
            Secondary muted text for descriptions.
          </p>

          {/* Buttons */}
          <div className="flex gap-2">
            <Button size="sm">Primary</Button>
            <Button size="sm" variant="outline">Outline</Button>
            <Button size="sm" variant="destructive">Danger</Button>
            <Button size="sm" variant="ghost">Ghost</Button>
          </div>

          {/* Input */}
          <Input placeholder="Sample input field..." className="h-8 text-sm" readOnly />

          {/* Status badges */}
          <div className="flex gap-2 flex-wrap">
            <StatusBadge label="Online" colorVar="--nv-status-online" />
            <StatusBadge label="Degraded" colorVar="--nv-status-degraded" />
            <StatusBadge label="Offline" colorVar="--nv-status-offline" />
            <StatusBadge label="Unknown" colorVar="--nv-status-unknown" />
          </div>

          {/* Chart color swatches */}
          <div className="space-y-1.5">
            <p className="text-xs text-muted-foreground font-medium">Chart Colors</p>
            <div className="flex gap-1.5">
              <ChartSwatch colorVar="--nv-chart-green" />
              <ChartSwatch colorVar="--nv-chart-amber" />
              <ChartSwatch colorVar="--nv-chart-sage" />
              <ChartSwatch colorVar="--nv-chart-red" />
              <ChartSwatch colorVar="--nv-chart-blue" />
            </div>
          </div>

          {/* Sidebar preview */}
          <div className="rounded-md overflow-hidden border">
            <div
              className="p-2 space-y-1"
              style={{ backgroundColor: 'var(--nv-sidebar-bg)' }}
            >
              <div
                className="px-2 py-1 rounded text-xs"
                style={{ color: 'var(--nv-sidebar-item)' }}
              >
                Dashboard
              </div>
              <div
                className="px-2 py-1 rounded text-xs font-medium"
                style={{
                  color: 'var(--nv-sidebar-active)',
                  backgroundColor: 'var(--nv-sidebar-active-bg)',
                }}
              >
                Devices
              </div>
              <div
                className="px-2 py-1 rounded text-xs"
                style={{ color: 'var(--nv-sidebar-item)' }}
              >
                Settings
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function StatusBadge({
  label,
  colorVar,
}: {
  label: string
  colorVar: string
}) {
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[10px] font-medium"
      style={{
        backgroundColor: `color-mix(in srgb, var(${colorVar}) 15%, transparent)`,
        color: `var(${colorVar})`,
      }}
    >
      <span
        className="h-1.5 w-1.5 rounded-full"
        style={{ backgroundColor: `var(${colorVar})` }}
      />
      {label}
    </span>
  )
}

function ChartSwatch({ colorVar }: { colorVar: string }) {
  return (
    <div
      className="h-6 w-6 rounded"
      style={{ backgroundColor: `var(${colorVar})` }}
      title={colorVar.replace('--nv-', '')}
    />
  )
}
