/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { Activity, AlertTriangle, Database } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { CartesianGrid, ComposedChart, Line, XAxis, YAxis } from 'recharts'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/ui/chart'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { getTokenTrend } from '../api'
import { getTokenTrendApplicability } from '../lib/query'
import type {
  TokenTrendData,
  TokenTrendMetrics,
  TokenTrendPanelProps,
} from '../types'

const METRIC_CONFIG = {
  input_tokens: {
    color: 'var(--chart-1)',
    swatchClassName: 'bg-chart-1',
  },
  output_tokens: {
    color: 'var(--chart-2)',
    swatchClassName: 'bg-chart-2',
  },
  cache_creation_tokens: {
    color: 'var(--chart-4)',
    swatchClassName: 'bg-chart-4',
  },
  cache_read_tokens: {
    color: 'var(--chart-5)',
    swatchClassName: 'bg-chart-5',
  },
  cache_hit_rate: {
    color: 'var(--chart-3)',
    swatchClassName: 'bg-chart-3',
  },
} as const

type MetricKey = keyof typeof METRIC_CONFIG

function TokenTrendSkeleton() {
  return (
    <div className='flex flex-col gap-4' aria-busy='true'>
      <div className='grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5'>
        {Array.from({ length: 5 }, (_, index) => (
          <div key={index} className='flex flex-col gap-2 py-1'>
            <Skeleton className='h-3 w-20' />
            <Skeleton className='h-6 w-24' />
          </div>
        ))}
      </div>
      <Skeleton className='h-[260px] w-full rounded-lg' />
    </div>
  )
}

function TokenTrendUnavailableNotice(props: { reason: string }) {
  const { t } = useTranslation()
  let description = t('Token trend is currently unavailable.')
  if (props.reason === 'consume_logging_disabled') {
    description = t(
      'Token trend is unavailable because consume logging is disabled.'
    )
  } else if (props.reason === 'data_export_disabled') {
    description = t(
      'Token trend is unavailable because data export is disabled.'
    )
  }

  return (
    <TokenTrendNotice
      title={t('Token trend unavailable')}
      description={description}
    />
  )
}

function formatMetricValue(
  key: MetricKey,
  rawValue: number | null,
  numberFormatter: Intl.NumberFormat,
  percentFormatter: Intl.NumberFormat
): string {
  if (rawValue == null) return '--'
  if (key === 'cache_hit_rate') {
    return percentFormatter.format(rawValue / 100)
  }
  return numberFormatter.format(rawValue)
}

function TokenTrendNotice(props: {
  title: string
  description: string
  destructive?: boolean
}) {
  return (
    <Alert variant={props.destructive ? 'destructive' : 'default'}>
      <AlertTriangle aria-hidden='true' />
      <AlertTitle>{props.title}</AlertTitle>
      <AlertDescription>{props.description}</AlertDescription>
    </Alert>
  )
}

function TokenTrendSummary(props: {
  totals: TokenTrendMetrics
  chartConfig: ChartConfig
  numberFormatter: Intl.NumberFormat
  percentFormatter: Intl.NumberFormat
}) {
  return (
    <div
      className='bg-muted/30 grid grid-cols-2 gap-x-3 gap-y-4 rounded-lg p-3 sm:grid-cols-3 lg:grid-cols-5'
      role='list'
    >
      {(Object.keys(METRIC_CONFIG) as MetricKey[]).map((key) => {
        const rawValue = props.totals[key]
        const value = formatMetricValue(
          key,
          rawValue,
          props.numberFormatter,
          props.percentFormatter
        )

        return (
          <div key={key} className='min-w-0' role='listitem'>
            <div className='flex min-w-0 items-center gap-2'>
              <span
                className={cn(
                  'size-2 shrink-0 rounded-full',
                  METRIC_CONFIG[key].swatchClassName
                )}
                aria-hidden='true'
              />
              <span className='text-muted-foreground text-xs leading-tight'>
                {props.chartConfig[key]?.label}
              </span>
            </div>
            <div className='mt-1 truncate text-lg font-semibold tabular-nums'>
              {value}
            </div>
          </div>
        )
      })}
    </div>
  )
}

