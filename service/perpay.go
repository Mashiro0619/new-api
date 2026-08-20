package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	perPayCreateOrderPath       = "/api/v1/orders"
	perPaySignatureVersion      = "v1"
	perPaySignatureDomain       = "PERPAY-HMAC-SHA256"
	perPayWebhookDomain         = "perpay:webhook:v1"
	perPayWebhookMaxSkew        = 5 * time.Minute
	perPayMaximumResponseBytes  = 1024 * 1024
	perPayMaximumWebhookBytes   = 128 * 1024
	perPayHTTPTimeout           = 15 * time.Second
)

var (
	perPayUUIDPattern      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	perPaySignaturePattern = regexp.MustCompile(`^v1=([0-9a-f]{64})$`)
)

type PerPayClient struct {
	baseURL    string
	clientID   string
	apiSecret  []byte
	httpClient *http.Client
}

type PerPayCreateOrderInput struct {
	IdempotencyKey string `json:"idempotency_key"`
	MerchantOrderNo string `json:"merchant_order_no"`
	AmountCents     int64  `json:"amount_cents"`
	ProductName     string `json:"product_name"`
	NotifyURL       string `json:"notify_url,omitempty"`
}

type PerPayCreatedOrder struct {
	OrderID             string
	MerchantOrderNo      string
	RequestedAmountCents int64
	PayableAmountCents   int64
	CheckoutURL          string
}

type perPayOrderEnvelope struct {
	Data struct {
		OrderID             string `json:"order_id"`
		MerchantOrderNo      string `json:"merchant_order_no"`
		RequestedAmountCents int64  `json:"requested_amount_cents"`
		PayableAmountCents   int64  `json:"payable_amount_cents"`
		Currency             string `json:"currency"`
		Checkout             struct {
			Status      string `json:"status"`
			CheckoutURL string `json:"checkout_url"`
		} `json:"checkout"`
	} `json:"data"`
}

type perPayErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

func NewPerPayClient(address, clientID, apiKey string, httpClient *http.Client) (*PerPayClient, error) {
	if err := operation_setting.ValidatePerPayAddress(address); err != nil || address == "" {
		if err == nil {
			err = errors.New("PerPay 地址未配置")
		}
		return nil, err
	}
	if err := operation_setting.ValidatePerPayClientId(clientID); err != nil || clientID == "" {
		if err == nil {
			err = errors.New("PerPay Client ID 未配置")
		}
		return nil, err
	}
	secret, err := operation_setting.DecodePerPaySecret(apiKey)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: perPayHTTPTimeout}
	}
	clientCopy := *httpClient
	if clientCopy.Timeout <= 0 {
		clientCopy.Timeout = perPayHTTPTimeout
	}
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &PerPayClient{
		baseURL:    strings.TrimRight(address, "/"),
		clientID:   clientID,
		apiSecret:  secret,
		httpClient: &clientCopy,
	}, nil
}

func (client *PerPayClient) CreateOrder(ctx context.Context, input PerPayCreateOrderInput) (*PerPayCreatedOrder, error) {
	if client == nil {
		return nil, errors.New("PerPay 客户端未初始化")
	}
	if input.IdempotencyKey == "" || input.MerchantOrderNo == "" || input.AmountCents <= 0 || strings.TrimSpace(input.ProductName) == "" {
		return nil, errors.New("PerPay 订单参数无效")
	}
	if err := validatePerPayHTTPSURL(input.NotifyURL); err != nil {
		return nil, fmt.Errorf("PerPay 通知地址无效: %w", err)
	}
	body, err := common.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("序列化 PerPay 订单失败: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("生成 PerPay 请求随机数失败: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	signature := perPayAPIRequestSignature(client.apiSecret, http.MethodPost, perPayCreateOrderPath, timestamp, nonce, client.clientID, body)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+perPayCreateOrderPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 PerPay 请求失败: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-PerPay-Client-Id", client.clientID)
	request.Header.Set("X-PerPay-Timestamp", timestamp)
	request.Header.Set("X-PerPay-Nonce", nonce)
	request.Header.Set("X-PerPay-Signature-Version", perPaySignatureVersion)
	request.Header.Set("X-PerPay-Signature", signature)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("PerPay 请求失败: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, perPayMaximumResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取 PerPay 响应失败: %w", err)
	}
	if len(responseBody) > perPayMaximumResponseBytes {
		return nil, errors.New("PerPay 响应过大")
	}
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		var envelope perPayErrorEnvelope
		_ = common.Unmarshal(responseBody, &envelope)
		message := strings.TrimSpace(envelope.Error.Message)
		if message == "" {
			message = strings.TrimSpace(envelope.Message)
		}
		if message == "" {
			message = http.StatusText(response.StatusCode)
		}
		return nil, fmt.Errorf("PerPay 返回 HTTP %d: %s", response.StatusCode, message)
	}

	var envelope perPayOrderEnvelope
	if err := common.Unmarshal(responseBody, &envelope); err != nil {
		return nil, fmt.Errorf("解析 PerPay 响应失败: %w", err)
	}
	order := envelope.Data
	if !perPayUUIDPattern.MatchString(order.OrderID) || order.MerchantOrderNo != input.MerchantOrderNo ||
		order.RequestedAmountCents != input.AmountCents || order.PayableAmountCents <= input.AmountCents ||
		order.PayableAmountCents > 9_999_999_999 || order.PayableAmountCents-input.AmountCents > 99 ||
		order.Currency != "CNY" || order.Checkout.Status != "OPEN" {
		return nil, errors.New("PerPay 响应与本地订单不一致")
	}
	if err := validatePerPayCheckoutURL(order.Checkout.CheckoutURL, client.baseURL); err != nil {
		return nil, errors.New("PerPay 返回了无效的收银台地址")
	}
	return &PerPayCreatedOrder{
		OrderID:             order.OrderID,
		MerchantOrderNo:      order.MerchantOrderNo,
		RequestedAmountCents: order.RequestedAmountCents,
		PayableAmountCents:   order.PayableAmountCents,
		CheckoutURL:          order.Checkout.CheckoutURL,
	}, nil
}

