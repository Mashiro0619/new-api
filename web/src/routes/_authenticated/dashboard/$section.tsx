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

import { Dashboard } from '@/features/dashboard'
import { canAccessDashboardSection } from '@/features/dashboard/lib/access'
import {
  DASHBOARD_DEFAULT_SECTION,
  DASHBOARD_SECTION_IDS,
  type DashboardSectionId,
} from '@/features/dashboard/section-registry'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const dashboardSearchSchema = z.object({
  page: z.number().optional().catch(1),
  filter: z.string().optional().catch(''),
  status: z.string().optional().catch(''),
  role: z.string().optional().catch(''),
  group: z.string().optional().catch(''),
  sortBy: z
    .enum(['id', 'username', 'quota', 'group', 'created_at', 'last_login_at'])
    .optional()
    .catch('id'),
  sortOrder: z.enum(['asc', 'desc']).optional().catch('desc'),
  startTimestamp: z.number().optional().catch(undefined),
  endTimestamp: z.number().optional().catch(undefined),
  model: z.string().optional().catch(''),
  token: z.string().optional().catch(''),
  usageGroup: z.string().optional().catch(''),
  channel: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/dashboard/$section')({
  beforeLoad: ({ params }) => {
    const validSections = DASHBOARD_SECTION_IDS as unknown as string[]
    if (!validSections.includes(params.section)) {
      throw redirect({
        to: '/dashboard/$section',
        params: { section: DASHBOARD_DEFAULT_SECTION },
      })
    }

    const role = useAuthStore.getState().auth.user?.role ?? ROLE.GUEST
    if (
      !canAccessDashboardSection(params.section as DashboardSectionId, role)
    ) {
      throw redirect({
        to: '/dashboard/$section',
        params: { section: 'models' },
      })
    }
  },
  validateSearch: dashboardSearchSchema,
  component: Dashboard,
})
