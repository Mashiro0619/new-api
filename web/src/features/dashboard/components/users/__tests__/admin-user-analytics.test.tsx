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
import { render } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import { AdminUserAnalytics } from '../admin-user-analytics'

vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({
    useNavigate: () => vi.fn(),
    useSearch: () => ({}),
  }),
}))

vi.mock('@/features/token-trend', () => ({
  TokenTrendPanel: () => <div data-slot='token-trend-panel' />,
}))

vi.mock('../analytics-filters', () => ({
  AnalyticsFilters: () => <div data-slot='analytics-filters' />,
}))

vi.mock('../analytics-user-table', () => ({
  AnalyticsUserTable: () => <div data-slot='analytics-user-table' />,
}))

describe('AdminUserAnalytics layout', () => {
  test('places filters and token trend in equal ultra-wide columns', () => {
    const { container } = render(<AdminUserAnalytics />)

    const toolbarGrid = container.querySelector(
      '[data-slot="admin-user-analytics-toolbar-grid"]'
    )
    if (!toolbarGrid) {
      throw new Error('Expected the user analytics toolbar grid')
    }

    expect(toolbarGrid).toHaveClass('grid', 'min-[1800px]:grid-cols-2')
    expect(toolbarGrid.children).toHaveLength(2)
    expect(
      toolbarGrid.querySelector('[data-slot="analytics-filters"]')
    ).toBeVisible()
    expect(
      toolbarGrid.querySelector('[data-slot="token-trend-panel"]')
    ).toBeVisible()
  })
})
