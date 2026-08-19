package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func perPayTestSecret(seed byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{seed}, 32))
}

func TestPerPayClientCreateOrderSignsExactBody(t *testing.T) {
	secretText := perPayTestSecret(0x42)
	secret, err := base64.RawURLEncoding.DecodeString(secretText)
	require.NoError(t, err)

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, perPayCreateOrderPath, r.URL.RequestURI())
		body, readErr := io.ReadAll(r.Body)
		require.NoError(t, readErr)
		require.Equal(t, "default", r.Header.Get("X-PerPay-Client-Id"))
		nonce := r.Header.Get("X-PerPay-Nonce")
		require.Len(t, nonce, 43)
		timestamp := r.Header.Get("X-PerPay-Timestamp")
		_, parseErr := strconv.ParseInt(timestamp, 10, 64)
		require.NoError(t, parseErr)
		expected := perPayAPIRequestSignature(secret, http.MethodPost, perPayCreateOrderPath, timestamp, nonce, "default", body)
		require.Equal(t, expected, r.Header.Get("X-PerPay-Signature"))

		var request PerPayCreateOrderInput
		require.NoError(t, json.Unmarshal(body, &request))
		require.Equal(t, int64(1234), request.AmountCents)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"data":{"order_id":"018f4d21-5a1d-4c76-8a1c-4e45b5991bb5","merchant_order_no":%q,"requested_amount_cents":1234,"payable_amount_cents":1235,"currency":"CNY","checkout":{"status":"OPEN","checkout_url":%q}}}`, request.MerchantOrderNo, server.URL+"/checkout/test")
	}))
	defer server.Close()

	client, err := NewPerPayClient(server.URL, "default", secretText, server.Client())
	require.NoError(t, err)
	order, err := client.CreateOrder(t.Context(), PerPayCreateOrderInput{
		IdempotencyKey:  "new-api:trade-1",
		MerchantOrderNo: "trade-1",
		AmountCents:      1234,
		Description:      "new-api recharge",
		NotifyURL:        server.URL + "/api/user/perpay/notify",
	})
	require.NoError(t, err)
	require.Equal(t, "trade-1", order.MerchantOrderNo)
	require.Equal(t, int64(1235), order.PayableAmountCents)
	require.Equal(t, server.URL+"/checkout/test", order.CheckoutURL)
}

func TestVerifyPerPayWebhook(t *testing.T) {
	secretText := perPayTestSecret(0x24)
	secret, err := base64.RawURLEncoding.DecodeString(secretText)
	require.NoError(t, err)
	now := time.UnixMilli(1_750_000_000_000)
	body := []byte(`{"schema":"perpay:outbox-event:v2","event_id":"018f4d21-5a1d-4c76-8a1c-4e45b5991bb5"}`)
	auth := PerPayWebhookAuthentication{
		Version:    "1",
		KeyID:      "128f4d21-5a1d-4c76-8a1c-4e45b5991bb5",
		Timestamp:  strconv.FormatInt(now.UnixMilli(), 10),
		DeliveryID: "228f4d21-5a1d-4c76-8a1c-4e45b5991bb5",
		EventID:    "018f4d21-5a1d-4c76-8a1c-4e45b5991bb5",
		Attempt:    "1",
	}
	bodyHash := sha256.Sum256(body)
	canonical := strings.Join([]string{
		perPayWebhookDomain,
		auth.KeyID,
		auth.Timestamp,
		auth.DeliveryID,
		auth.EventID,
		auth.Attempt,
		hex.EncodeToString(bodyHash[:]),
	}, "\n")
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(canonical))
	auth.Signature = "v1=" + hex.EncodeToString(mac.Sum(nil))

	require.NoError(t, VerifyPerPayWebhook(secretText, auth, body, now))
	require.Error(t, VerifyPerPayWebhook(secretText, auth, append(body, ' '), now))
	require.Error(t, VerifyPerPayWebhook(secretText, auth, body, now.Add(6*time.Minute)))
}

func TestParsePerPayWebhookEvent(t *testing.T) {
	body := []byte(`{"schema":"perpay:outbox-event:v2","event_id":"018f4d21-5a1d-4c76-8a1c-4e45b5991bb5","event_type":"PAYMENT_CONFIRMED","order_id":"128f4d21-5a1d-4c76-8a1c-4e45b5991bb5","order_version":2,"merchant_order_no":"trade-1","requested_amount_cents":1234,"payable_amount_cents":1235,"received_amount_cents":1235,"currency":"CNY","payment_status":"CONFIRMED","payment_basis":"INFERRED","extra":"accepted"}`)
	event, err := ParsePerPayWebhookEvent(body)
	require.NoError(t, err)
	require.Equal(t, "trade-1", event.MerchantOrderNo)
	require.Equal(t, int64(1235), event.ReceivedAmountCents)

	_, err = ParsePerPayWebhookEvent(append(body, []byte(` {}`)...))
	require.Error(t, err)
}
