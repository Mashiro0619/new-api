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

import type { DashboardUserFilters } from '../../types'
import {
  buildDashboardUserApiFilters,
  normalizeDashboardUserFilters,
  validateDashboardUserFilters,
} from '../user-analytics'

const validFilters: DashboardUserFilters = {
  startTimestamp: 1_700_000_000,
  endTimestamp: 1_700_003_600,
  model: ' gpt-4.1 ',
  token: ' production ',
  group: ' default ',
  channel: ' 7 ',
}

describe('dashboard user analytics filters', () => {
  test('accepts an equal time boundary and a trimmed positive channel', () => {
    const validation = validateDashboardUserFilters({
      ...validFilters,
      endTimestamp: validFilters.startTimestamp,
    })

    expect(validation).toEqual({
      valid: true,
      timeRangeInvalid: false,
      channelInvalid: false,
    })
  })

  test('rejects a reversed range and a non-integer channel together', () => {
    const validation = validateDashboardUserFilters({
      ...validFilters,
      endTimestamp: validFilters.startTimestamp - 1,
      channel: '7.5',
    })

    expect(validation).toEqual({
      valid: false,
      timeRangeInvalid: true,
      channelInvalid: true,
    })
  })

  test('falls back from invalid timestamps and clears an invalid channel', () => {
    const fallbackRange = {
      startTimestamp: 1_800_000_000,
      endTimestamp: 1_800_003_600,
    }

    expect(
      normalizeDashboardUserFilters(
        {
          ...validFilters,
          startTimestamp: Number.NaN,
          channel: '-1',
        },
        fallbackRange
      )
    ).toEqual({
      ...fallbackRange,
      model: 'gpt-4.1',
      token: 'production',
      group: 'default',
      channel: '',
    })
  })

  test('builds one normalized filter contract for all analytics requests', () => {
    expect(buildDashboardUserApiFilters(validFilters)).toEqual({
      start_timestamp: validFilters.startTimestamp,
      end_timestamp: validFilters.endTimestamp,
      model_name: 'gpt-4.1',
      token_name: 'production',
      group: 'default',
      channel: 7,
    })
  })

  test('omits empty optional filters from analytics requests', () => {
    expect(
      buildDashboardUserApiFilters({
        ...validFilters,
        model: ' ',
        token: '',
        group: ' ',
        channel: '',
      })
    ).toEqual({
      start_timestamp: validFilters.startTimestamp,
      end_timestamp: validFilters.endTimestamp,
      model_name: undefined,
      token_name: undefined,
      group: undefined,
      channel: undefined,
    })
  })
})
