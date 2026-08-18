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
import type { PropsWithChildren, ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { Home } from '../index'

const statusMock = vi.hoisted(() => ({
  current: {} as {
    self_use_mode_enabled?: boolean
    register_enabled?: boolean
  },
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { children?: ReactNode; to: string }) => (
    <a href={props.to}>{props.children}</a>
  ),
}))

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: PropsWithChildren) => props.children,
}))

vi.mock('@/components/rich-content', () => ({
  RichContent: () => null,
}))

vi.mock('@/context/theme-provider', () => ({
  useTheme: () => ({ resolvedTheme: 'light' }),
}))

vi.mock('@/features/auth/sign-in/components/user-auth-form', () => ({
  UserAuthForm: () => <div aria-label='Sign-in form' />,
}))

vi.mock('@/features/home/hooks', () => ({
  useHomePageContent: () => ({ content: '', isLoaded: true, isUrl: false }),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: statusMock.current }),
}))

describe('home registration entry', () => {
  test('shows a link to sign up when registration is enabled', () => {
    statusMock.current = {
      self_use_mode_enabled: false,
      register_enabled: true,
    }

    render(<Home />)

    expect(screen.getByRole('link', { name: 'Sign up' })).toHaveAttribute(
      'href',
      '/sign-up'
    )
  })

  test.each([
    [
      'self-use mode is enabled',
      { self_use_mode_enabled: true, register_enabled: true },
    ],
    [
      'registration is disabled',
      { self_use_mode_enabled: false, register_enabled: false },
    ],
  ])('hides the sign-up link when %s', (_scenario, status) => {
    statusMock.current = status

    render(<Home />)

    expect(
      screen.queryByRole('link', { name: 'Sign up' })
    ).not.toBeInTheDocument()
  })
})
