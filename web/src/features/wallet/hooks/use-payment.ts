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

import {
  calculateAmount,
  calculateStripeAmount,
  calculateWaffoAmount,
  calculateWaffoPancakeAmount,
  requestPayment,
  requestStripePayment,
  requestPerPayPayment,
  isApiSuccess,
} from '../api'
import {
  isStripePayment,
  isWaffoPayment,
  isWaffoPancakePayment,
  isPerPayPayment,
  submitPaymentForm,
} from '../lib'
import type { AmountRequest, AmountResponse, ApiResponse } from '../types'

// ============================================================================
// Payment Hook
// ============================================================================

type AmountCalculator = (request: AmountRequest) => Promise<AmountResponse>

export interface PaymentAmountCalculators {
  regular: AmountCalculator
  stripe: AmountCalculator
  waffo: AmountCalculator
  waffoPancake: AmountCalculator
}

type PreparedPerPayPayment = {
  topupAmount: number
  checkoutUrl: string
  tradeNo: string
  payableAmount: number
}

const defaultPaymentAmountCalculators: PaymentAmountCalculators = {
  regular: calculateAmount,
  stripe: calculateStripeAmount,
  waffo: calculateWaffoAmount,
  waffoPancake: calculateWaffoPancakeAmount,
}

export async function requestPaymentAmount(
  topupAmount: number,
  paymentType: string,
  calculators: PaymentAmountCalculators = defaultPaymentAmountCalculators
): Promise<number> {
  let calculator = calculators.regular
  if (isStripePayment(paymentType)) {
    calculator = calculators.stripe
  } else if (isWaffoPayment(paymentType)) {
    calculator = calculators.waffo
  } else if (isWaffoPancakePayment(paymentType)) {
    calculator = calculators.waffoPancake
  }

  const response = await calculator({ amount: topupAmount })
  if (!isApiSuccess(response) || !response.data) {
    throw new Error(
      getApiErrorMessage(response) || i18next.t('Payment request failed')
    )
  }

  const amount = Number.parseFloat(response.data)
  if (!Number.isFinite(amount) || amount <= 0) {
    throw new Error(i18next.t('Payment request failed'))
  }

  return amount
}