function TokenTrendChart(props: {
  data: TokenTrendData
  chartConfig: ChartConfig
  axisFormatter: Intl.DateTimeFormat
  tooltipFormatter: Intl.DateTimeFormat
  compactNumberFormatter: Intl.NumberFormat
  numberFormatter: Intl.NumberFormat
  percentFormatter: Intl.NumberFormat
  ariaLabel: string
}) {
  return (
    <ChartContainer
      config={props.chartConfig}
      className='aspect-auto h-[260px] w-full sm:h-[300px]'
      initialDimension={{ width: 720, height: 300 }}
      role='img'
      aria-label={props.ariaLabel}
    >
      <ComposedChart
        data={props.data.points}
        accessibilityLayer
        margin={{ top: 8, right: 4, left: 0, bottom: 0 }}
      >
        <CartesianGrid vertical={false} strokeDasharray='3 3' />
        <XAxis
          dataKey='timestamp'
          axisLine={false}
          tickLine={false}
          tickMargin={10}
          minTickGap={28}
          tickFormatter={(timestamp: number) =>
            props.axisFormatter.format(new Date(timestamp * 1000))
          }
        />
        <YAxis
          yAxisId='tokens'
          axisLine={false}
          tickLine={false}
          tickMargin={8}
          width={44}
          tickFormatter={(value: number) =>
            props.compactNumberFormatter.format(value)
          }
        />
        <YAxis
          yAxisId='rate'
          orientation='right'
          domain={[0, 100]}
          axisLine={false}
          tickLine={false}
          tickMargin={8}
          width={40}
          tickFormatter={(value: number) => `${Math.round(value)}%`}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              labelFormatter={(_label, payload) => {
                const timestamp = Number(payload[0]?.payload?.timestamp)
                return Number.isFinite(timestamp)
                  ? props.tooltipFormatter.format(new Date(timestamp * 1000))
                  : ''
              }}
              formatter={(value, name, item) => {
                const key = String(name) as MetricKey
                const isRate = key === 'cache_hit_rate'
                const formattedValue = isRate
                  ? props.percentFormatter.format(Number(value) / 100)
                  : props.numberFormatter.format(Number(value))
                return (
                  <>
                    <span
                      className='size-2 shrink-0 rounded-full'
                      style={{ backgroundColor: item.color }}
                      aria-hidden='true'
                    />
                    <span className='text-muted-foreground flex-1'>
                      {props.chartConfig[key]?.label ?? name}
                    </span>
                    <span className='text-foreground font-mono font-medium tabular-nums'>
                      {formattedValue}
                    </span>
                  </>
                )
              }}
            />
          }
        />
        {(Object.keys(METRIC_CONFIG) as MetricKey[]).map((key) => (
          <Line
            key={key}
            dataKey={key}
            name={key}
            type='monotone'
            yAxisId={key === 'cache_hit_rate' ? 'rate' : 'tokens'}
            stroke={`var(--color-${key})`}
            strokeWidth={key === 'cache_hit_rate' ? 2.5 : 2}
            strokeDasharray={key === 'cache_hit_rate' ? '6 4' : undefined}
            dot={false}
            activeDot={{ r: 4 }}
            connectNulls={false}
          />
        ))}
      </ComposedChart>
    </ChartContainer>
  )
}

