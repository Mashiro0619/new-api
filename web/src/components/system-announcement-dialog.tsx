/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { getNotice } from '@/lib/api'
import { consumeAuthLoginEvent } from '@/lib/auth-session'
import { useAuthStore } from '@/stores/auth-store'
import { useNotificationStore } from '@/stores/notification-store'

export function SystemAnnouncementDialog() {
  const { t } = useTranslation()
  const sessionId = useAuthStore((state) => state.auth.session?.sid)
  const { data, isFetching } = useQuery({
    queryKey: ['notice'],
    queryFn: getNotice,
    staleTime: 1000 * 60 * 5,
  })
  const [open, setOpen] = useState(false)
  const [loginEventPending, setLoginEventPending] = useState(false)
  const isClosedToday = useNotificationStore((state) => state.isNoticeClosed())
  const setClosedUntilDate = useNotificationStore(
    (state) => state.setClosedUntilDate
  )
  const markNoticeRead = useNotificationStore((state) => state.markNoticeRead)

  const notice = data?.success ? (data.data ?? '').trim() : ''

  useEffect(() => {
    if (sessionId && consumeAuthLoginEvent(sessionId)) {
      setLoginEventPending(true)
    }
  }, [sessionId])

  useEffect(() => {
    if (isFetching || !loginEventPending) return

    setLoginEventPending(false)
    if (notice && !isClosedToday) {
      markNoticeRead(notice)
      setOpen(true)
    }
  }, [isClosedToday, isFetching, loginEventPending, markNoticeRead, notice])

  const closeDialog = (closeForToday = false) => {
    if (closeForToday) {
      setClosedUntilDate(new Date().toDateString())
    }
    setOpen(false)
  }

  if (!notice) return null

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) closeDialog()
      }}
      title={t('System Announcements')}
      description={t('Latest platform updates and notices')}
      contentHeight='auto'
      bodyClassName='max-h-[min(60vh,32rem)] overflow-y-auto'
      footer={
        <>
          <Button variant='outline' onClick={() => closeDialog()}>
            {t('Close')}
          </Button>
          <Button onClick={() => closeDialog(true)}>{t('Close Today')}</Button>
        </>
      }
    >
      <RichContent breaks content={notice} />
    </Dialog>
  )
}
