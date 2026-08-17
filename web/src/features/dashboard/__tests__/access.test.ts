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
import { describe, expect, test } from 'vitest'

import { ROLE } from '@/lib/roles'

import {
  canAccessDashboardSection,
  getDashboardLandingSection,
} from '../lib/access'

describe('dashboard analytics access', () => {
  test('admin roles land on user analytics', () => {
    expect(getDashboardLandingSection(ROLE.ADMIN)).toBe('users')
    expect(getDashboardLandingSection(ROLE.SUPER_ADMIN)).toBe('users')
  })

  test('common users land on their model analytics', () => {
    expect(getDashboardLandingSection(ROLE.USER)).toBe('models')
  })

  test('user analytics rejects non-admin roles', () => {
    expect(canAccessDashboardSection('users', ROLE.USER)).toBe(false)
    expect(canAccessDashboardSection('users', ROLE.ADMIN)).toBe(true)
    expect(canAccessDashboardSection('models', ROLE.USER)).toBe(true)
  })
})
