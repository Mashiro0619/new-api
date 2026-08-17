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
import { describe, expect, test } from 'vitest'

import {
  buildUsageLogTokenTrendFilters,
  buildUsageLogTokenTrendScope,
  shouldUseFixedUsageLogTable,
} from '../token-trend'

describe('usage log token trend adapter', () => {
  test('converts applied URL timestamps and filters to the trend contract', () => {
    expect(
      buildUsageLogTokenTrendFilters(
        {
          startTime: 1_700_000_000_999,
          endTime: 1_700_003_600_999,
          model: 'gpt-4.1',
          token: 'production',
          group: 'default',
          channel: '7',
          type: ['2'],
          requestId: 'req_123',
          upstreamRequestId: 'upstream_123',
        },
        { mode: 'admin' }
      )
    ).toEqual({
      startTimestamp: 1_700_000_000,
      endTimestamp: 1_700_003_600,
      model: 'gpt-4.1',
      token: 'production',
      group: 'default',
      channel: '7',
      type: '2',
      requestId: 'req_123',
      upstreamRequestId: 'upstream_123',
    })
  })

  test('uses the same default range as the log table when URL times are absent', () => {
    const defaultRange = {
      start: new Date(1_700_000_000_000),
      end: new Date(1_700_003_600_000),
    }

    expect(
      buildUsageLogTokenTrendFilters({}, { mode: 'self' }, defaultRange)
    ).toMatchObject({
      startTimestamp: 1_700_000_000,
      endTimestamp: 1_700_003_600,
    })
  })

  test('drops a stale admin channel filter from the self trend scope', () => {
    const defaultRange = {
      start: new Date(1_700_000_000_000),
      end: new Date(1_700_003_600_000),
    }

    expect(
      buildUsageLogTokenTrendFilters(
        { channel: '7' },
        { mode: 'self' },
        defaultRange
      )
    ).not.toHaveProperty('channel')
  })

  test('maps all-user and self views to separate API scopes', () => {
    expect(buildUsageLogTokenTrendScope(true, 'alice')).toEqual({
      mode: 'admin',
      username: 'alice',
    })
    expect(buildUsageLogTokenTrendScope(false, 'alice')).toEqual({
      mode: 'self',
    })
  })

  test('lets common logs scroll as a page while task logs keep a fixed table', () => {
    expect(shouldUseFixedUsageLogTable('common')).toBe(false)
    expect(shouldUseFixedUsageLogTable('drawing')).toBe(true)
    expect(shouldUseFixedUsageLogTable('task')).toBe(true)
  })
})
