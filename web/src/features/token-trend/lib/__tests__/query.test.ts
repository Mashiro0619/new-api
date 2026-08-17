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
  buildTokenTrendQuery,
  getTokenTrendApplicability,
  getTokenTrendGranularity,
} from '../query'

describe('token trend query', () => {
  test('uses hourly buckets for ranges up to 48 hours', () => {
    expect(getTokenTrendGranularity(1_000, 1_000 + 48 * 60 * 60)).toBe('hour')
  })

  test('uses daily buckets for ranges longer than 48 hours', () => {
    expect(getTokenTrendGranularity(1_000, 1_000 + 48 * 60 * 60 + 1)).toBe(
      'day'
    )
  })

  test('builds an admin user query with applied analytics filters', () => {
    const params = buildTokenTrendQuery(
      { mode: 'admin', userId: 42, username: 'ignored' },
      {
        startTimestamp: 100,
        endTimestamp: 200,
        model: 'gpt-4.1',
        token: 'production',
        group: 'default',
        channel: '7',
      },
      -480
    )

    expect(Object.fromEntries(params)).toEqual({
      start_timestamp: '100',
      end_timestamp: '200',
      granularity: 'hour',
      timezone_offset: '-480',
      model_name: 'gpt-4.1',
      token_name: 'production',
      group: 'default',
      channel: '7',
      user_id: '42',
    })
  })

  test('never sends identity filters to the self endpoint', () => {
    const params = buildTokenTrendQuery(
      { mode: 'self' },
      { startTimestamp: 100, endTimestamp: 200 },
      300
    )

    expect(params.get('timezone_offset')).toBe('300')
    expect(params.has('user_id')).toBe(false)
    expect(params.has('username')).toBe(false)
  })
})

describe('token trend applicability', () => {
  test('skips exact request filters', () => {
    expect(
      getTokenTrendApplicability({
        startTimestamp: 100,
        endTimestamp: 200,
        requestId: 'req_123',
      })
    ).toEqual({ kind: 'exact-request' })
  })

  test('accepts all and consume log types', () => {
    expect(
      getTokenTrendApplicability({
        startTimestamp: 100,
        endTimestamp: 200,
        type: '0',
      })
    ).toEqual({ kind: 'applicable' })
    expect(
      getTokenTrendApplicability({
        startTimestamp: 100,
        endTimestamp: 200,
        type: 2,
      })
    ).toEqual({ kind: 'applicable' })
  })

  test('skips log types without token metrics', () => {
    expect(
      getTokenTrendApplicability({
        startTimestamp: 100,
        endTimestamp: 200,
        type: '5',
      })
    ).toEqual({ kind: 'unsupported-type' })
  })
})
