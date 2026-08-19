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
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { UsageLogsTable } from '../usage-logs-table'

const routerMocks = vi.hoisted(() => ({
  navigate: vi.fn(),
}))

vi.mock('@tanstack/react-query', () => ({
  useQuery: () => ({
    data: { items: [], total: 0 },
    isFetching: false,
    isLoading: false,
  }),
}))

vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({
    useNavigate: () => routerMocks.navigate,
    useSearch: () => ({}),
  }),
}))

vi.mock('@/components/data-table', () => ({
  DataTablePage: (props: { toolbar?: ReactNode }) => <div>{props.toolbar}</div>,
  DataTableRow: () => null,
  useDataTable: () => ({ table: {} }),
}))

vi.mock('@/features/token-trend', () => ({
  TokenTrendPanel: () => <div data-slot='token-trend-panel' />,
}))

vi.mock('@/hooks', () => ({
  useMediaQuery: () => false,
}))

vi.mock('@/hooks/use-table-url-state', () => ({
  useTableUrlState: () => ({
    columnFilters: [],
    ensurePageInRange: vi.fn(),
    onColumnFiltersChange: vi.fn(),
    onPaginationChange: vi.fn(),
    pagination: { pageIndex: 0, pageSize: 100 },
  }),
}))

vi.mock('../../lib/columns', () => ({
  useColumnsByCategory: () => [],
}))

vi.mock('../common-logs-filter-bar', () => ({
  CommonLogsFilterBar: () => <div data-slot='common-logs-filter-bar' />,
}))

vi.mock('../task-logs-filter-bar', () => ({
  TaskLogsFilterBar: () => null,
}))

vi.mock('../usage-logs-mobile-card', () => ({
  UsageLogsMobileList: () => null,
}))

vi.mock('../usage-logs-provider', () => ({
  useLogsViewScope: () => ({ isAdminView: false }),
}))

describe('UsageLogsTable layout', () => {
  test('places common-log filters and token trend in equal ultra-wide columns', () => {
    const { container } = render(<UsageLogsTable logCategory='common' />)

    const toolbarGrid = container.querySelector(
      '[data-slot="usage-logs-toolbar-grid"]'
    )
    if (!toolbarGrid) {
      throw new Error('Expected the common logs toolbar grid')
    }

    expect(toolbarGrid).toHaveClass('grid', 'min-[1800px]:grid-cols-2')
    expect(toolbarGrid.children).toHaveLength(2)
    expect(
      toolbarGrid.querySelector('[data-slot="common-logs-filter-bar"]')
    ).toBeVisible()
    expect(
      toolbarGrid.querySelector('[data-slot="token-trend-panel"]')
    ).toBeVisible()
  })
})
