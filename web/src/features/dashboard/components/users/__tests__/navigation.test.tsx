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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { PropsWithChildren, ReactNode } from 'react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { AnalyticsUserTable } from '../analytics-user-table'
import { UserAnalyticsDetail } from '../user-analytics-detail'

const routerMocks = vi.hoisted(() => ({
  sectionSearch: {
    page: 3,
    filter: 'alice',
    status: '1',
    role: '10',
    group: 'list-group',
    sortBy: 'username' as const,
    sortOrder: 'asc' as const,
  },
  detailSearch: {
    logPage: 2,
    startTimestamp: 1_700_000_000,
    endTimestamp: 1_700_003_600,
    model: 'gpt-4.1',
    token: 'production',
    group: 'analytics-group',
    channel: '7',
    listPage: 3,
    listFilter: 'alice',
    listStatus: '1',
    listRole: '10',
    listGroup: 'list-group',
    listSortBy: 'username' as const,
    listSortOrder: 'asc' as const,
  },
  sectionNavigate: vi.fn(),
  detailNavigate: vi.fn(),
  globalNavigate: vi.fn(),
  linkProps: vi.fn(),
}))

const userApiMocks = vi.hoisted(() => ({
  getGroups: vi.fn(),
  getUsers: vi.fn(),
  searchUsers: vi.fn(),
}))

const dashboardApiMocks = vi.hoisted(() => ({
  getDashboardUser: vi.fn(),
  getDashboardUserLogStats: vi.fn(),
  getUserQuotaDates: vi.fn(),
}))

vi.mock('@tanstack/react-router', async () => {
  const { createElement, forwardRef } =
    await vi.importActual<typeof import('react')>('react')
  type MockLinkProps = {
    children?: ReactNode
    to?: string
    params?: unknown
    search?: unknown
    'aria-label'?: string
  }
  const Link = forwardRef<HTMLAnchorElement, MockLinkProps>((props, ref) => {
    routerMocks.linkProps(props)
    return createElement(
      'a',
      { ref, href: '#detail', 'aria-label': props['aria-label'] },
      props.children
    )
  })

  return {
    getRouteApi: (routeId: string) => {
      if (routeId === '/_authenticated/dashboard/$section') {
        return {
          useSearch: () => routerMocks.sectionSearch,
          useNavigate: () => routerMocks.sectionNavigate,
        }
      }
      return {
        useParams: () => ({ userId: '42' }),
        useSearch: () => routerMocks.detailSearch,
        useNavigate: () => routerMocks.detailNavigate,
      }
    },
    Link,
    useNavigate: () => routerMocks.globalNavigate,
  }
})

vi.mock('@/features/users/api', () => userApiMocks)
vi.mock('@/features/dashboard/api', () => dashboardApiMocks)
vi.mock('@/features/token-trend', () => ({ TokenTrendPanel: () => null }))
vi.mock('@/components/page-transition', () => ({
  FadeIn: (props: PropsWithChildren) => props.children,
}))
vi.mock('../analytics-filters', () => ({ AnalyticsFilters: () => null }))
vi.mock('../user-detail-panels', () => ({
  ModelUsageTable: () => null,
  UserStatStrip: () => null,
}))
vi.mock('../user-request-table', () => ({ UserRequestTable: () => null }))
vi.mock('@/components/ui/tooltip', () => ({
  Tooltip: (props: PropsWithChildren) => props.children,
  TooltipContent: (props: PropsWithChildren) => props.children,
  TooltipTrigger: (props: PropsWithChildren<{ render?: ReactNode }>) =>
    props.render ?? props.children,
}))

const analyticsFilters = {
  startTimestamp: 1_700_000_000,
  endTimestamp: 1_700_003_600,
  model: 'gpt-4.1',
  token: 'production',
  group: 'analytics-group',
  channel: '7',
}

const analyticsUser = {
  id: 42,
  username: 'alice',
  display_name: 'Alice',
  email: 'alice@example.com',
  quota: 1_000,
  used_quota: 500,
  request_count: 12,
  group: 'list-group',
  status: 1,
  role: 10,
}

function renderWithQueryClient(children: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  userApiMocks.getGroups.mockResolvedValue({ success: true, data: [] })
  userApiMocks.getUsers.mockResolvedValue({
    success: true,
    data: { items: [analyticsUser], total: 1, page: 3, page_size: 20 },
  })
  userApiMocks.searchUsers.mockResolvedValue({
    success: true,
    data: { items: [analyticsUser], total: 1, page: 3, page_size: 20 },
  })
  dashboardApiMocks.getDashboardUser.mockResolvedValue({
    success: true,
    data: analyticsUser,
  })
  dashboardApiMocks.getDashboardUserLogStats.mockResolvedValue({
    success: true,
    data: { quota: 0, rpm: 0, tpm: 0 },
  })
  dashboardApiMocks.getUserQuotaDates.mockResolvedValue({
    success: true,
    data: [],
  })
})

describe('user analytics navigation state', () => {
  test('list drill-down carries both list state and analytics filters', async () => {
    renderWithQueryClient(<AnalyticsUserTable filters={analyticsFilters} />)

    expect(await screen.findByText('alice')).toBeVisible()
    expect(screen.getByRole('link', { name: 'View usage' })).toBeVisible()
    await waitFor(() => expect(routerMocks.linkProps).toHaveBeenCalled())

    const detailLink = routerMocks.linkProps.mock.calls
      .map(([props]) => props)
      .find((props) => props.to === '/dashboard/users/$userId')
    expect(detailLink).toMatchObject({
      params: { userId: '42' },
      search: {
        listPage: 3,
        listFilter: 'alice',
        listStatus: '1',
        listRole: '10',
        listGroup: 'list-group',
        listSortBy: 'username',
        listSortOrder: 'asc',
        ...analyticsFilters,
      },
    })
  })

  test('back action restores list state without dropping analytics filters', async () => {
    const user = userEvent.setup()
    renderWithQueryClient(<UserAnalyticsDetail />)

    await user.click(screen.getByRole('button', { name: 'Back to users' }))

    expect(routerMocks.globalNavigate).toHaveBeenCalledWith({
      to: '/dashboard/$section',
      params: { section: 'users' },
      search: {
        page: 3,
        filter: 'alice',
        status: '1',
        role: '10',
        group: 'list-group',
        sortBy: 'username',
        sortOrder: 'asc',
        startTimestamp: analyticsFilters.startTimestamp,
        endTimestamp: analyticsFilters.endTimestamp,
        model: 'gpt-4.1',
        token: 'production',
        usageGroup: 'analytics-group',
        channel: '7',
      },
    })
  })

  test('failed user list can be retried without leaving analytics', async () => {
    const user = userEvent.setup()
    userApiMocks.searchUsers.mockRejectedValueOnce(new Error('network'))
    renderWithQueryClient(<AnalyticsUserTable filters={analyticsFilters} />)

    expect(await screen.findByText('Failed to load users')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Retry' }))

    expect(await screen.findByText('alice')).toBeVisible()
    expect(userApiMocks.searchUsers).toHaveBeenCalledTimes(2)
  })

  test('failed user summary can be retried in place', async () => {
    const user = userEvent.setup()
    dashboardApiMocks.getDashboardUser.mockRejectedValueOnce(
      new Error('network')
    )
    renderWithQueryClient(<UserAnalyticsDetail />)

    expect(
      await screen.findByText('Failed to load user details.')
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Retry' }))

    expect(await screen.findByText('@alice / #42')).toBeVisible()
    expect(dashboardApiMocks.getDashboardUser).toHaveBeenCalledTimes(2)
  })
})
