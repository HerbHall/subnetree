import { ResponsiveContainer, LineChart, Line, YAxis } from 'recharts'

interface SparklineChartProps {
  data: Array<{ timestamp: string; value: number }>
  color?: string
  height?: number
  width?: number
  className?: string
}

export function SparklineChart({
  data,
  color = 'var(--nv-chart-green)',
  height = 32,
  width = 120,
  className,
}: SparklineChartProps) {
  if (!data || data.length < 2) {
    return (
      <div
        className={className}
        style={{ width, height }}
      >
        <div className="flex items-center justify-center h-full">
          <div className="w-full border-t border-dashed border-muted-foreground/30" />
        </div>
      </div>
    )
  }

  return (
    <div className={className} style={{ width, height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
          <YAxis hide domain={['dataMin', 'dataMax']} />
          <Line
            type="monotone"
            dataKey="value"
            stroke={color}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
