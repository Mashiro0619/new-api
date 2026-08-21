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
import { RefreshCw } from 'lucide-react'
import { useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

import { getSiteModelCallModels, getSiteModelCallSummary } from './api'
import { parseSiteModelCallsConfig } from './config'

const numberFormatter = new Intl.NumberFormat()

function formatNullableNumber(value: number | null): string {
  return value == null ? '--' : numberFormatter.format(value)
}

function formatNullablePercent(value: number | null): string {
  return value == null ? '--' : `${value.toFixed(2)}%`
}

export function SiteModelCalls() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const summaryQuery = useQuery({
    queryKey: ['site-model-calls', 'summary'],
    queryFn: getSiteModelCallSummary,
  })
  const modelsQuery = useQuery({
    queryKey: ['site-model-calls', 'models'],
    queryFn: getSiteModelCallModels,
  })

  const rows = useMemo(() => {
    const byName = new Map(
      (summaryQuery.data?.data ?? []).map((row) => [row.model_name, row])
    )
    const configuredModels = parseSiteModelCallsConfig(
      status?.AllSiteModelCalls
    ).models
    const modelNames =
      configuredModels.length > 0
        ? configuredModels
        : (modelsQuery.data?.data ?? [])
    return modelNames
      .map(
        (modelName) =>
          byName.get(modelName) ?? {
            model_name: modelName,
            request_count: 0,
            success_count: 0,
            failure_count: 0,
            success_rate: 0,
            total_tokens: null,
            cache_read_tokens: null,
            cache_hit_rate: null,
          }
      )
      .sort((a, b) => {
        if (a.request_count !== b.request_count) {
          return b.request_count - a.request_count
        }
        return a.model_name.localeCompare(b.model_name)
      })
  }, [
    modelsQuery.data?.data,
    status?.AllSiteModelCalls,
    summaryQuery.data?.data,
  ])

  const isLoading = summaryQuery.isLoading || modelsQuery.isLoading
  const error = summaryQuery.error ?? modelsQuery.error
  const refresh = () => {
    void Promise.all([summaryQuery.refetch(), modelsQuery.refetch()])
  }

  let body: ReactNode
  if (error) {
    body = (
      <div className='text-destructive flex flex-1 items-center justify-center text-sm'>
        {t('Failed to load model call statistics')}
      </div>
    )
  } else if (isLoading) {
    body = (
      <div className='text-muted-foreground flex flex-1 items-center justify-center text-sm'>
        {t('Loading...')}
      </div>
    )
  } else if (rows.length === 0) {
    body = (
      <div className='text-muted-foreground flex flex-1 items-center justify-center text-sm'>
        {t('No model call data')}
      </div>
    )
  } else {
    body = (
      <div className='min-h-0 flex-1 overflow-auto'>
        <StaticDataTable
          data={rows}
          getRowKey={(row) => row.model_name}
          columns={[
            {
              id: 'model',
              header: t('Model'),
              cellClassName: 'font-medium',
              cell: (row) => row.model_name,
            },
            {
              id: 'success',
              header: t('Success count'),
              className: 'text-right',
              cellClassName: 'text-right tabular-nums',
              cell: (row) => numberFormatter.format(row.success_count),
            },
            {
              id: 'failure',
              header: t('Failure count'),
              className: 'text-right',
              cellClassName: 'text-right tabular-nums',
              cell: (row) => numberFormatter.format(row.failure_count),
            },
            {
              id: 'total',
              header: t('Total calls'),
              className: 'text-right',
              cellClassName: 'text-right tabular-nums',
              cell: (row) => numberFormatter.format(row.request_count),
            },
            {
              id: 'rate',
              header: t('Success rate'),
              className: 'text-right',
              cellClassName: 'text-right tabular-nums',
              cell: (row) => `${row.success_rate.toFixed(2)}%`,
            },
            {
              id: 'tokens',
              header: t('Total Tokens'),
              className: 'text-right',
              cellClassName: 'text-right tabular-nums',
              cell: (row) => formatNullableNumber(row.total_tokens),
            },
            {
              id: 'cache-rate',
              header: t('Cache Hit Rate'),
              className: 'text-right',
              cellClassName: 'text-right tabular-nums',
              cell: (row) => formatNullablePercent(row.cache_hit_rate),
            },
          ]}
        />
      </div>
    )
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('All-site model calls')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='icon'
          onClick={refresh}
          disabled={isLoading}
          aria-label={t('Refresh')}
          title={t('Refresh')}
        >
          <RefreshCw className={isLoading ? 'animate-spin' : ''} />
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-3'>
          <p className='text-muted-foreground text-sm'>
            {t('Cumulative calls retained by performance monitoring.')}
          </p>
          {body}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
