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
import { ArrowLeft01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { GroupBadge } from '@/components/group-badge'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { TokenTrendPanel } from '@/features/token-trend'
import {
  USER_ROLES,
  USER_STATUSES,
  USER_STATUS,
} from '@/features/users/constants'
import { formatCompactNumber, formatQuota } from '@/lib/format'

import {
  getDashboardUser,
  getDashboardUserLogStats,
  getUserQuotaDates,
} from '../../api'
import {
  buildDashboardUserApiFilters,
  getTodayDashboardRange,
  normalizeDashboardUserFilters,
} from '../../lib/user-analytics'
import type { DashboardUserFilters } from '../../types'
import { AnalyticsFilters } from './analytics-filters'
import { ModelUsageTable, UserStatStrip } from './user-detail-panels'
import { UserRequestTable } from './user-request-table'

const route = getRouteApi('/_authenticated/dashboard/users/$userId')

export function UserAnalyticsDetail() {
  const { t } = useTranslation()
  const globalNavigate = useNavigate()
  const params = route.useParams()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const userId = Number(params.userId)
  const fallbackRange = useMemo(getTodayDashboardRange, [])
  const filters = normalizeDashboardUserFilters(
    {
      startTimestamp: search.startTimestamp ?? fallbackRange.startTimestamp,
      endTimestamp: search.endTimestamp ?? fallbackRange.endTimestamp,
      model: search.model ?? '',
      token: search.token ?? '',
      group: search.group ?? '',
      channel: search.channel ?? '',
    },
    fallbackRange
  )
  const apiFilters = buildDashboardUserApiFilters(filters)
  const commonParams = {
    ...apiFilters,
    user_id: userId,
    type: 2,
  }

  const userQuery = useQuery({
    queryKey: ['dashboard', 'analytics-user', userId],
    queryFn: async () => {
      const result = await getDashboardUser(userId)
      if (!result.success || !result.data) {
        throw new Error(result.message || 'user')
      }
      return result.data
    },
  })
  const statsQuery = useQuery({
    queryKey: ['dashboard', 'analytics-user-stats', commonParams],
    queryFn: async () => {
      const result = await getDashboardUserLogStats(commonParams)
      if (!result.success) throw new Error(result.message || 'stats')
      return result.data
    },
  })
  const quotaQuery = useQuery({
    queryKey: ['dashboard', 'analytics-user-quota', commonParams],
    queryFn: async () => {
      const result = await getUserQuotaDates(commonParams, true)
      if (!result.success) throw new Error('quota')
      return result.data ?? []
    },
  })

  const applyFilters = (next: DashboardUserFilters) => {
    void navigate({
      search: (previous) => ({
        ...previous,
        logPage: 1,
        startTimestamp: next.startTimestamp,
        endTimestamp: next.endTimestamp,
        model: next.model || undefined,
        token: next.token || undefined,
        group: next.group || undefined,
        channel: next.channel || undefined,
      }),
    })
  }
  const user = userQuery.data
  const role = user && USER_ROLES[user.role as keyof typeof USER_ROLES]
  let status
  if (user) {
    status =
      user.DeletedAt != null
        ? USER_STATUSES[USER_STATUS.DELETED]
        : USER_STATUSES[user.status as keyof typeof USER_STATUSES]
  }
  let userSummary: ReactNode
  if (userQuery.isLoading) {
    userSummary = (
      <div className='flex items-center gap-3'>
        <Skeleton className='size-11 rounded-full' />
        <div className='flex flex-col gap-2'>
          <Skeleton className='h-5 w-36' />
          <Skeleton className='h-4 w-52' />
        </div>
      </div>
    )
  } else if (userQuery.isError || !user) {
    userSummary = (
      <div className='flex flex-wrap items-center gap-3'>
        <p className='text-destructive text-sm'>
          {t('Failed to load user details.')}
        </p>
        <Button
          variant='outline'
          size='sm'
          onClick={() => void userQuery.refetch()}
        >
          {t('Retry')}
        </Button>
      </div>
    )
  } else {
    userSummary = (
      <>
        <div className='flex min-w-0 items-center gap-3'>
          <Avatar className='size-11'>
            <AvatarFallback>
              {(user.display_name || user.username).slice(0, 2)}
            </AvatarFallback>
          </Avatar>
          <div className='min-w-0'>
            <h2 className='truncate text-lg font-semibold'>
              {user.display_name || user.username}
            </h2>
            <p className='text-muted-foreground truncate text-sm'>
              @{user.username} / #{user.id}
            </p>
          </div>
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          {role && <StatusBadge label={t(role.labelKey)} copyable={false} />}
          <GroupBadge group={user.group} />
          {status && (
            <StatusBadge
              label={t(status.labelKey)}
              variant={status.variant}
              copyable={false}
            />
          )}
          <span className='text-muted-foreground text-xs'>
            {t('{{count}} lifetime requests', {
              count: formatCompactNumber(user.request_count),
            })}
          </span>
          <span className='text-muted-foreground text-xs'>
            {formatQuota(user.used_quota)}
          </span>
        </div>
      </>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('User usage details')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='sm'
          onClick={() =>
            void globalNavigate({
              to: '/dashboard/$section',
              params: { section: 'users' },
              search: {
                page: search.listPage,
                filter: search.listFilter,
                status: search.listStatus,
                role: search.listRole,
                group: search.listGroup,
                sortBy: search.listSortBy,
                sortOrder: search.listSortOrder,
                startTimestamp: filters.startTimestamp,
                endTimestamp: filters.endTimestamp,
                model: filters.model || undefined,
                token: filters.token || undefined,
                usageGroup: filters.group || undefined,
                channel: filters.channel || undefined,
              },
            })
          }
        >
          <HugeiconsIcon icon={ArrowLeft01Icon} data-icon='inline-start' />
          {t('Back to users')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <section className='flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-center sm:justify-between'>
            {userSummary}
          </section>

          <div className='overflow-hidden rounded-lg border'>
            <AnalyticsFilters
              key={`${filters.startTimestamp}-${filters.endTimestamp}-${filters.model}-${filters.token}-${filters.group}-${filters.channel}`}
              value={filters}
              onApply={applyFilters}
            />
          </div>
          <TokenTrendPanel
            scope={{ mode: 'admin', userId }}
            filters={{
              startTimestamp: filters.startTimestamp,
              endTimestamp: filters.endTimestamp,
              model: filters.model,
              token: filters.token,
              group: filters.group,
              channel: filters.channel,
              type: 2,
            }}
          />

          <UserStatStrip
            stats={statsQuery.data}
            rows={quotaQuery.data}
            loading={statsQuery.isLoading || quotaQuery.isLoading}
            error={statsQuery.isError || quotaQuery.isError}
            onRetry={() => {
              void statsQuery.refetch()
              void quotaQuery.refetch()
            }}
          />
          <div className='flex flex-col gap-4'>
            <ModelUsageTable
              rows={quotaQuery.data}
              loading={quotaQuery.isLoading}
              error={quotaQuery.isError}
              onRetry={() => void quotaQuery.refetch()}
            />
            <UserRequestTable
              userId={userId}
              filters={filters}
              page={Math.max(search.logPage ?? 1, 1)}
              onPageChange={(logPage) =>
                void navigate({
                  search: (previous) => ({ ...previous, logPage }),
                })
              }
            />
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
