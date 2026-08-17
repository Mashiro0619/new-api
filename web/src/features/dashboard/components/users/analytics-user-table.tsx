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
import {
  ArrowLeft01Icon,
  ArrowRight01Icon,
  ArrowUpDownIcon,
  Search01Icon,
  UserMultipleIcon,
  ViewIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { GroupBadge } from '@/components/group-badge'
import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getGroups, getUsers, searchUsers } from '@/features/users/api'
import {
  USER_ROLES,
  USER_STATUSES,
  USER_STATUS,
  isUserDeleted,
} from '@/features/users/constants'
import type { UserSortBy } from '@/features/users/types'
import { formatCompactNumber, formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { DashboardUserFilters } from '../../types'

const route = getRouteApi('/_authenticated/dashboard/$section')
const PAGE_SIZE = 20

export function AnalyticsUserTable(props: { filters: DashboardUserFilters }) {
  const { t } = useTranslation()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const [keyword, setKeyword] = useState(search.filter ?? '')
  const page = Math.max(search.page ?? 1, 1)
  const sortBy = (search.sortBy ?? 'id') as UserSortBy
  const sortOrder = search.sortOrder ?? 'desc'

  const groupsQuery = useQuery({
    queryKey: ['dashboard', 'analytics-user-groups'],
    queryFn: getGroups,
    staleTime: 5 * 60_000,
  })
  const usersQuery = useQuery({
    queryKey: [
      'dashboard',
      'analytics-users',
      page,
      search.filter,
      search.status,
      search.role,
      search.group,
      sortBy,
      sortOrder,
    ],
    queryFn: async () => {
      const params = {
        p: page,
        page_size: PAGE_SIZE,
        sort_by: sortBy,
        sort_order: sortOrder,
      } as const
      const hasFilters = Boolean(
        search.filter || search.status || search.role || search.group
      )
      const result = hasFilters
        ? await searchUsers({
            ...params,
            keyword: search.filter,
            status: search.status,
            role: search.role,
            group: search.group,
          })
        : await getUsers(params)
      if (!result.success) throw new Error(result.message || 'users')
      return result.data ?? { items: [], total: 0, page, page_size: PAGE_SIZE }
    },
    placeholderData: (previous) => previous,
  })

  useEffect(() => {
    setKeyword(search.filter ?? '')
  }, [search.filter])

  const updateSearch = (changes: Record<string, unknown>) => {
    void navigate({
      search: (previous) => ({ ...previous, page: 1, ...changes }),
    })
  }
  const toggleSort = (column: UserSortBy) => {
    updateSearch({
      sortBy: column,
      sortOrder:
        sortBy === column && sortOrder === 'desc' ? ('asc' as const) : 'desc',
    })
  }
  const total = usersQuery.data?.total ?? 0
  const pageCount = Math.max(Math.ceil(total / PAGE_SIZE), 1)

  useEffect(() => {
    if (
      !usersQuery.isLoading &&
      !usersQuery.isPlaceholderData &&
      page > pageCount
    ) {
      void navigate({
        search: (previous) => ({ ...previous, page: pageCount }),
        replace: true,
      })
    }
  }, [
    navigate,
    page,
    pageCount,
    usersQuery.isLoading,
    usersQuery.isPlaceholderData,
  ])

  return (
    <section className='overflow-hidden rounded-lg border'>
      <div className='flex flex-col gap-3 border-b px-4 py-3 sm:px-5'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div>
            <h2 className='text-sm font-semibold'>{t('User Analytics')}</h2>
            <p className='text-muted-foreground text-xs'>
              {t('{{count}} users', { count: total })}
            </p>
          </div>
          <form
            className='flex w-full gap-2 sm:w-auto'
            onSubmit={(event) => {
              event.preventDefault()
              updateSearch({ filter: keyword.trim() || undefined })
            }}
          >
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('Filter by username, name or email...')}
              aria-label={t('Filter by username, name or email...')}
              className='min-w-0 sm:w-72'
            />
            <Button type='submit' size='icon' aria-label={t('Search')}>
              <HugeiconsIcon icon={Search01Icon} />
            </Button>
          </form>
        </div>
        <div className='flex flex-wrap gap-2'>
          <NativeSelect
            size='sm'
            value={search.status ?? ''}
            aria-label={t('Status')}
            onChange={(event) =>
              updateSearch({ status: event.target.value || undefined })
            }
          >
            <NativeSelectOption value=''>
              {t('All statuses')}
            </NativeSelectOption>
            <NativeSelectOption value='1'>{t('Enabled')}</NativeSelectOption>
            <NativeSelectOption value='2'>{t('Disabled')}</NativeSelectOption>
            <NativeSelectOption value='-1'>{t('Deleted')}</NativeSelectOption>
          </NativeSelect>
          <NativeSelect
            size='sm'
            value={search.role ?? ''}
            aria-label={t('Role')}
            onChange={(event) =>
              updateSearch({ role: event.target.value || undefined })
            }
          >
            <NativeSelectOption value=''>{t('All roles')}</NativeSelectOption>
            <NativeSelectOption value='1'>{t('User')}</NativeSelectOption>
            <NativeSelectOption value='10'>{t('Admin')}</NativeSelectOption>
            <NativeSelectOption value='100'>{t('Root')}</NativeSelectOption>
          </NativeSelect>
          <NativeSelect
            size='sm'
            value={search.group ?? ''}
            aria-label={t('Group')}
            onChange={(event) =>
              updateSearch({ group: event.target.value || undefined })
            }
          >
            <NativeSelectOption value=''>{t('All groups')}</NativeSelectOption>
            {(groupsQuery.data?.data ?? []).map((group) => (
              <NativeSelectOption key={group} value={group}>
                {group}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
      </div>

      {usersQuery.isError && (
        <ErrorState
          className='min-h-64'
          title={t('Failed to load users')}
          description={t('Please try again later.')}
          onRetry={() => void usersQuery.refetch()}
        />
      )}
      {!usersQuery.isError && usersQuery.isLoading && (
        <div className='flex flex-col gap-2 p-4'>
          {Array.from({ length: 6 }, (_, index) => (
            <Skeleton key={index} className='h-12 w-full' />
          ))}
        </div>
      )}
      {!usersQuery.isError && !usersQuery.isLoading && total === 0 && (
        <Empty className='min-h-64'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <HugeiconsIcon icon={UserMultipleIcon} />
            </EmptyMedia>
            <EmptyTitle>{t('No Users Found')}</EmptyTitle>
            <EmptyDescription>
              {t('No users available. Try adjusting your search or filters.')}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
      {!usersQuery.isError && !usersQuery.isLoading && total > 0 && (
        <div
          aria-busy={usersQuery.isFetching}
          className={cn(
            'transition-opacity duration-150',
            usersQuery.isFetching && 'pointer-events-none opacity-60'
          )}
        >
          <UserRowsTable
            users={usersQuery.data?.items ?? []}
            sortBy={sortBy}
            toggleSort={toggleSort}
            filters={props.filters}
          />
        </div>
      )}

      {total > 0 && (
        <div className='flex items-center justify-between gap-3 border-t px-4 py-3 sm:px-5'>
          <span className='text-muted-foreground text-xs'>
            {t('Page {{page}} of {{total}}', { page, total: pageCount })}
          </span>
          <div className='flex gap-1'>
            <Button
              size='icon'
              variant='outline'
              disabled={page <= 1}
              aria-label={t('Previous page')}
              onClick={() =>
                void navigate({
                  search: (previous) => ({ ...previous, page: page - 1 }),
                })
              }
            >
              <HugeiconsIcon icon={ArrowLeft01Icon} />
            </Button>
            <Button
              size='icon'
              variant='outline'
              disabled={page >= pageCount}
              aria-label={t('Next page')}
              onClick={() =>
                void navigate({
                  search: (previous) => ({ ...previous, page: page + 1 }),
                })
              }
            >
              <HugeiconsIcon icon={ArrowRight01Icon} />
            </Button>
          </div>
        </div>
      )}
    </section>
  )
}

interface UserRowsTableProps {
  users: NonNullable<Awaited<ReturnType<typeof getUsers>>['data']>['items']
  sortBy: UserSortBy
  toggleSort: (column: UserSortBy) => void
  filters: DashboardUserFilters
}

function UserRowsTable(props: UserRowsTableProps) {
  const { t } = useTranslation()
  const search = route.useSearch()
  const sortableHead = (label: string, column: UserSortBy) => (
    <Button variant='ghost' size='sm' onClick={() => props.toggleSort(column)}>
      {label}
      <HugeiconsIcon icon={ArrowUpDownIcon} data-icon='inline-end' />
    </Button>
  )

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{sortableHead(t('User'), 'username')}</TableHead>
          <TableHead>{t('Role')}</TableHead>
          <TableHead>{sortableHead(t('Group'), 'group')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead className='text-right'>{t('Requests')}</TableHead>
          <TableHead className='text-right'>{t('Usage')}</TableHead>
          <TableHead className='w-16 text-right'>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {props.users.map((user) => {
          const role = USER_ROLES[user.role as keyof typeof USER_ROLES]
          const status = isUserDeleted(user)
            ? USER_STATUSES[USER_STATUS.DELETED]
            : USER_STATUSES[user.status as keyof typeof USER_STATUSES]
          return (
            <TableRow key={user.id}>
              <TableCell>
                <div className='flex min-w-48 items-center gap-2.5'>
                  <Avatar className='size-8'>
                    <AvatarFallback>
                      {(user.display_name || user.username).slice(0, 2)}
                    </AvatarFallback>
                  </Avatar>
                  <div className='min-w-0'>
                    <div className='truncate font-medium'>{user.username}</div>
                    <div className='text-muted-foreground max-w-48 truncate text-xs'>
                      {user.display_name || user.email || `#${user.id}`}
                    </div>
                  </div>
                </div>
              </TableCell>
              <TableCell>{role ? t(role.labelKey) : '-'}</TableCell>
              <TableCell>
                <GroupBadge group={user.group} />
              </TableCell>
              <TableCell>
                {status ? (
                  <StatusBadge
                    label={t(status.labelKey)}
                    variant={status.variant}
                    copyable={false}
                  />
                ) : (
                  '-'
                )}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatCompactNumber(user.request_count)}
              </TableCell>
              <TableCell className='text-right font-mono'>
                {formatQuota(user.used_quota)}
              </TableCell>
              <TableCell className='text-right'>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        size='icon'
                        variant='ghost'
                        nativeButton={false}
                        render={
                          <Link
                            to='/dashboard/users/$userId'
                            params={{ userId: String(user.id) }}
                            search={{
                              listPage: search.page,
                              listFilter: search.filter,
                              listStatus: search.status,
                              listRole: search.role,
                              listGroup: search.group,
                              listSortBy: search.sortBy,
                              listSortOrder: search.sortOrder,
                              startTimestamp: props.filters.startTimestamp,
                              endTimestamp: props.filters.endTimestamp,
                              model: props.filters.model || undefined,
                              token: props.filters.token || undefined,
                              group: props.filters.group || undefined,
                              channel: props.filters.channel || undefined,
                            }}
                          />
                        }
                        aria-label={t('View usage')}
                      />
                    }
                  >
                    <HugeiconsIcon icon={ViewIcon} />
                  </TooltipTrigger>
                  <TooltipContent>{t('View usage')}</TooltipContent>
                </Tooltip>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
