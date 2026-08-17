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
import type { DashboardUserFilters } from '../types'

export interface DashboardUserFilterValidation {
  valid: boolean
  timeRangeInvalid: boolean
  channelInvalid: boolean
}

export interface DashboardUserApiFilters {
  start_timestamp: number
  end_timestamp: number
  model_name?: string
  token_name?: string
  group?: string
  channel?: number
}

export function getTodayDashboardRange(): Pick<
  DashboardUserFilters,
  'startTimestamp' | 'endTimestamp'
> {
  const end = new Date()
  const start = new Date(end)
  start.setHours(0, 0, 0, 0)
  return {
    startTimestamp: Math.floor(start.getTime() / 1000),
    endTimestamp: Math.floor(end.getTime() / 1000),
  }
}

export function validateDashboardUserFilters(
  filters: DashboardUserFilters
): DashboardUserFilterValidation {
  const timeRangeInvalid =
    !Number.isSafeInteger(filters.startTimestamp) ||
    !Number.isSafeInteger(filters.endTimestamp) ||
    filters.startTimestamp <= 0 ||
    filters.endTimestamp <= 0 ||
    filters.endTimestamp < filters.startTimestamp
  const channel = filters.channel.trim()
  const channelNumber = Number(channel)
  const channelInvalid =
    channel !== '' &&
    (!/^\d+$/.test(channel) ||
      !Number.isSafeInteger(channelNumber) ||
      channelNumber <= 0)

  return {
    valid: !timeRangeInvalid && !channelInvalid,
    timeRangeInvalid,
    channelInvalid,
  }
}

export function normalizeDashboardUserFilters(
  filters: DashboardUserFilters,
  fallbackRange = getTodayDashboardRange()
): DashboardUserFilters {
  const validation = validateDashboardUserFilters(filters)
  return {
    startTimestamp: validation.timeRangeInvalid
      ? fallbackRange.startTimestamp
      : filters.startTimestamp,
    endTimestamp: validation.timeRangeInvalid
      ? fallbackRange.endTimestamp
      : filters.endTimestamp,
    model: filters.model.trim(),
    token: filters.token.trim(),
    group: filters.group.trim(),
    channel: validation.channelInvalid ? '' : filters.channel.trim(),
  }
}

export function buildDashboardUserApiFilters(
  filters: DashboardUserFilters
): DashboardUserApiFilters {
  const normalized = normalizeDashboardUserFilters(filters)
  const channel = Number(normalized.channel)
  return {
    start_timestamp: normalized.startTimestamp,
    end_timestamp: normalized.endTimestamp,
    model_name: normalized.model || undefined,
    token_name: normalized.token || undefined,
    group: normalized.group || undefined,
    channel: normalized.channel ? channel : undefined,
  }
}
