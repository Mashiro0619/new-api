package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestRechargePerPayCreditsQuotaExactlyOnce(t *testing.T) {
	truncateTables(t)
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 601, 0)
	order := createEpayTestOrder(t, user.Id, "PERPAYTESTONCE", PaymentProviderPerPay, common.TopUpStatusPending)
	order.PaymentMethod = PaymentMethodPerPay
	order.Money = 10.25
	require.NoError(t, DB.Save(&order).Error)

	alreadyDone, err := RechargePerPay(order.TradeNo, 1025, "127.0.0.1")
	require.NoError(t, err)
	require.False(t, alreadyDone)
	require.Equal(t, 2*500000, getUserQuotaForPaymentGuardTest(t, user.Id))
	require.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, order.TradeNo))

	alreadyDone, err = RechargePerPay(order.TradeNo, 1025, "127.0.0.1")
	require.NoError(t, err)
	require.True(t, alreadyDone)
	require.Equal(t, 2*500000, getUserQuotaForPaymentGuardTest(t, user.Id))
}

func TestRechargePerPayRejectsProviderAndAmountMismatch(t *testing.T) {
	testCases := []struct {
		name          string
		provider      string
		requestedCents int64
		wantError     error
	}{
		{name: "foreign provider", provider: PaymentProviderStripe, requestedCents: 1000, wantError: ErrPaymentMethodMismatch},
		{name: "wrong amount", provider: PaymentProviderPerPay, requestedCents: 1001, wantError: ErrTopUpAmountMismatch},
	}

	for index, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			user := insertUserForPaymentGuardTest(t, 610+index, 7)
			order := createEpayTestOrder(t, user.Id, "PERPAYGUARD"+tc.name, tc.provider, common.TopUpStatusPending)
			_, err := RechargePerPay(order.TradeNo, tc.requestedCents, "127.0.0.1")
			require.ErrorIs(t, err, tc.wantError)
			require.Equal(t, 7, getUserQuotaForPaymentGuardTest(t, user.Id))
			require.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, order.TradeNo))
		})
	}
}
