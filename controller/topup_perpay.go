package controller

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const perPayWebhookBodyLimit = 128 * 1024

var errPerPayConfirmedEventInvalid = errors.New("付款确认事件的状态或金额字段无效")

func RequestPerPay(c *gin.Context) {
	if !isPerPayTopUpEnabled() {
		common.ApiErrorMsg(c, "PerPay 支付尚未配置完成")
		return
	}

	var req AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.Amount < getMinTopup() {
		common.ApiErrorMsg(c, fmt.Sprintf("充值数量不能小于 %d", getMinTopup()))
		return
	}
	userID := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, userID, req.Amount) {
		return
	}
	group, err := model.GetUserGroup(userID, true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	username, err := model.GetUsernameById(userID, false)
	if err != nil || strings.TrimSpace(username) == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PerPay 获取用户名失败 user_id=%d error=%q", userID, err))
		common.ApiErrorMsg(c, "获取用户信息失败")
		return
	}

	payMoney := decimal.NewFromFloat(getPayMoney(req.Amount, group)).Round(2)
	amountCents := payMoney.Mul(decimal.NewFromInt(100)).IntPart()
	if amountCents < 1 || amountCents > 9_999_999_998 {
		common.ApiErrorMsg(c, "充值金额超出 PerPay 支持范围")
		return
	}
	storedMoney, _ := payMoney.Float64()
	notifyURL, err := perPayNotifyURL(system_setting.ServerAddress)
	if err != nil {
		logger.LogError(c.Request.Context(), "PerPay 通知地址无效: "+err.Error())
		common.ApiErrorMsg(c, "系统公开地址必须配置为 HTTPS 站点地址")
		return
	}
	returnURL, err := perPayReturnURL(system_setting.ServerAddress)
	if err != nil {
		logger.LogError(c.Request.Context(), "PerPay 返回地址无效: "+err.Error())
		common.ApiErrorMsg(c, "系统公开地址必须配置为 HTTPS 站点地址")
		return
	}

	tradeNo := fmt.Sprintf("PPUSR%dNO%s%d", userID, common.GetRandomString(8), time.Now().Unix())
	creditedAmount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		creditedAmount = decimal.NewFromInt(req.Amount).
			Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	topUp := &model.TopUp{
		UserId:          userID,
		Amount:          creditedAmount,
		Money:           storedMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodPerPay,
		PaymentProvider: model.PaymentProviderPerPay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PerPay 创建本地充值订单失败 user_id=%d trade_no=%s error=%q", userID, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建充值订单失败")
		return
	}

	client, err := service.NewPerPayClient(
		operation_setting.PerPayAddress,
		operation_setting.PerPayClientId,
		operation_setting.PerPayAPIKey,
		nil,
	)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PerPay 客户端初始化失败 trade_no=%s error=%q", tradeNo, err.Error()))
		common.ApiErrorMsg(c, "PerPay 支付配置无效")
		return
	}
	order, err := client.CreateOrder(c.Request.Context(), service.PerPayCreateOrderInput{
		IdempotencyKey:  "new-api:" + tradeNo,
		MerchantOrderNo: tradeNo,
		AmountCents:      amountCents,
		ProductName:      "中转站充值 " + payMoney.StringFixed(2),
		Note:             "用户名：" + username,
		NotifyURL:        notifyURL,
		ReturnURL:        returnURL,
	})
	if err != nil {
		// The remote result can be unknown after a network failure. Keep the
		// local order pending so a later signed notification can still settle it.
		logger.LogError(c.Request.Context(), fmt.Sprintf("PerPay 创建收银台失败 user_id=%d trade_no=%s error=%q", userID, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建 PerPay 收银台失败，请稍后重试")
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("PerPay 充值订单创建成功 user_id=%d trade_no=%s amount_cents=%d payable_amount_cents=%d", userID, tradeNo, amountCents, order.PayableAmountCents))
	common.ApiSuccess(c, gin.H{
		"checkout_url": order.CheckoutURL,
		"trade_no":     tradeNo,
		"payable_amount_cents": order.PayableAmountCents,
	})
}

func PerPayNotify(c *gin.Context) {
	if !isPerPayWebhookConfigured() {
		logger.LogWarn(c.Request.Context(), "PerPay Webhook 因通知配置不完整被拒绝")
		c.Status(http.StatusServiceUnavailable)
		return
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		c.Status(http.StatusUnsupportedMediaType)
		return
	}
	authentication, err := perPayWebhookAuthentication(c.Request.Header)
	if err != nil {
		logger.LogWarn(c.Request.Context(), "PerPay Webhook 请求头无效: "+err.Error())
		c.Status(http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, perPayWebhookBodyLimit+1))
	if err != nil || len(body) == 0 || len(body) > perPayWebhookBodyLimit {
		c.Status(http.StatusRequestEntityTooLarge)
		return
	}
	if err := service.VerifyPerPayWebhook(operation_setting.PerPayWebhookSecret, authentication, body, time.Now()); err != nil {
		logger.LogWarn(c.Request.Context(), "PerPay Webhook 验签失败: "+err.Error())
		c.Status(http.StatusUnauthorized)
		return
	}
	event, err := service.ParsePerPayWebhookEvent(body)
	if err != nil || event.EventID != authentication.EventID {
		logger.LogWarn(c.Request.Context(), "PerPay Webhook 事件解析失败或事件 ID 不一致")
		c.Status(http.StatusBadRequest)
		return
	}

	switch event.EventType {
	case "PAYMENT_CONFIRMED":
		if err := settlePerPayConfirmedEvent(c, event); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("PerPay Webhook 入账失败 event_id=%s trade_no=%s error=%q", event.EventID, event.MerchantOrderNo, err.Error()))
			c.Status(perPaySettlementErrorStatus(err))
			return
		}
	case "PAYMENT_DISPUTED", "REFUND_UPDATED":
		recordPerPayFollowUpEvent(c, event)
	default:
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("收到尚未处理的 PerPay 事件 event_id=%s event_type=%s", event.EventID, event.EventType))
	}

	writePerPayWebhookAck(c, event.EventID, authentication.DeliveryID)
}