export function usePayment() {
  const [amount, setAmount] = useState<number>(0)
  const [calculating, setCalculating] = useState(false)
  const [processing, setProcessing] = useState(false)
  const [preparedPerPayPayment, setPreparedPerPayPayment] =
    useState<PreparedPerPayPayment | null>(null)
  const [pendingPerPayTradeNo, setPendingPerPayTradeNo] = useState<
    string | null
  >(null)
  const clearPendingPerPayTradeNo = useCallback(() => {
    setPendingPerPayTradeNo(null)
  }, [])

  const clearPreparedPerPayPayment = useCallback(() => {
    setPreparedPerPayPayment(null)
  }, [])

  const preparePerPayPayment = useCallback(async (topupAmount: number) => {
    try {
      setProcessing(true)
      const response = await requestPerPayPayment({
        amount: Math.floor(topupAmount),
      })
      if (!isApiSuccess(response)) {
        toast.error(
          getApiErrorMessage(response) || i18next.t('Payment request failed')
        )
        return 0
      }

      const checkoutUrl = response.data?.checkout_url
      const tradeNo = response.data?.trade_no?.trim()
      const payableAmountCents = response.data?.payable_amount_cents
      if (
        !checkoutUrl ||
        !isSafePerPayCheckoutUrl(checkoutUrl) ||
        !tradeNo ||
        typeof payableAmountCents !== 'number' ||
        !Number.isInteger(payableAmountCents) ||
        payableAmountCents <= 0
      ) {
        toast.error(i18next.t('Payment request failed'))
        return 0
      }

      const preparedPayment = {
        topupAmount: Math.floor(topupAmount),
        checkoutUrl,
        tradeNo,
        payableAmount: payableAmountCents / 100,
      }
      setPreparedPerPayPayment(preparedPayment)
      return preparedPayment.payableAmount
    } catch {
      toast.error(i18next.t('Payment request failed'))
      return 0
    } finally {
      setProcessing(false)
    }
  }, [])

  // Calculate payment amount
  const calculatePaymentAmount = useCallback(
    async (topupAmount: number, paymentType: string) => {
      try {
        setCalculating(true)
        const calculatedAmount = await requestPaymentAmount(
          topupAmount,
          paymentType
        )
        setAmount(calculatedAmount)
        return calculatedAmount
      } catch (error) {
        setAmount(0)
        toast.error(
          error instanceof Error && error.message
            ? error.message
            : i18next.t('Payment request failed')
        )
        return 0
      } finally {
        setCalculating(false)
      }
    },
    []
  )

  // Process payment
  const processPayment = useCallback(
    async (topupAmount: number, paymentType: string) => {
      let redirectWindow: Window | null = null
      try {
        setProcessing(true)

        const isPerPay = isPerPayPayment(paymentType)
        if (isPerPay) {
          const preparedPayment =
            preparedPerPayPayment?.topupAmount === Math.floor(topupAmount)
              ? preparedPerPayPayment
              : null
          // Open synchronously while the confirmation click is still trusted by
          // the browser. The checkout URL arrives after the API request.
          redirectWindow = window.open('', '_blank')
          let checkoutUrl = preparedPayment?.checkoutUrl
          let tradeNo = preparedPayment?.tradeNo
          if (!checkoutUrl || !tradeNo) {
            const response = await requestPerPayPayment({
              amount: Math.floor(topupAmount),
            })
            if (!isApiSuccess(response)) {
              redirectWindow?.close()
              toast.error(
                getApiErrorMessage(response) ||
                  i18next.t('Payment request failed')
              )
              return false
            }
            checkoutUrl = response.data?.checkout_url
            tradeNo = response.data?.trade_no?.trim()
          }
          if (!checkoutUrl || !isSafePerPayCheckoutUrl(checkoutUrl)) {
            redirectWindow?.close()
            toast.error(i18next.t('Invalid payment redirect URL'))
            return false
          }
          if (tradeNo) {
            setPendingPerPayTradeNo(tradeNo)
          }
          toast.success(i18next.t('Redirecting to payment page...'))
          if (redirectWindow) {
            redirectWindow.location.href = checkoutUrl
          } else {
            window.open(checkoutUrl, '_blank')
          }
          return true
        }

        const isStripe = isStripePayment(paymentType)
        const amount = Math.floor(topupAmount)
        if (isStripe) {
          // Keep the new tab tied to the confirmation click while the payment
          // session is being created.
          redirectWindow = window.open('', '_blank')
        }

        const response = isStripe
          ? await requestStripePayment({
              amount,
              payment_method: 'stripe',
            })
          : await requestPayment({
              amount,
              payment_method: paymentType,
            })

        if (!isApiSuccess(response)) {
          redirectWindow?.close()
          toast.error(
            getApiErrorMessage(response) || i18next.t('Payment request failed')
          )
          return false
        }

        // Handle Stripe payment
        const payLink =
          response.data &&
          typeof response.data === 'object' &&
          'pay_link' in response.data
            ? response.data.pay_link
            : null
        if (isStripe && typeof payLink === 'string' && payLink) {
          if (redirectWindow) {
            redirectWindow.location.href = payLink
          } else {
            window.open(payLink, '_blank')
          }
          toast.success(i18next.t('Redirecting to payment page...'))
          return true
        }

        redirectWindow?.close()

        // Handle non-Stripe payment
        if (!isStripe && response.data) {
          const url = (response as unknown as { url?: string }).url
          if (url) {
            submitPaymentForm(url, response.data)
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        return false
      } catch {
        redirectWindow?.close()
        toast.error(i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    [preparedPerPayPayment]
  )

  return {
    amount,
    calculating,
    processing,
    calculatePaymentAmount,
    preparePerPayPayment,
    clearPreparedPerPayPayment,
    processPayment,
    setAmount,
    pendingPerPayTradeNo,
    clearPendingPerPayTradeNo,
    actualPaymentAmount: preparedPerPayPayment?.payableAmount,
  }
}

function getApiErrorMessage(response: ApiResponse<unknown>): string | null {
  const message =
    typeof response.message === 'string' ? response.message.trim() : ''
  if (message && message.toLowerCase() !== 'error') return message

  if (typeof response.data === 'string' && response.data.trim()) {
    return response.data.trim()
  }

  if (
    response.data &&
    typeof response.data === 'object' &&
    'message' in response.data
  ) {
    const dataMessage = (response.data as { message?: unknown }).message
    if (typeof dataMessage === 'string' && dataMessage.trim()) {
      return dataMessage.trim()
    }
  }

  return message || null
}

function isSafePerPayCheckoutUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'https:' && !url.username && !url.password
  } catch {
    return false
  }
}