export function TokenTrendPanel(props: TokenTrendPanelProps) {
  const { t } = useTranslation()
  const applicability = getTokenTrendApplicability(props.filters)
  const isApplicable = applicability.kind === 'applicable'
  const timezoneOffset = new Date().getTimezoneOffset()
  const query = useQuery({
    queryKey: [
      'token-trend',
      props.scope,
      props.filters.startTimestamp,
      props.filters.endTimestamp,
      props.filters.model,
      props.filters.token,
      props.filters.group,
      props.filters.channel,
      timezoneOffset,
    ],
    queryFn: () => getTokenTrend(props.scope, props.filters, timezoneOffset),
    enabled: isApplicable,
  })

  const formatters = useMemo(() => {
    const language = 'zh-CN'
    const duration = props.filters.endTimestamp - props.filters.startTimestamp
    let axisOptions: Intl.DateTimeFormatOptions
    if (duration <= 24 * 60 * 60) {
      axisOptions = { hour: '2-digit', minute: '2-digit' }
    } else if (duration <= 48 * 60 * 60) {
      axisOptions = {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }
    } else {
      axisOptions = { month: 'short', day: 'numeric' }
    }
    return {
      axis: new Intl.DateTimeFormat(language, axisOptions),
      tooltip: new Intl.DateTimeFormat(language, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }),
      compactNumber: new Intl.NumberFormat(language, {
        notation: 'compact',
        maximumFractionDigits: 1,
      }),
      number: new Intl.NumberFormat(language),
      percent: new Intl.NumberFormat(language, {
        style: 'percent',
        maximumFractionDigits: 1,
      }),
    }
  }, [props.filters.endTimestamp, props.filters.startTimestamp])

  const chartConfig = useMemo<ChartConfig>(
    () => ({
      input_tokens: {
        label: t('Input Tokens'),
        color: METRIC_CONFIG.input_tokens.color,
      },
      output_tokens: {
        label: t('Output Tokens'),
        color: METRIC_CONFIG.output_tokens.color,
      },
      cache_creation_tokens: {
        label: t('Cache Creation'),
        color: METRIC_CONFIG.cache_creation_tokens.color,
      },
      cache_read_tokens: {
        label: t('Cache Read'),
        color: METRIC_CONFIG.cache_read_tokens.color,
      },
      cache_hit_rate: {
        label: t('Cache Hit Rate'),
        color: METRIC_CONFIG.cache_hit_rate.color,
      },
    }),
    [t]
  )

  let content
  if (applicability.kind === 'exact-request') {
    content = (
      <TokenTrendNotice
        title={t('Token trend not applicable')}
        description={t(
          'Token trend is not available for exact request filters.'
        )}
      />
    )
  } else if (applicability.kind === 'unsupported-type') {
    content = (
      <TokenTrendNotice
        title={t('Token trend not applicable')}
        description={t(
          'Token trend is only available for all or consume log types.'
        )}
      />
    )
  } else if (query.isLoading) {
    content = <TokenTrendSkeleton />
  } else if (query.isError) {
    content = (
      <ErrorState
        className='min-h-[260px]'
        title={t('Failed to load token trend')}
        description={t('Please try again later.')}
        onRetry={() => void query.refetch()}
      />
    )
  } else if (query.data && !query.data.available) {
    content = <TokenTrendUnavailableNotice reason={query.data.reason} />
  } else if (!query.data || query.data.points.length === 0) {
    content = (
      <EmptyState
        icon={Database}
        className='min-h-[260px]'
        title={t('No token trend data')}
        description={t(
          'Token metrics will appear after new consume requests are recorded.'
        )}
      />
    )
  } else {
    content = (
      <div className='flex flex-col gap-4'>
        <TokenTrendSummary
          totals={query.data.totals}
          chartConfig={chartConfig}
          numberFormatter={formatters.number}
          percentFormatter={formatters.percent}
        />
        <TokenTrendChart
          data={query.data}
          chartConfig={chartConfig}
          axisFormatter={formatters.axis}
          tooltipFormatter={formatters.tooltip}
          compactNumberFormatter={formatters.compactNumber}
          numberFormatter={formatters.number}
          percentFormatter={formatters.percent}
          ariaLabel={t('Token usage trend chart')}
        />
      </div>
    )
  }

  const trackingStartedAt = query.data?.tracking_started_at

  return (
    <Card className={props.className}>
      <CardHeader>
        <CardTitle className='flex items-center gap-2'>
          <Activity className='text-chart-1 size-4' aria-hidden='true' />
          {t('Token Usage Trend')}
        </CardTitle>
        <CardDescription>
          {t('Input, output, and cache token usage over time.')}
        </CardDescription>
        {query.data?.available && (
          <div className='text-muted-foreground mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs'>
            {trackingStartedAt != null && (
              <span>
                {t('Tracking since {{time}}', {
                  time: formatters.tooltip.format(
                    new Date(trackingStartedAt * 1000)
                  ),
                })}
              </span>
            )}
            <span>
              {t('{{count}} tracked requests', {
                count: formatters.number.format(
                  query.data.totals.tracked_requests
                ),
              })}
            </span>
          </div>
        )}
      </CardHeader>
      <CardContent>{content}</CardContent>
      <span className='sr-only'>
        {query.data?.available && query.data.points.length > 0
          ? t(
              'Token trend includes input, output, cache creation, cache read, and cache hit rate.'
            )
          : ''}
      </span>
    </Card>
  )
}
