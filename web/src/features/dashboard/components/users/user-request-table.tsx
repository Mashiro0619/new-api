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
import { ArrowLeft01Icon, ArrowRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatCompactNumber, formatQuota, formatTimestamp } from '@/lib/format'

import { getDashboardUserLogs } from '../../api'
import { buildDashboardUserApiFilters } from '../../lib/user-analytics'
import type { DashboardUserFilters } from '../../types'

const PAGE_SIZE = 20

export function UserRequestTable(props: {
  userId: number
  filters: DashboardUserFilters
  page: number
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const apiFilters = buildDashboardUserApiFilters(props.filters)
  const query = useQuery({
    queryKey: [
      'dashboard',
      'user-requests',
      props.userId,
      props.filters,
      props.page,
    ],
    queryFn: async () => {
      const result = await getDashboardUserLogs({
        ...apiFilters,
        user_id: props.userId,
        p: props.page,
        page_size: PAGE_SIZE,
        type: 2,
      })
      if (!result.success) throw new Error(result.message || 'logs')
      return (
        result.data ?? { items: [], total: 0, page: 1, page_size: PAGE_SIZE }
      )
    },
    placeholderData: (previous) => previous,
  })
  const total = query.data?.total ?? 0
  const pageCount = Math.max(Math.ceil(total / PAGE_SIZE), 1)
  let requestContent: ReactNode
  if (query.isLoading) {
    requestContent = (
      <div className='flex flex-col gap-2 p-4'>
        {['a', 'b', 'c', 'd', 'e'].map((key) => (
          <Skeleton key={key} className='h-10 w-full' />
        ))}
      </div>
    )
  } else if (query.isError) {
    requestContent = (
      <ErrorState
        className='min-h-48'
        title={t('Failed to load request records.')}
        description={t('Please try again later.')}
        onRetry={() => void query.refetch()}
      />
    )
  } else if (total === 0) {
    requestContent = (
      <p className='text-muted-foreground px-4 py-10 text-center text-sm'>
        {t('No request records for this time range.')}
      </p>
    )
  } else {
    requestContent = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Time')}</TableHead>
            <TableHead>{t('Model')}</TableHead>
            <TableHead>{t('Token')}</TableHead>
            <TableHead className='text-right'>{t('Input')}</TableHead>
            <TableHead className='text-right'>{t('Output')}</TableHead>
            <TableHead className='text-right'>{t('Usage')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {(query.data?.items ?? []).map((log) => (
            <TableRow key={log.id}>
              <TableCell>{formatTimestamp(log.created_at)}</TableCell>
              <TableCell className='max-w-56 truncate'>
                {log.model_name || '-'}
              </TableCell>
              <TableCell className='max-w-40 truncate'>
                {log.token_name || '-'}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatCompactNumber(log.prompt_tokens ?? 0)}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatCompactNumber(log.completion_tokens ?? 0)}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatQuota(log.quota ?? 0)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    )
  }

  return (
    <section className='overflow-hidden rounded-lg border lg:col-span-2'>
      <div className='flex items-center justify-between gap-3 border-b px-4 py-3 sm:px-5'>
        <div>
          <h2 className='text-sm font-semibold'>{t('Request records')}</h2>
          <p className='text-muted-foreground text-xs'>
            {t('{{count}} records', { count: total })}
          </p>
        </div>
        <div className='flex gap-1'>
          <Button
            size='icon'
            variant='ghost'
            disabled={props.page <= 1}
            aria-label={t('Previous page')}
            onClick={() => props.onPageChange(props.page - 1)}
          >
            <HugeiconsIcon icon={ArrowLeft01Icon} />
          </Button>
          <Button
            size='icon'
            variant='ghost'
            disabled={props.page >= pageCount}
            aria-label={t('Next page')}
            onClick={() => props.onPageChange(props.page + 1)}
          >
            <HugeiconsIcon icon={ArrowRight01Icon} />
          </Button>
        </div>
      </div>
      {requestContent}
    </section>
  )
}
