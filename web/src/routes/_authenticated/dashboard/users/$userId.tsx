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
import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { UserAnalyticsDetail } from '@/features/dashboard/components/users/user-analytics-detail'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const userAnalyticsSearchSchema = z.object({
  logPage: z.number().optional().catch(1),
  startTimestamp: z.number().optional().catch(undefined),
  endTimestamp: z.number().optional().catch(undefined),
  model: z.string().optional().catch(''),
  token: z.string().optional().catch(''),
  group: z.string().optional().catch(''),
  channel: z.string().optional().catch(''),
  listPage: z.number().optional().catch(1),
  listFilter: z.string().optional().catch(''),
  listStatus: z.string().optional().catch(''),
  listRole: z.string().optional().catch(''),
  listGroup: z.string().optional().catch(''),
  listSortBy: z
    .enum(['id', 'username', 'quota', 'group', 'created_at', 'last_login_at'])
    .optional()
    .catch('id'),
  listSortOrder: z.enum(['asc', 'desc']).optional().catch('desc'),
})

export const Route = createFileRoute('/_authenticated/dashboard/users/$userId')(
  {
    beforeLoad: ({ params }) => {
      const role = useAuthStore.getState().auth.user?.role ?? ROLE.GUEST
      const userId = Number(params.userId)
      if (role < ROLE.ADMIN) {
        throw redirect({
          to: '/dashboard/$section',
          params: { section: 'models' },
        })
      }
      if (!Number.isInteger(userId) || userId <= 0) {
        throw redirect({
          to: '/dashboard/$section',
          params: { section: 'users' },
        })
      }
    },
    validateSearch: userAnalyticsSearchSchema,
    component: UserAnalyticsDetail,
  }
)
