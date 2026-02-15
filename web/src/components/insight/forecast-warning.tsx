import { useQuery } from '@tanstack/react-query'
import { TrendingUp } from 'lucide-react'
import { getDeviceForecasts } from '@/api/insight'
import type { Forecast } from '@/api/insight'

const THIRTY_DAYS_NS = 30 * 24 * 60 * 60 * 1e9

function formatTimeToThreshold(ns: number): string {
  const hours = Math.floor(ns / 3.6e12)
  if (hours < 24) return `${hours} hour${hours !== 1 ? 's' : ''}`
  const days = Math.floor(hours / 24)
  return `${days} day${days !== 1 ? 's' : ''}`
}

interface ForecastWarningProps {
  deviceId: string
}

export function ForecastWarning({ deviceId }: ForecastWarningProps) {
  const { data: forecasts } = useQuery({
    queryKey: ['insight', 'forecasts', deviceId],
    queryFn: () => getDeviceForecasts(deviceId),
    refetchInterval: 300000,
  })

  const concerning = forecasts?.filter(
    (f: Forecast) =>
      f.time_to_threshold != null &&
      f.time_to_threshold > 0 &&
      f.time_to_threshold <= THIRTY_DAYS_NS
  ) ?? []

  if (concerning.length === 0) return null

  return (
    <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4">
      <div className="flex items-center gap-2 mb-2">
        <TrendingUp className="h-4 w-4 text-amber-500" />
        <p className="text-sm font-medium text-amber-600 dark:text-amber-400">
          Capacity Forecast Warning
        </p>
      </div>
      <div className="space-y-1.5">
        {concerning.map((forecast: Forecast) => (
          <div
            key={`${forecast.device_id}-${forecast.metric_name}`}
            className="flex items-center justify-between text-sm"
          >
            <span className="text-muted-foreground">
              <span className="font-medium text-foreground">
                {forecast.metric_name}
              </span>
              {' '}at {forecast.current_value.toFixed(1)}%, projected to breach{' '}
              {forecast.threshold.toFixed(0)}% threshold
            </span>
            <span className="text-amber-600 dark:text-amber-400 font-medium shrink-0 ml-3">
              in {formatTimeToThreshold(forecast.time_to_threshold!)}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
