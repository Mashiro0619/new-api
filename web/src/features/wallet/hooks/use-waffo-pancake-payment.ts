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
import i18next from 'i18next'
import { useState, useCallback } from 'react'
import { toast } from 'sonner'

import { requestWaffoPancakePayment, isApiSuccess } from '../api'

function getCheckoutUrl(data: unknown): string | null {
  if (!data || typeof data !== 'object') {
    return null
  }

  if ('checkout_url' in data && typeof data.checkout_url === 'string') {
    return data.checkout_url
  }

  return null
}

/**
 * Reject non-navigable schemes (e.g. javascript:, data:) and relative URLs.
 * Only http/https are allowed for backend-provided redirect targets.
 */
function isSafeHttpCheckoutUrl(value: string): boolean {
  const trimmed = value.trim()
  if (!trimmed) {
    return false
  }
  try {
    const u = new URL(trimmed)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

function getErrorMessage(message: string | undefined, data: unknown): string {
  if (typeof data === 'string' && data.trim()) {
    return data
  }

  return message || i18next.t('Payment request failed')
}

/** Hook for the Waffo Pancake hosted-checkout flow. */
export function useWaffoPancakePayment() {
  const [processing, setProcessing] = useState(false)

  const processWaffoPancakePayment = useCallback(
    async (topupAmount: number) => {
      setProcessing(true)
      // Open synchronously while the confirmation click is still trusted by
      // the browser. The checkout URL arrives after the API request.
      const checkoutWindow = window.open('', '_blank')

      try {
        const response = await requestWaffoPancakePayment({
          amount: Math.floor(topupAmount),
        })

        if (isApiSuccess(response)) {
          const checkoutUrl = getCheckoutUrl(response.data)

          if (checkoutUrl) {
            if (!isSafeHttpCheckoutUrl(checkoutUrl)) {
              checkoutWindow?.close()
              toast.error(i18next.t('Invalid payment redirect URL'))
              return false
            }
            toast.success(i18next.t('Redirecting to payment page...'))
            if (checkoutWindow) {
              checkoutWindow.location.href = checkoutUrl
            } else {
              window.open(checkoutUrl, '_blank')
            }
            return true
          }
        }

        checkoutWindow?.close()
        toast.error(getErrorMessage(response.message, response.data))
        return false
      } catch {
        checkoutWindow?.close()
        toast.error(i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  return { processing, processWaffoPancakePayment }
}
