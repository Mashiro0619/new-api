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
import {
  formatCurrencyFromUSD,
  formatLocalCurrencyAmount,
} from '@/lib/currency'

import type { TopupRecord } from '../../types'

interface PaymentSuccessDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  record: TopupRecord | null
}

export function PaymentSuccessDialog(props: PaymentSuccessDialogProps) {
  const record = props.record

  return (
    <AlertDialog open={props.open} onOpenChange={props.onOpenChange}>
      <AlertDialogContent className='gap-0 overflow-hidden p-0 max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'>
        <div className='flex flex-col items-center px-5 pt-5 text-center sm:px-6 sm:pt-6'>
          <AlertDialogMedia className='mb-3 size-12 rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'>
            <CheckCircle2 className='size-6' aria-hidden='true' />
          </AlertDialogMedia>
          <AlertDialogHeader className='!place-items-center !text-center'>
            <AlertDialogTitle className='text-xl font-semibold'>
              支付成功
            </AlertDialogTitle>
            <AlertDialogDescription>
              支付已到账，余额已更新。
            </AlertDialogDescription>
          </AlertDialogHeader>
        </div>

        {record ? (
          <div className='mx-5 mt-5 space-y-2 sm:mx-6'>
            <div className='rounded-lg border bg-emerald-50 px-4 py-3 text-center dark:bg-emerald-950/30'>
              <div className='text-muted-foreground text-xs'>充值额度</div>
              <div className='mt-1 text-2xl font-semibold text-emerald-700 tabular-nums dark:text-emerald-300'>
                {formatCurrencyFromUSD(record.amount, {
                  digitsLarge: 2,
                  digitsSmall: 2,
                  abbreviate: false,
                })}
              </div>
            </div>

            <div className='grid grid-cols-2 gap-2'>
              <div className='bg-muted/50 rounded-lg px-3 py-2.5'>
                <div className='text-muted-foreground text-xs'>支付金额</div>
                <div className='mt-1 text-sm font-semibold tabular-nums'>
                  {formatLocalCurrencyAmount(record.money, {
                    digitsLarge: 2,
                    digitsSmall: 2,
                    abbreviate: false,
                  })}
                </div>
              </div>
              <div className='bg-muted/50 min-w-0 rounded-lg px-3 py-2.5'>
                <div className='text-muted-foreground text-xs'>订单号</div>
                <code className='text-foreground mt-1 block truncate font-mono text-xs'>
                  {record.trade_no}
                </code>
              </div>
            </div>
          </div>
        ) : null}

        <AlertDialogFooter className='!mx-0 mt-5 !mb-0 px-5 pb-5 sm:px-6 sm:pb-6'>
          <AlertDialogAction
            className='w-full sm:w-auto'
            onClick={() => props.onOpenChange(false)}
          >
            完成
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
