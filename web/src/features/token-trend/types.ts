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
export type TokenTrendGranularity = 'hour' | 'day'

export type TokenTrendScope =
  | { mode: 'self' }
  | {
      mode: 'admin'
      userId?: number
      username?: string
    }

export interface TokenTrendFilters {
  startTimestamp: number
  endTimestamp: number
  model?: string
  token?: string
  group?: string
  channel?: string | number
  type?: string | number
  requestId?: string
  upstreamRequestId?: string
}

export interface TokenTrendMetrics {
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  cache_hit_rate: number | null
  tracked_requests: number
}

export interface TokenTrendPoint extends TokenTrendMetrics {
  timestamp: number
}

export interface TokenTrendData {
  available: boolean
  reason: string
  tracking_started_at: number | null
  start_timestamp: number
  end_timestamp: number
  granularity: TokenTrendGranularity
  totals: TokenTrendMetrics
  points: TokenTrendPoint[]
}

export interface TokenTrendResponse {
  success: boolean
  message?: string
  data?: TokenTrendData
}

export interface TokenTrendPanelProps {
  scope: TokenTrendScope
  filters: TokenTrendFilters
  className?: string
}