func perPayAPIRequestSignature(secret []byte, method, target, timestamp, nonce, clientID string, body []byte) string {
	bodyHash := sha256.Sum256(body)
	canonical := strings.Join([]string{
		perPaySignatureDomain,
		perPaySignatureVersion,
		strings.ToUpper(method),
		target,
		timestamp,
		nonce,
		clientID,
		hex.EncodeToString(bodyHash[:]),
	}, "\n")
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

type PerPayWebhookAuthentication struct {
	Version      string
	KeyID        string
	Timestamp    string
	DeliveryID   string
	EventID      string
	Attempt      string
	Signature    string
}

type PerPayWebhookEvent struct {
	Schema               string          `json:"schema"`
	EventID              string          `json:"event_id"`
	EventType            string          `json:"event_type"`
	OrderID              string          `json:"order_id"`
	OrderVersion         int64           `json:"order_version"`
	MerchantOrderNo      string          `json:"merchant_order_no"`
	RequestedAmountCents int64           `json:"requested_amount_cents"`
	PayableAmountCents   int64           `json:"payable_amount_cents"`
	ReceivedAmountCents  int64           `json:"received_amount_cents"`
	Currency             string          `json:"currency"`
	PaymentStatus        string          `json:"payment_status"`
	PaymentBasis         string          `json:"payment_basis"`
	RefundStatus         string          `json:"refund_status"`
	EventDetails         json.RawMessage `json:"event_details"`
	OccurredAt           int64           `json:"occurred_at"`
}

func VerifyPerPayWebhook(secret string, authentication PerPayWebhookAuthentication, body []byte, now time.Time) error {
	secretBytes, err := operation_setting.DecodePerPaySecret(secret)
	if err != nil {
		return err
	}
	if authentication.Version != "1" {
		return errors.New("不支持的 PerPay Webhook 版本")
	}
	for label, value := range map[string]string{
		"key_id": authentication.KeyID,
		"delivery_id": authentication.DeliveryID,
		"event_id": authentication.EventID,
	} {
		if !perPayUUIDPattern.MatchString(value) {
			return fmt.Errorf("PerPay Webhook %s 无效", label)
		}
	}
	timestamp, err := strconv.ParseInt(authentication.Timestamp, 10, 64)
	if err != nil || timestamp < 0 || strconv.FormatInt(timestamp, 10) != authentication.Timestamp {
		return errors.New("PerPay Webhook 时间戳无效")
	}
	if delta := now.UnixMilli() - timestamp; delta > perPayWebhookMaxSkew.Milliseconds() || delta < -perPayWebhookMaxSkew.Milliseconds() {
		return errors.New("PerPay Webhook 时间戳已过期")
	}
	attempt, err := strconv.ParseInt(authentication.Attempt, 10, 64)
	if err != nil || attempt < 1 || strconv.FormatInt(attempt, 10) != authentication.Attempt {
		return errors.New("PerPay Webhook 尝试次数无效")
	}
	match := perPaySignaturePattern.FindStringSubmatch(authentication.Signature)
	if len(match) != 2 {
		return errors.New("PerPay Webhook 签名格式无效")
	}
	bodyHash := sha256.Sum256(body)
	canonical := strings.Join([]string{
		perPayWebhookDomain,
		authentication.KeyID,
		authentication.Timestamp,
		authentication.DeliveryID,
		authentication.EventID,
		authentication.Attempt,
		hex.EncodeToString(bodyHash[:]),
	}, "\n")
	mac := hmac.New(sha256.New, secretBytes)
	_, _ = mac.Write([]byte(canonical))
	expected := mac.Sum(nil)
	supplied, _ := hex.DecodeString(match[1])
	if !hmac.Equal(expected, supplied) {
		return errors.New("PerPay Webhook 签名无效")
	}
	return nil
}

func ParsePerPayWebhookEvent(body []byte) (*PerPayWebhookEvent, error) {
	if len(body) == 0 || len(body) > perPayMaximumWebhookBytes {
		return nil, errors.New("PerPay Webhook 请求体大小无效")
	}
	var event PerPayWebhookEvent
	if err := common.Unmarshal(body, &event); err != nil {
		return nil, errors.New("PerPay Webhook JSON 无效")
	}
	if event.Schema != "perpay:outbox-event:v2" || !perPayUUIDPattern.MatchString(event.EventID) ||
		!perPayUUIDPattern.MatchString(event.OrderID) || event.OrderVersion < 1 {
		return nil, errors.New("PerPay Webhook 事件结构无效")
	}
	return &event, nil
}

func validatePerPayHTTPSURL(value string) error {
	if value == "" || value != strings.TrimSpace(value) {
		return errors.New("地址为空或包含首尾空格")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("地址必须是无用户信息和片段的绝对 HTTPS URL")
	}
	return nil
}

func validatePerPayCheckoutURL(value, baseURL string) error {
	if err := validatePerPayHTTPSURL(value); err != nil {
		return err
	}
	checkout, checkoutErr := url.Parse(value)
	base, baseErr := url.Parse(baseURL)
	if checkoutErr != nil || baseErr != nil || !strings.EqualFold(checkout.Scheme, base.Scheme) ||
		!strings.EqualFold(checkout.Host, base.Host) {
		return errors.New("收银台地址与 PerPay 站点不同源")
	}
	return nil
}
