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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ChevronLeft,
  ChevronRight,
  Save,
  ShieldAlert,
  Trash2,
} from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StaticDataTable } from '@/components/data-table'
import { SectionPageLayout } from '@/components/layout'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

import {
  clearProhibitedWordStats,
  getProhibitedWordConfig,
  getProhibitedWordSummary,
  updateProhibitedWordConfig,
} from './api'

const PAGE_SIZE = 50

function normalizeKeywords(value: string): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const raw of value.split('\n')) {
    const keyword = raw.trim()
    const key = keyword.toLocaleLowerCase()
    if (!keyword || seen.has(key)) continue
    seen.add(key)
    result.push(keyword)
  }
  return result
}

function formatCount(value: number | undefined): string {
  return new Intl.NumberFormat().format(value ?? 0)
}

export function ProhibitedWordDetection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [keywordText, setKeywordText] = useState('')
  const [clearDialogOpen, setClearDialogOpen] = useState(false)

  const configQuery = useQuery({
    queryKey: ['prohibited-word-detection', 'config'],
    queryFn: getProhibitedWordConfig,
  })
  const summaryQuery = useQuery({
    queryKey: ['prohibited-word-detection', 'summary', page],
    queryFn: () => getProhibitedWordSummary(page, PAGE_SIZE),
  })
  const updateConfigMutation = useMutation({
    mutationFn: updateProhibitedWordConfig,
    onSuccess: async () => {
      setPage(1)
      await queryClient.invalidateQueries({
        queryKey: ['prohibited-word-detection'],
      })
      toast.success(t('Keywords saved'))
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const clearStatsMutation = useMutation({
    mutationFn: clearProhibitedWordStats,
    onSuccess: async () => {
      setClearDialogOpen(false)
      await queryClient.invalidateQueries({
        queryKey: ['prohibited-word-detection', 'summary'],
      })
      toast.success(t('Statistics cleared'))
    },
    onError: (error: Error) => toast.error(error.message),
  })

  useEffect(() => {
    setKeywordText((configQuery.data?.data.keywords ?? []).join('\n'))
  }, [configQuery.data?.data.keywords])

  const keywords = configQuery.data?.data.keywords ?? []
  const items = summaryQuery.data?.data.items ?? []
  const total = summaryQuery.data?.data.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const columns = [
    {
      id: 'username',
      header: t('Username'),
      cellClassName: 'font-medium whitespace-nowrap',
      cell: (item: (typeof items)[number]) => item.username,
    },
    ...keywords.map((keyword) => ({
      id: `keyword-${keyword}`,
      header: keyword,
      className: 'text-right whitespace-nowrap',
      cellClassName: 'text-right tabular-nums',
      cell: (item: (typeof items)[number]) => formatCount(item.counts[keyword]),
    })),
  ]

  let content: ReactNode
  if (configQuery.isLoading || summaryQuery.isLoading) {
    content = (
      <div className='text-muted-foreground flex flex-1 items-center justify-center text-sm'>
        {t('Loading...')}
      </div>
    )
  } else if (configQuery.error || summaryQuery.error) {
    content = (
      <div className='text-destructive flex flex-1 items-center justify-center text-sm'>
        {t('Failed to load prohibited word statistics')}
      </div>
    )
  } else if (items.length === 0) {
    content = (
      <div className='text-muted-foreground flex flex-1 items-center justify-center text-sm'>
        {t('No users found')}
      </div>
    )
  } else {
    content = (
      <div className='min-h-0 flex-1 overflow-auto'>
        <StaticDataTable
          tableClassName='min-w-[720px]'
          data={items}
          getRowKey={(item) => item.user_id}
          columns={columns}
        />
      </div>
    )
  }

  const handleSave = () => {
    updateConfigMutation.mutate(normalizeKeywords(keywordText))
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Prohibited word detection')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='destructive'
          onClick={() => setClearDialogOpen(true)}
          disabled={clearStatsMutation.isPending}
        >
          <Trash2 className='mr-2 size-4' />
          {t('Clear statistics')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-4'>
          <div className='bg-muted/20 shrink-0 space-y-3 rounded-lg border p-4'>
            <div className='flex items-center gap-2'>
              <ShieldAlert className='text-destructive size-5' />
              <h3 className='font-semibold'>{t('Prohibited words')}</h3>
            </div>
            <Textarea
              value={keywordText}
              onChange={(event) => setKeywordText(event.target.value)}
              rows={5}
              placeholder={t('Enter one keyword per line')}
              aria-label={t('Prohibited words')}
            />
            <div className='flex justify-end'>
              <Button
                onClick={handleSave}
                disabled={updateConfigMutation.isPending}
              >
                <Save className='mr-2 size-4' />
                {updateConfigMutation.isPending
                  ? t('Saving...')
                  : t('Save keywords')}
              </Button>
            </div>
          </div>

          {content}

          <div className='flex shrink-0 items-center justify-between gap-3 border-t pt-3'>
            <span className='text-muted-foreground text-sm'>
              {t('Page {{page}} of {{total}}', {
                page,
                total: totalPages,
              })}
            </span>
            <div className='flex items-center gap-2'>
              <Button
                variant='outline'
                size='icon'
                onClick={() => setPage((current) => Math.max(1, current - 1))}
                disabled={page <= 1 || summaryQuery.isFetching}
                aria-label={t('Previous')}
                title={t('Previous')}
              >
                <ChevronLeft />
              </Button>
              <Button
                variant='outline'
                size='icon'
                onClick={() =>
                  setPage((current) => Math.min(totalPages, current + 1))
                }
                disabled={page >= totalPages || summaryQuery.isFetching}
                aria-label={t('Next')}
                title={t('Next')}
              >
                <ChevronRight />
              </Button>
            </div>
          </div>
        </div>
      </SectionPageLayout.Content>

      <AlertDialog open={clearDialogOpen} onOpenChange={setClearDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Clear statistics')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('This will delete all prohibited word hit statistics.')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              onClick={() => clearStatsMutation.mutate()}
              disabled={clearStatsMutation.isPending}
            >
              {t('Clear')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SectionPageLayout>
  )
}
