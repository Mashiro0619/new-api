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
import type {
  TokenTrendFilters,
  TokenTrendGranularity,
  TokenTrendScope,
} from '../types'

const GRANULARITY_SWITCH_SECONDS = 48 * 60 * 60
const ALL_LOG_TYPES = new Set(['', '0', '2'])

export type TokenTrendApplicability =
  | { kind: 'applicable' }
  | { kind: 'exact-request' }
  | { kind: 'unsupported-type' }

function appendTrimmed(
  params: URLSearchParams,
  key: string,
  value: string | number | undefined
): void {
  if (value === undefined) return
  const normalized = String(value).trim()
  if (normalized !== '') params.set(key, normalized)
}

export function getTokenTrendGranularity(
  startTimestamp: number,
  endTimestamp: number
): TokenTrendGranularity {
  return endTimestamp - startTimestamp <= GRANULARITY_SWITCH_SECONDS
    ? 'hour'
    : 'day'
}

export function getTokenTrendApplicability(
  filters: TokenTrendFilters
): TokenTrendApplicability {
  if (filters.requestId?.trim() || filters.upstreamRequestId?.trim()) {
    return { kind: 'exact-request' }
  }

  const logType = String(filters.type ?? '').trim()
  if (!ALL_LOG_TYPES.has(logType)) return { kind: 'unsupported-type' }

  return { kind: 'applicable' }
}

export function buildTokenTrendQuery(
  scope: TokenTrendScope,
  filters: TokenTrendFilters,
  timezoneOffset: number
): URLSearchParams {
  const params = new URLSearchParams({
    start_timestamp: String(filters.startTimestamp),
    end_timestamp: String(filters.endTimestamp),
    granularity: getTokenTrendGranularity(
      filters.startTimestamp,
      filters.endTimestamp
    ),
    timezone_offset: String(timezoneOffset),
  })

  appendTrimmed(params, 'model_name', filters.model)
  appendTrimmed(params, 'token_name', filters.token)
  appendTrimmed(params, 'group', filters.group)
  appendTrimmed(params, 'channel', filters.channel)

  if (scope.mode === 'admin') {
    if (scope.userId && scope.userId > 0) {
      params.set('user_id', String(scope.userId))
    } else {
      appendTrimmed(params, 'username', scope.username)
    }
  }

  return params
}