func settlePerPayConfirmedEvent(c *gin.Context, event *service.PerPayWebhookEvent) error {
	if event.MerchantOrderNo == "" || event.PaymentStatus != "CONFIRMED" || event.Currency != "CNY" ||
		(event.PaymentBasis != "INFERRED" && event.PaymentBasis != "MANUAL") ||
		event.RequestedAmountCents <= 0 || event.PayableAmountCents <= 0 ||
		event.ReceivedAmountCents != event.PayableAmountCents {
		return errPerPayConfirmedEventInvalid
	}
	topUp := model.GetTopUpByTradeNo(event.MerchantOrderNo)
	if topUp == nil {
		return model.ErrTopUpNotFound
	}
	if topUp.PaymentProvider != model.PaymentProviderPerPay {
		return model.ErrPaymentMethodMismatch
	}
	expectedCents := decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
	if expectedCents != event.RequestedAmountCents {
		return model.ErrTopUpAmountMismatch
	}
	alreadyDone, err := model.RechargePerPay(event.MerchantOrderNo, event.RequestedAmountCents, c.ClientIP())
	if err != nil {
		return err
	}
	if alreadyDone {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("PerPay 重复付款通知已幂等确认 event_id=%s trade_no=%s", event.EventID, event.MerchantOrderNo))
	}
	return nil
}

func perPaySettlementErrorStatus(err error) int {
	if errors.Is(err, errPerPayConfirmedEventInvalid) ||
		errors.Is(err, model.ErrTopUpNotFound) ||
		errors.Is(err, model.ErrPaymentMethodMismatch) ||
		errors.Is(err, model.ErrTopUpStatusInvalid) ||
		errors.Is(err, model.ErrTopUpAmountMismatch) ||
		errors.Is(err, model.ErrInvalidTopUpQuota) ||
		errors.Is(err, model.ErrTopUpQuotaLimitExceeded) {
		return http.StatusConflict
	}
	return http.StatusServiceUnavailable
}

func recordPerPayFollowUpEvent(c *gin.Context, event *service.PerPayWebhookEvent) {
	topUp := model.GetTopUpByTradeNo(event.MerchantOrderNo)
	if topUp == nil || topUp.PaymentProvider != model.PaymentProviderPerPay {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PerPay 后续事件未找到对应订单 event_id=%s event_type=%s trade_no=%s", event.EventID, event.EventType, event.MerchantOrderNo))
		return
	}
	message := fmt.Sprintf("PerPay 订单收到 %s 事件，未自动扣减已入账额度，event_id=%s", event.EventType, event.EventID)
	logger.LogWarn(c.Request.Context(), fmt.Sprintf("%s trade_no=%s", message, event.MerchantOrderNo))
	model.RecordTopupLog(topUp.UserId, message, c.ClientIP(), model.PaymentMethodPerPay, model.PaymentProviderPerPay)
}

func perPayNotifyURL(serverAddress string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(serverAddress), "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("ServerAddress 必须是 HTTPS 站点地址")
	}
	return trimmed + "/api/user/perpay/notify", nil
}

func perPayReturnURL(serverAddress string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(serverAddress), "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("ServerAddress 必须是 HTTPS 站点地址")
	}
	return trimmed + "/wallet", nil
}

func perPayWebhookAuthentication(headers http.Header) (service.PerPayWebhookAuthentication, error) {
	read := func(name string) (string, error) {
		values := headers.Values(name)
		if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
			return "", fmt.Errorf("%s 缺失或重复", name)
		}
		return values[0], nil
	}
	version, err := read("X-PerPay-Webhook-Version")
	if err != nil {
		return service.PerPayWebhookAuthentication{}, err
	}
	keyID, err := read("X-PerPay-Webhook-Key-Id")
	if err != nil {
		return service.PerPayWebhookAuthentication{}, err
	}
	timestamp, err := read("X-PerPay-Webhook-Timestamp")
	if err != nil {
		return service.PerPayWebhookAuthentication{}, err
	}
	deliveryID, err := read("X-PerPay-Webhook-Delivery-Id")
	if err != nil {
		return service.PerPayWebhookAuthentication{}, err
	}
	eventID, err := read("X-PerPay-Webhook-Event-Id")
	if err != nil {
		return service.PerPayWebhookAuthentication{}, err
	}
	attempt, err := read("X-PerPay-Webhook-Attempt")
	if err != nil {
		return service.PerPayWebhookAuthentication{}, err
	}
	signature, err := read("X-PerPay-Webhook-Signature")
	if err != nil {
		return service.PerPayWebhookAuthentication{}, err
	}
	return service.PerPayWebhookAuthentication{
		Version: version, KeyID: keyID, Timestamp: timestamp, DeliveryID: deliveryID,
		EventID: eventID, Attempt: attempt, Signature: signature,
	}, nil
}

func writePerPayWebhookAck(c *gin.Context, eventID, deliveryID string) {
	body, _ := common.Marshal(gin.H{
		"schema":      "perpay:webhook-ack:v1",
		"ack":         true,
		"event_id":    eventID,
		"delivery_id": deliveryID,
	})
	c.Data(http.StatusOK, "application/json", body)
}
