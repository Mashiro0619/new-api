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
import { FilterIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatTimestampForInput, parseTimestampFromInput } from '@/lib/format'

import {
  normalizeDashboardUserFilters,
  validateDashboardUserFilters,
} from '../../lib/user-analytics'
import type { DashboardUserFilters } from '../../types'

interface AnalyticsFiltersProps {
  value: DashboardUserFilters
  onApply: (filters: DashboardUserFilters) => void
  compact?: boolean
}

export function AnalyticsFilters(props: AnalyticsFiltersProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState(props.value)
  const [submitted, setSubmitted] = useState(false)
  const validation = validateDashboardUserFilters(draft)

  const setField = (field: keyof DashboardUserFilters, value: string) => {
    setDraft((current) => ({ ...current, [field]: value }))
  }

  return (
    <form
      className='flex flex-col gap-3 border-b px-4 py-3 sm:px-5'
      onSubmit={(event) => {
        event.preventDefault()
        setSubmitted(true)
        if (!validation.valid) return
        props.onApply(normalizeDashboardUserFilters(draft))
      }}
    >
      <div className='grid gap-2 sm:grid-cols-2 lg:grid-cols-4'>
        <label className='flex min-w-0 flex-col gap-1 text-xs font-medium'>
          <span className='text-muted-foreground'>{t('Start time')}</span>
          <Input
            type='datetime-local'
            aria-invalid={submitted && validation.timeRangeInvalid}
            aria-describedby={
              submitted && validation.timeRangeInvalid
                ? 'analytics-time-range-error'
                : undefined
            }
            value={formatTimestampForInput(draft.startTimestamp)}
            onChange={(event) =>
              setDraft((current) => ({
                ...current,
                startTimestamp: parseTimestampFromInput(event.target.value),
              }))
            }
          />
        </label>
        <label className='flex min-w-0 flex-col gap-1 text-xs font-medium'>
          <span className='text-muted-foreground'>{t('End time')}</span>
          <Input
            type='datetime-local'
            aria-invalid={submitted && validation.timeRangeInvalid}
            aria-describedby={
              submitted && validation.timeRangeInvalid
                ? 'analytics-time-range-error'
                : undefined
            }
            value={formatTimestampForInput(draft.endTimestamp)}
            onChange={(event) =>
              setDraft((current) => ({
                ...current,
                endTimestamp: parseTimestampFromInput(event.target.value),
              }))
            }
          />
        </label>
        {!props.compact && (
          <>
            <label className='flex min-w-0 flex-col gap-1 text-xs font-medium'>
              <span className='text-muted-foreground'>{t('Model')}</span>
              <Input
                value={draft.model}
                placeholder={t('All models')}
                onChange={(event) => setField('model', event.target.value)}
              />
            </label>
            <label className='flex min-w-0 flex-col gap-1 text-xs font-medium'>
              <span className='text-muted-foreground'>{t('Token')}</span>
              <Input
                value={draft.token}
                placeholder={t('All tokens')}
                onChange={(event) => setField('token', event.target.value)}
              />
            </label>
            <label className='flex min-w-0 flex-col gap-1 text-xs font-medium'>
              <span className='text-muted-foreground'>{t('Group')}</span>
              <Input
                value={draft.group}
                placeholder={t('All groups')}
                onChange={(event) => setField('group', event.target.value)}
              />
            </label>
            <label className='flex min-w-0 flex-col gap-1 text-xs font-medium'>
              <span className='text-muted-foreground'>{t('Channel')}</span>
              <Input
                inputMode='numeric'
                aria-invalid={submitted && validation.channelInvalid}
                aria-describedby={
                  submitted && validation.channelInvalid
                    ? 'analytics-channel-error'
                    : undefined
                }
                value={draft.channel}
                placeholder={t('All channels')}
                onChange={(event) => setField('channel', event.target.value)}
              />
            </label>
          </>
        )}
      </div>
      {submitted && !validation.valid && (
        <div
          className='text-destructive flex flex-col gap-1 text-xs'
          role='alert'
        >
          {validation.timeRangeInvalid && (
            <p id='analytics-time-range-error'>
              {t('Select a valid time range.')}
            </p>
          )}
          {validation.channelInvalid && (
            <p id='analytics-channel-error'>
              {t('Channel must be a positive integer.')}
            </p>
          )}
        </div>
      )}
      <div className='flex justify-end'>
        <Button type='submit' size='sm'>
          <HugeiconsIcon icon={FilterIcon} data-icon='inline-start' />
          {t('Apply filters')}
        </Button>
      </div>
    </form>
  )
}
