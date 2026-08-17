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
import type { PropsWithChildren } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { ModelUsageTable, UserStatStrip } from '../user-detail-panels'

vi.mock('@/components/page-transition', () => ({
  FadeIn: (props: PropsWithChildren) => props.children,
}))

describe('user analytics detail error states', () => {
  test('statistics failure replaces zero values with a retry action', async () => {
    const retry = vi.fn()
    const user = userEvent.setup()

    render(
      <UserStatStrip
        stats={{ quota: 0, rpm: 0, tpm: 0 }}
        rows={[]}
        loading={false}
        error
        onRetry={retry}
      />
    )

    expect(screen.getByText('Failed to load')).toBeVisible()
    expect(screen.queryByText('RPM')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(retry).toHaveBeenCalledOnce()
  })

  test('model usage failure is not presented as an empty result', async () => {
    const retry = vi.fn()
    const user = userEvent.setup()

    render(<ModelUsageTable rows={[]} loading={false} error onRetry={retry} />)

    expect(screen.getByText('Failed to load')).toBeVisible()
    expect(
      screen.queryByText('No usage data for this time range.')
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(retry).toHaveBeenCalledOnce()
  })
})
