package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tokenTrendResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Data    model.TokenTrendData `json:"data"`
}

func setupTokenTrendControllerTestDB(t *testing.T) {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.QuotaData{}, &model.Log{}))

	originalLogConsumeEnabled := common.LogConsumeEnabled
	originalDataExportEnabled := common.DataExportEnabled
	common.LogConsumeEnabled = true
	common.DataExportEnabled = true
	t.Cleanup(func() {
		common.LogConsumeEnabled = originalLogConsumeEnabled
		common.DataExportEnabled = originalDataExportEnabled
	})
}

func performTokenTrendRequest(t *testing.T, handler gin.HandlerFunc, target string, userID int) tokenTrendResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	if userID > 0 {
		ctx.Set("id", userID)
	}
	ctx.Request = httptest.NewRequest(http.MethodGet, target, nil)
	handler(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response tokenTrendResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestGetAllTokenTrendUsesAdminUserIDFilter(t *testing.T) {
	setupTokenTrendControllerTestDB(t)
	require.NoError(t, model.DB.Create(&[]model.QuotaData{
		{UserID: 1, Username: "alice", CreatedAt: 3600, InputTokens: 10, TokenMetricsCount: 1},
		{UserID: 2, Username: "bob", CreatedAt: 3600, InputTokens: 99, TokenMetricsCount: 1},
	}).Error)

	response := performTokenTrendRequest(t, GetAllTokenTrend,
		"/api/data/token-trend?start_timestamp=1&end_timestamp=7200&granularity=hour&user_id=1", 0)
	require.True(t, response.Success, response.Message)
	assert.True(t, response.Data.Available)
	assert.Equal(t, int64(10), response.Data.Totals.InputTokens)
	assert.Equal(t, int64(1), response.Data.Totals.TrackedRequests)
}

func TestGetAllTokenTrendAppliesClientTimezoneOffset(t *testing.T) {
	setupTokenTrendControllerTestDB(t)
	location := time.FixedZone("UTC+5:30", 5*60*60+30*60)
	firstHour := time.Date(2026, time.February, 3, 10, 0, 0, 0, location).Unix()
	require.NoError(t, model.DB.Create(&model.QuotaData{
		UserID: 1, Username: "alice", CreatedAt: firstHour + 15*60,
		InputTokens: 10, TokenMetricsCount: 1,
	}).Error)

	response := performTokenTrendRequest(t, GetAllTokenTrend,
		"/api/data/token-trend?start_timestamp="+strconv.FormatInt(firstHour, 10)+
			"&end_timestamp="+strconv.FormatInt(firstHour+30*60, 10)+
			"&granularity=hour&timezone_offset=-330&user_id=1", 0)
	require.True(t, response.Success, response.Message)
	require.Len(t, response.Data.Points, 1)
	assert.Equal(t, firstHour, response.Data.Points[0].Timestamp)
}

func TestGetUserTokenTrendForcesAuthenticatedIdentity(t *testing.T) {
	setupTokenTrendControllerTestDB(t)
	require.NoError(t, model.DB.Create(&[]model.QuotaData{
		{UserID: 1, Username: "alice", CreatedAt: 3600, InputTokens: 10, TokenMetricsCount: 1},
		{UserID: 2, Username: "bob", CreatedAt: 3600, InputTokens: 99, TokenMetricsCount: 1},
	}).Error)

	response := performTokenTrendRequest(t, GetUserTokenTrend,
		"/api/data/token-trend/self?start_timestamp=1&end_timestamp=7200&granularity=hour", 1)
	require.True(t, response.Success, response.Message)
	assert.Equal(t, int64(10), response.Data.Totals.InputTokens)

	rejected := performTokenTrendRequest(t, GetUserTokenTrend,
		"/api/data/token-trend/self?start_timestamp=1&end_timestamp=7200&granularity=hour&user_id=2", 1)
	assert.False(t, rejected.Success)
	assert.Equal(t, "identity filters are not allowed on self endpoint", rejected.Message)
}

func TestGetTokenTrendReportsUnavailableConfiguration(t *testing.T) {
	setupTokenTrendControllerTestDB(t)
	common.DataExportEnabled = false

	response := performTokenTrendRequest(t, GetAllTokenTrend,
		"/api/data/token-trend?start_timestamp=1&end_timestamp=7200&granularity=hour", 0)
	require.True(t, response.Success, response.Message)
	assert.False(t, response.Data.Available)
	assert.Equal(t, "data_export_disabled", response.Data.Reason)
	assert.Empty(t, response.Data.Points)

	common.LogConsumeEnabled = false
	response = performTokenTrendRequest(t, GetAllTokenTrend,
		"/api/data/token-trend?start_timestamp=1&end_timestamp=7200&granularity=hour", 0)
	require.True(t, response.Success, response.Message)
	assert.Equal(t, "consume_logging_disabled", response.Data.Reason)
}

func TestGetTokenTrendUserReturnsDeletedAndHigherRoleUsers(t *testing.T) {
	setupTokenTrendControllerTestDB(t)
	user := model.User{
		Username:     "root-user",
		Password:     "password",
		DisplayName:  "Root User",
		Role:         common.RoleRootUser,
		Status:       common.UserStatusEnabled,
		Group:        "default",
		UsedQuota:    123,
		RequestCount: 7,
	}
	require.NoError(t, model.DB.Create(&user).Error)
	require.NoError(t, model.DB.Delete(&user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/token-trend/users/1", nil)
	GetTokenTrendUser(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool                 `json:"success"`
		Data    model.TokenTrendUser `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, user.Id, response.Data.ID)
	assert.Equal(t, common.RoleRootUser, response.Data.Role)
	assert.True(t, response.Data.DeletedAt.Valid)
	assert.NotContains(t, recorder.Body.String(), "password")
}

func TestGetTokenTrendValidatesRangeAndIdentityFilters(t *testing.T) {
	setupTokenTrendControllerTestDB(t)
	tests := []struct {
		name    string
		target  string
		message string
	}{
		{
			name:    "invalid granularity",
			target:  "/api/data/token-trend?start_timestamp=1&end_timestamp=2&granularity=week",
			message: "invalid granularity",
		},
		{
			name:    "hourly range too long",
			target:  "/api/data/token-trend?start_timestamp=1&end_timestamp=2678402&granularity=hour",
			message: "time range exceeds limit for hour granularity",
		},
		{
			name:    "conflicting identity",
			target:  "/api/data/token-trend?start_timestamp=1&end_timestamp=2&granularity=hour&user_id=1&username=alice",
			message: "user_id and username cannot be used together",
		},
		{
			name:    "timezone offset above maximum",
			target:  "/api/data/token-trend?start_timestamp=1&end_timestamp=2&granularity=hour&timezone_offset=841",
			message: "invalid timezone_offset",
		},
		{
			name:    "timezone offset below minimum",
			target:  "/api/data/token-trend?start_timestamp=1&end_timestamp=2&granularity=hour&timezone_offset=-841",
			message: "invalid timezone_offset",
		},
		{
			name:    "timezone offset is not numeric",
			target:  "/api/data/token-trend?start_timestamp=1&end_timestamp=2&granularity=hour&timezone_offset=local",
			message: "invalid timezone_offset",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performTokenTrendRequest(t, GetAllTokenTrend, test.target, 0)
			assert.False(t, response.Success)
			assert.Equal(t, test.message, response.Message)
		})
	}
}

func TestGetAllQuotaDatesAcceptsUserID(t *testing.T) {
	setupTokenTrendControllerTestDB(t)
	require.NoError(t, model.DB.Create(&[]model.QuotaData{
		{UserID: 1, Username: "old-alice", ModelName: "gpt-a", TokenName: "primary", UseGroup: "vip", ChannelID: 11, CreatedAt: 3600, Count: 1},
		{UserID: 1, Username: "old-alice", ModelName: "gpt-b", TokenName: "secondary", UseGroup: "default", ChannelID: 12, CreatedAt: 3600, Count: 1},
		{UserID: 2, Username: "bob", ModelName: "gpt-a", TokenName: "primary", UseGroup: "vip", ChannelID: 11, CreatedAt: 3600, Count: 1},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data?start_timestamp=1&end_timestamp=7200&user_id=1&model_name=gpt-a&token_name=primary&group=vip&channel=11", nil)
	GetAllQuotaDates(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool              `json:"success"`
		Data    []model.QuotaData `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, 1, response.Data[0].UserID)
}
