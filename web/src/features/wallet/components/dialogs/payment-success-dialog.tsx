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
import { CheckCircle2 } from 'lucide-react'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

import type { TopupRecord } from '../../types'

interface PaymentSuccessDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  record: TopupRecord | null
}

export function PaymentSuccessDialog(props: PaymentSuccessDialogProps) {
  return (
    <AlertDialog open={props.open} onOpenChange={props.onOpenChange}>
      <AlertDialogContent className='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'>
        <AlertDialogMedia className='bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'>
          <CheckCircle2 className='size-6' aria-hidden='true' />
        </AlertDialogMedia>
        <AlertDialogHeader>
          <AlertDialogTitle>支付成功</AlertDialogTitle>
          <AlertDialogDescription>
            支付已到账，余额已更新。
          </AlertDialogDescription>
        </AlertDialogHeader>

        {props.record ? (
          <div className='bg-muted/50 flex items-center justify-between gap-3 rounded-lg px-3 py-2 text-sm'>
            <span className='text-muted-foreground'>订单号</span>
            <code className='text-foreground min-w-0 truncate font-mono'>
              {props.record.trade_no}
            </code>
          </div>
        ) : null}

        <AlertDialogFooter>
          <AlertDialogAction>完成</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
