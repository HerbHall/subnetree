import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  type TooltipContentProps,
} from 'recharts'
import { Skeleton } from '@/components/ui/skeleton'
import type { MetricName, MetricRange } from '@/api/types'

interface TimeSeriesChartProps {
  data: Array<{ timestamp: string; value: number }>
  metric: MetricName
  range: MetricRange
  loading?: boolean
  className?: string
}

const metricConfig: Record<MetricName, { unit: string; label: string; color: string; fillColor: string }> = {
  latency: {
    unit: 'ms',
    label: 'Latency',
    color: 'var(--nv-chart-blue)',
    fillColor: 'var(--nv-chart-blue)',
  },
  packet_loss: {
    unit: '%',
    label: 'Packet Loss',
    color: 'var(--nv-chart-amber)',
    fillColor: 'var(--nv-chart-amber)',
  },
  success_rate: {
    unit: '%',
    label: 'Success Rate',
    color: 'var(--nv-chart-green)',
    fillColor: 'var(--nv-chart-green)',
  },
}

/** Ranges where we show time-of-day vs date formatting. */
const shortRanges: MetricRange[] = ['1h', '6h', '24h']

function formatXAxisTick(timestamp: string, range: MetricRange): string {
  const date = new Date(timestamp)
  if (shortRanges.includes(range)) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`
}

function CustomTooltip({
  active,
  payload,
  label,
  metric,
}: Partial<TooltipContentProps<number, string>> & { metric: MetricName }) {
  if (!active || !payload || payload.length === 0) return null

  const cfg = metricConfig[metric]
  const displayValue = payload[0].value as number
  const date = new Date(label as string)

  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground mb-1">
        {date.toLocaleString()}
      </p>
      <p className="text-sm font-medium">
        {displayValue.toFixed(1)} {cfg.unit}
      </p>
    </div>
  )
}

function transformData(data: Array<{ timestamp: string; value: number }>, metric: MetricName) {
  return data.map((point) => ({
    ...point,
    displayValue: metric === 'packet_loss' ? point.value * 100 : point.value,
  }))
}

export function TimeSeriesChart({ data, metric, range, loading, className }: TimeSeriesChartProps) {
  const cfg = metricConfig[metric]

  if (loading) {
    return (
      <div className={className}>
        <Skeleton className="w-full h-[300px]" />
      </div>
    )
  }

  if (!data || data.length === 0) {
    return (
      <div className={className}>
        <div className="flex items-center justify-center h-[300px] text-muted-foreground text-sm">
          No metric data available for this time range.
        </div>
      </div>
    )
  }

  const chartData = transformData(data, metric)

  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={chartData} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid
            strokeDasharray="3 3"
            stroke="currentColor"
            strokeOpacity={0.08}
          />
          <XAxis
            dataKey="timestamp"
            tickFormatter={(ts: string) => formatXAxisTick(ts, range)}
            tick={{ fill: 'currentColor', fontSize: 12 }}
            tickLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
            axisLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
            minTickGap={40}
          />
          <YAxis
            tick={{ fill: 'currentColor', fontSize: 12 }}
            tickLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
            axisLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
            label={{
              value: cfg.unit,
              angle: -90,
              position: 'insideLeft',
              style: { fill: 'currentColor', fontSize: 12 },
            }}
            width={50}
          />
          <Tooltip content={<CustomTooltip metric={metric} />} />
          <Area
            type="monotone"
            dataKey="displayValue"
            stroke={cfg.color}
            fill={cfg.fillColor}
            fillOpacity={0.15}
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4, strokeWidth: 2 }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
