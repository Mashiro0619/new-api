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
import { getRouteApi } from '@tanstack/react-router'
import { useMemo } from 'react'

import { TokenTrendPanel } from '@/features/token-trend'

import {
  getTodayDashboardRange,
  normalizeDashboardUserFilters,
} from '../../lib/user-analytics'
import type { DashboardUserFilters } from '../../types'
import { AnalyticsFilters } from './analytics-filters'
import { AnalyticsUserTable } from './analytics-user-table'

const route = getRouteApi('/_authenticated/dashboard/$section')

export function AdminUserAnalytics() {
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const fallbackRange = useMemo(getTodayDashboardRange, [])
  const filters = normalizeDashboardUserFilters(
    {
      startTimestamp: search.startTimestamp ?? fallbackRange.startTimestamp,
      endTimestamp: search.endTimestamp ?? fallbackRange.endTimestamp,
      model: search.model ?? '',
      token: search.token ?? '',
      group: search.usageGroup ?? '',
      channel: search.channel ?? '',
    },
    fallbackRange
  )

  const applyFilters = (next: DashboardUserFilters) => {
    void navigate({
      search: (previous) => ({
        ...previous,
        startTimestamp: next.startTimestamp,
        endTimestamp: next.endTimestamp,
        model: next.model || undefined,
        token: next.token || undefined,
        usageGroup: next.group || undefined,
        channel: next.channel || undefined,
      }),
    })
  }

  return (
    <div className='flex flex-col gap-4'>
      <div
        className='grid gap-4 min-[1800px]:grid-cols-2'
        data-slot='admin-user-analytics-toolbar-grid'
      >
        <div className='min-w-0 overflow-hidden rounded-lg border'>
          <AnalyticsFilters
            key={`${filters.startTimestamp}-${filters.endTimestamp}-${filters.model}-${filters.token}-${filters.group}-${filters.channel}`}
            value={filters}
            onApply={applyFilters}
          />
        </div>
        <TokenTrendPanel
          className='min-w-0'
          scope={{ mode: 'admin' }}
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
      </div>
      <AnalyticsUserTable filters={filters} />
    </div>
  )
}
