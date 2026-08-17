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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { getTokenTrend } from '../../api'
import type { TokenTrendData, TokenTrendFilters } from '../../types'
import { TokenTrendPanel } from '../token-trend-panel'

vi.mock('../../api', () => ({
  getTokenTrend: vi.fn(),
}))

const baseFilters: TokenTrendFilters = {
  startTimestamp: 1_700_000_000,
  endTimestamp: 1_700_003_600,
}

const emptyMetrics = {
  input_tokens: 0,
  output_tokens: 0,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  cache_hit_rate: null,
  tracked_requests: 0,
}

function renderPanel(filters: TokenTrendFilters = baseFilters) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <TokenTrendPanel scope={{ mode: 'self' }} filters={filters} />
    </QueryClientProvider>
  )
}

afterEach(() => {
  vi.clearAllMocks()
})

describe('TokenTrendPanel states', () => {
  test('shows an exact-request notice without fetching trend data', () => {
    renderPanel({ ...baseFilters, requestId: 'req_123' })

    expect(screen.getByText('Token trend not applicable')).toBeVisible()
    expect(getTokenTrend).not.toHaveBeenCalled()
  })

  test('explains when consume logging disables token trend data', async () => {
    vi.mocked(getTokenTrend).mockResolvedValue({
      available: false,
      reason: 'consume_logging_disabled',
      tracking_started_at: null,
      start_timestamp: baseFilters.startTimestamp,
      end_timestamp: baseFilters.endTimestamp,
      granularity: 'hour',
      totals: emptyMetrics,
      points: [],
    })

    renderPanel()

    expect(
      await screen.findByText(
        'Token trend is unavailable because consume logging is disabled.'
      )
    ).toBeVisible()
  })

  test('shows an empty state when tracking has no points in range', async () => {
    const emptyData: TokenTrendData = {
      available: true,
      reason: '',
      tracking_started_at: null,
      start_timestamp: baseFilters.startTimestamp,
      end_timestamp: baseFilters.endTimestamp,
      granularity: 'hour',
      totals: emptyMetrics,
      points: [],
    }
    vi.mocked(getTokenTrend).mockResolvedValue(emptyData)

    renderPanel()

    expect(await screen.findByText('No token trend data')).toBeVisible()
  })

  test('formats the backend percentage value without multiplying it twice', async () => {
    vi.mocked(getTokenTrend).mockResolvedValue({
      available: true,
      reason: '',
      tracking_started_at: baseFilters.startTimestamp,
      start_timestamp: baseFilters.startTimestamp,
      end_timestamp: baseFilters.endTimestamp,
      granularity: 'hour',
      totals: {
        ...emptyMetrics,
        input_tokens: 1_000,
        cache_read_tokens: 1_000,
        cache_hit_rate: 50,
        tracked_requests: 1,
      },
      points: [
        {
          timestamp: baseFilters.startTimestamp,
          ...emptyMetrics,
          input_tokens: 1_000,
          cache_read_tokens: 1_000,
          cache_hit_rate: 50,
          tracked_requests: 1,
        },
      ],
    })

    renderPanel()

    expect(await screen.findByText('50%')).toBeVisible()
    expect(screen.queryByText('5,000%')).not.toBeInTheDocument()
  })

  test('shows a recoverable error state when the request fails', async () => {
    vi.mocked(getTokenTrend).mockRejectedValue(new Error('network error'))

    renderPanel()

    expect(await screen.findByText('Failed to load token trend')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeVisible()
  })
})
