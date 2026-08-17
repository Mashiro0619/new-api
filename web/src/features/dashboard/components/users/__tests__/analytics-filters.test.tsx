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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import { AnalyticsFilters } from '../analytics-filters'

describe('analytics filters validation', () => {
  test('shows field errors and does not apply an invalid range or channel', async () => {
    const onApply = vi.fn()
    const user = userEvent.setup()

    render(
      <AnalyticsFilters
        value={{
          startTimestamp: 1_700_003_600,
          endTimestamp: 1_700_000_000,
          model: '',
          token: '',
          group: '',
          channel: '-1',
        }}
        onApply={onApply}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Apply filters' }))

    expect(screen.getByText('Select a valid time range.')).toBeVisible()
    expect(
      screen.getByText('Channel must be a positive integer.')
    ).toBeVisible()
    expect(screen.getByLabelText('Start time')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    expect(screen.getByLabelText('End time')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    expect(screen.getByLabelText('Channel')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    expect(onApply).not.toHaveBeenCalled()
  })
})
