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
import { api } from '@/lib/api'

import { buildTokenTrendQuery } from './lib/query'
import type {
  TokenTrendData,
  TokenTrendFilters,
  TokenTrendResponse,
  TokenTrendScope,
} from './types'

export async function getTokenTrend(
  scope: TokenTrendScope,
  filters: TokenTrendFilters,
  timezoneOffset: number
): Promise<TokenTrendData> {
  const endpoint =
    scope.mode === 'self'
      ? '/api/data/token-trend/self'
      : '/api/data/token-trend'
  const query = buildTokenTrendQuery(scope, filters, timezoneOffset)
  const response = await api.get<TokenTrendResponse>(
    `${endpoint}?${query.toString()}`
  )

  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to load token trend')
  }

  return response.data.data
}
