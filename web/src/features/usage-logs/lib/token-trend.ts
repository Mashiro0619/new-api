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
import type { TokenTrendFilters, TokenTrendScope } from '@/features/token-trend'

import { getDefaultTimeRange } from './utils'

export interface UsageLogTokenTrendSearch {
  startTime?: number
  endTime?: number
  model?: string
  token?: string
  group?: string
  channel?: string
  username?: string
  type?: string | readonly string[]
  requestId?: string
  upstreamRequestId?: string
}

export function buildUsageLogTokenTrendFilters(
  search: UsageLogTokenTrendSearch,
  scope: TokenTrendScope,
  defaultRange = getDefaultTimeRange()
): TokenTrendFilters {
  const startTime = search.startTime
    ? new Date(search.startTime)
    : defaultRange.start
  const endTime = search.endTime ? new Date(search.endTime) : defaultRange.end
  const logType =
    typeof search.type === 'string' ? search.type : search.type?.[0]

  return {
    startTimestamp: Math.floor(startTime.getTime() / 1000),
    endTimestamp: Math.floor(endTime.getTime() / 1000),
    model: search.model || undefined,
    token: search.token || undefined,
    group: search.group || undefined,
    ...(scope.mode === 'admin' && search.channel
      ? { channel: search.channel }
      : {}),
    type: logType,
    requestId: search.requestId || undefined,
    upstreamRequestId: search.upstreamRequestId || undefined,
  }
}

export function buildUsageLogTokenTrendScope(
  isAdminView: boolean,
  username?: string
): TokenTrendScope {
  if (!isAdminView) return { mode: 'self' }
  return { mode: 'admin', username: username || undefined }
}

export function shouldUseFixedUsageLogTable(logCategory: string): boolean {
  return logCategory !== 'common'
}
