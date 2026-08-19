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
import { Link } from '@tanstack/react-router'
import { useCallback, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { RichContent } from '@/components/rich-content'
import { useTheme } from '@/context/theme-provider'
import { UserAuthForm } from '@/features/auth/sign-in/components/user-auth-form'
import { useStatus } from '@/hooks/use-status'
import { isLikelyHtml } from '@/lib/content-format'

import { useHomePageContent } from './hooks'

function DefaultHome() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const canRegister =
    !status?.self_use_mode_enabled && status?.register_enabled !== false

  return (
    <PublicLayout
      showMainContainer={false}
      showAuthButtons={false}
      headerProps={{ showBrand: false }}
    >
      <main className='flex min-h-svh items-center pt-16'>
        <div className='mx-auto grid w-full max-w-7xl items-center gap-14 px-6 py-16 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,28rem)] lg:gap-24'>
          <h1 className='text-center text-[clamp(3.5rem,7vw,7rem)] leading-[0.9] font-semibold tracking-[-0.06em]'>
            <span className='whitespace-nowrap'>Mashiro AI</span>
            <span className='block text-center whitespace-nowrap'>中转站</span>
          </h1>
          <section className='w-full max-w-md justify-self-end'>
            <div className='mb-6 space-y-2'>
              <h2 className='text-2xl font-semibold tracking-tight'>
                {t('Sign in')}
              </h2>
              {canRegister && (
                <p className='text-muted-foreground text-sm sm:text-base'>
                  {t("Don't have an account?")}{' '}
                  <Link
                    to='/sign-up'
                    className='hover:text-primary font-medium underline underline-offset-4'
                  >
                    {t('Sign up')}
                  </Link>
                  .
                </p>
              )}
            </div>
            <UserAuthForm />
          </section>
        </div>
      </main>
    </PublicLayout>
  )
}

export function Home() {
  const { i18n, t } = useTranslation()
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const { resolvedTheme } = useTheme()
  const { content, isLoaded, isUrl } = useHomePageContent()

  const syncIframePreferences = useCallback(() => {
    try {
      iframeRef.current?.contentWindow?.postMessage(
        { themeMode: resolvedTheme },
        '*'
      )
      iframeRef.current?.contentWindow?.postMessage(
        { lang: i18n.language },
        '*'
      )
    } catch {
      // Cross-origin frames may reject access while navigating.
    }
  }, [i18n.language, resolvedTheme])

  useEffect(() => {
    if (isUrl) {
      syncIframePreferences()
    }
  }, [isUrl, syncIframePreferences])

  if (!isLoaded) {
    return <DefaultHome />
  }

  if (content) {
    if (isUrl) {
      return (
        <PublicLayout showMainContainer={false}>
          {/*
            allow-top-navigation-by-user-activation: the custom home page URL is
            admin-configured (trusted); this lets its target="_top" nav/menu links
            navigate the top-level window on user click. The default sandbox blocks
            this on desktop, while some mobile browsers allow it via allow-popups,
            causing inconsistent behavior. This token only permits user-activated
            top-level navigation and does NOT grant same-origin access.
          */}
          <iframe
            ref={iframeRef}
            src={content}
            className='h-screen w-full border-none'
            title={t('Custom Home Page')}
            sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-scripts allow-top-navigation-by-user-activation'
            onLoad={syncIframePreferences}
          />
        </PublicLayout>
      )
    }

    const contentIsHtml = isLikelyHtml(content)

    if (contentIsHtml) {
      return (
        <PublicLayout showMainContainer={false}>
          <RichContent
            mode='html'
            htmlVariant='isolated'
            content={content}
            className='custom-home-content'
          />
        </PublicLayout>
      )
    }

    return (
      <PublicLayout>
        <div className='mx-auto max-w-6xl px-4 py-8'>
          <RichContent
            mode='markdown'
            content={content}
            className='custom-home-content'
          />
        </div>
      </PublicLayout>
    )
  }

  return <DefaultHome />
}
