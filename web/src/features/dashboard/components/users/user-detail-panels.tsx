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
import { useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatCompactNumber, formatQuota } from '@/lib/format'

import type { DashboardLogStatistics, QuotaDataItem } from '../../types'

interface UserStatStripProps {
  stats?: DashboardLogStatistics
  rows?: QuotaDataItem[]
  loading: boolean
  error?: boolean
  onRetry?: () => void
}

export function UserStatStrip(props: UserStatStripProps) {
  const { t } = useTranslation()
  const totals = useMemo(
    () =>
      (props.rows ?? []).reduce(
        (result, row) => ({
          requests: result.requests + Number(row.count ?? 0),
          tokens: result.tokens + Number(row.token_used ?? 0),
        }),
        { requests: 0, tokens: 0 }
      ),
    [props.rows]
  )
  const items = [
    { label: t('Requests'), value: formatCompactNumber(totals.requests) },
    { label: t('Tokens'), value: formatCompactNumber(totals.tokens) },
    { label: t('Usage'), value: formatQuota(props.stats?.quota ?? 0) },
    { label: t('RPM'), value: formatCompactNumber(props.stats?.rpm ?? 0) },
    { label: t('TPM'), value: formatCompactNumber(props.stats?.tpm ?? 0) },
  ]
  if (props.error) {
    return (
      <div className='overflow-hidden rounded-lg border'>
        <ErrorState
          className='min-h-40'
          title={t('Failed to load')}
          description={t('Please try again later.')}
          onRetry={props.onRetry}
        />
      </div>
    )
  }
  return (
    <div className='divide-border/60 grid grid-cols-2 overflow-hidden rounded-lg border sm:grid-cols-3 lg:grid-cols-5 lg:divide-x'>
      {items.map((item) => (
        <div key={item.label} className='min-w-0 px-4 py-3'>
          <div className='text-muted-foreground text-xs font-medium'>
            {item.label}
          </div>
          {props.loading ? (
            <Skeleton className='mt-2 h-6 w-20' />
          ) : (
            <div className='mt-1 truncate font-mono text-lg font-semibold tabular-nums'>
              {item.value}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

export function ModelUsageTable(props: {
  rows?: QuotaDataItem[]
  loading: boolean
  error?: boolean
  onRetry?: () => void
}) {
  const { t } = useTranslation()
  const models = useMemo(() => {
    const groups = new Map<
      string,
      { requests: number; tokens: number; quota: number }
    >()
    for (const row of props.rows ?? []) {
      const name = row.model_name || t('Unknown model')
      const current = groups.get(name) ?? { requests: 0, tokens: 0, quota: 0 }
      current.requests += Number(row.count ?? 0)
      current.tokens += Number(row.token_used ?? 0)
      current.quota += Number(row.quota ?? 0)
      groups.set(name, current)
    }
    return [...groups.entries()]
      .map(([name, values]) => ({ name, ...values }))
      .sort((left, right) => right.quota - left.quota)
  }, [props.rows, t])
  let modelContent: ReactNode
  if (props.error) {
    modelContent = (
      <ErrorState
        className='min-h-48'
        title={t('Failed to load')}
        description={t('Please try again later.')}
        onRetry={props.onRetry}
      />
    )
  } else if (props.loading) {
    modelContent = (
      <div className='flex flex-col gap-2 p-4'>
        {['a', 'b', 'c'].map((key) => (
          <Skeleton key={key} className='h-10 w-full' />
        ))}
      </div>
    )
  } else if (models.length === 0) {
    modelContent = (
      <p className='text-muted-foreground px-4 py-10 text-center text-sm'>
        {t('No usage data for this time range.')}
      </p>
    )
  } else {
    modelContent = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Model')}</TableHead>
            <TableHead className='text-right'>{t('Requests')}</TableHead>
            <TableHead className='text-right'>{t('Tokens')}</TableHead>
            <TableHead className='text-right'>{t('Usage')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {models.map((model) => (
            <TableRow key={model.name}>
              <TableCell className='max-w-64 truncate font-medium'>
                {model.name}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatCompactNumber(model.requests)}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatCompactNumber(model.tokens)}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatQuota(model.quota)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    )
  }

  return (
    <section className='overflow-hidden rounded-lg border'>
      <div className='border-b px-4 py-3 sm:px-5'>
        <h2 className='text-sm font-semibold'>{t('Model usage')}</h2>
      </div>
      {modelContent}
    </section>
  )
}
