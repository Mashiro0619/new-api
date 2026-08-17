package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const (
	maxHourlyTokenTrendRange = int64(31 * 24 * 60 * 60)
	maxDailyTokenTrendRange  = int64(366 * 24 * 60 * 60)
)

func parseOptionalPositiveInt(c *gin.Context, key string) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		common.ApiErrorMsg(c, "invalid "+key)
		return 0, false
	}
	return value, true
}

func parseOptionalNonNegativeInt(c *gin.Context, key string) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		common.ApiErrorMsg(c, "invalid "+key)
		return 0, false
	}
	return value, true
}

func parseTokenTrendTimezoneOffset(c *gin.Context) (int, bool) {
	raw := c.Query("timezone_offset")
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < model.MinTokenTrendTimezoneOffset || value > model.MaxTokenTrendTimezoneOffset {
		common.ApiErrorMsg(c, "invalid timezone_offset")
		return 0, false
	}
	return value, true
}

func parseFlowQuotaTimeRange(c *gin.Context) (int64, int64, bool) {
	startTimestamp, err := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	if err != nil || startTimestamp <= 0 {
		common.ApiErrorMsg(c, "invalid start_timestamp")
		return 0, 0, false
	}
	endTimestamp, err := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if err != nil || endTimestamp <= 0 {
		common.ApiErrorMsg(c, "invalid end_timestamp")
		return 0, 0, false
	}
	if endTimestamp < startTimestamp {
		common.ApiErrorMsg(c, "invalid time range")
		return 0, 0, false
	}
	return startTimestamp, endTimestamp, true
}

func GetAllQuotaDates(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	userID, ok := parseOptionalPositiveInt(c, "user_id")
	if !ok {
		return
	}
	if userID > 0 && username != "" {
		common.ApiErrorMsg(c, "user_id and username cannot be used together")
		return
	}
	channelID, ok := parseOptionalNonNegativeInt(c, "channel")
	if !ok {
		return
	}
	dates, err := model.GetQuotaDatesByFilter(model.QuotaDataQuery{
		StartTime: startTimestamp,
		EndTime:   endTimestamp,
		UserID:    userID,
		Username:  username,
		ModelName: c.Query("model_name"),
		TokenName: c.Query("token_name"),
		UseGroup:  c.Query("group"),
		ChannelID: channelID,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func parseTokenTrendQuery(c *gin.Context, self bool) (model.TokenTrendQuery, bool) {
	query := model.TokenTrendQuery{}
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return query, false
	}
	query.StartTimestamp = startTimestamp
	query.EndTimestamp = endTimestamp
	query.Granularity = c.Query("granularity")
	timezoneOffset, ok := parseTokenTrendTimezoneOffset(c)
	if !ok {
		return query, false
	}
	query.TimezoneOffset = timezoneOffset

	var maxRange int64
	switch query.Granularity {
	case model.TokenTrendGranularityHour:
		maxRange = maxHourlyTokenTrendRange
	case model.TokenTrendGranularityDay:
		maxRange = maxDailyTokenTrendRange
	default:
		common.ApiErrorMsg(c, "invalid granularity")
		return query, false
	}
	if endTimestamp-startTimestamp > maxRange {
		common.ApiErrorMsg(c, "time range exceeds limit for "+query.Granularity+" granularity")
		return query, false
	}

	if self {
		if c.Query("user_id") != "" || c.Query("username") != "" {
			common.ApiErrorMsg(c, "identity filters are not allowed on self endpoint")
			return query, false
		}
		query.UserID = c.GetInt("id")
	} else {
		userID, valid := parseOptionalPositiveInt(c, "user_id")
		if !valid {
			return query, false
		}
		query.UserID = userID
		query.Username = c.Query("username")
		if query.UserID > 0 && query.Username != "" {
			common.ApiErrorMsg(c, "user_id and username cannot be used together")
			return query, false
		}
	}

	channelID, valid := parseOptionalNonNegativeInt(c, "channel")
	if !valid {
		return query, false
	}
	query.ChannelID = channelID
	query.ModelName = c.Query("model_name")
	query.TokenName = c.Query("token_name")
	query.UseGroup = c.Query("group")
	return query, true
}

func getTokenTrend(c *gin.Context, self bool) {
	query, ok := parseTokenTrendQuery(c, self)
	if !ok {
		return
	}
	if !common.LogConsumeEnabled || !common.DataExportEnabled {
		reason := "consume_logging_disabled"
		if common.LogConsumeEnabled {
			reason = "data_export_disabled"
		}
		common.ApiSuccess(c, model.TokenTrendData{
			Available:      false,
			Reason:         reason,
			StartTimestamp: query.StartTimestamp,
			EndTimestamp:   query.EndTimestamp,
			Granularity:    query.Granularity,
			Points:         make([]model.TokenTrendPoint, 0),
		})
		return
	}

	data, err := model.GetTokenTrend(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetAllTokenTrend(c *gin.Context) {
	getTokenTrend(c, false)
}

func GetUserTokenTrend(c *gin.Context) {
	getTokenTrend(c, true)
}

func GetTokenTrendUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		common.ApiErrorMsg(c, "invalid user_id")
		return
	}

	user, err := model.GetTokenTrendUser(userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, user)
}

func GetQuotaDatesByUser(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	dates, err := model.GetQuotaDataGroupByUser(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetUserQuotaDates(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 判断时间跨度是否超过 1 个月
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	dates, err := model.GetQuotaDataByUserId(userId, startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetAllFlowQuotaDates(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	username := c.Query("username")
	dates, err := model.GetFlowQuotaData(startTimestamp, endTimestamp, username, 0, c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetUserFlowQuotaDates(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	dates, err := model.GetFlowQuotaData(startTimestamp, endTimestamp, "", userId, common.RoleCommonUser)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}
