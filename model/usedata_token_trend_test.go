package model

import (
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTokenMetrics(t *testing.T) {
	tests := []struct {
		name                  string
		input                 int64
		output                int64
		cacheCreation         int64
		cacheRead             int64
		inputAlreadySeparated bool
		want                  TokenMetrics
	}{
		{
			name:          "standard input excludes both cache classes",
			input:         100,
			output:        20,
			cacheCreation: 10,
			cacheRead:     30,
			want: TokenMetrics{
				InputTokens:         60,
				OutputTokens:        20,
				CacheCreationTokens: 10,
				CacheReadTokens:     30,
			},
		},
		{
			name:                  "anthropic input is already separated",
			input:                 100,
			output:                20,
			cacheCreation:         10,
			cacheRead:             30,
			inputAlreadySeparated: true,
			want: TokenMetrics{
				InputTokens:         100,
				OutputTokens:        20,
				CacheCreationTokens: 10,
				CacheReadTokens:     30,
			},
		},
		{
			name:          "overlapping upstream counts clamp the remainder",
			input:         10,
			output:        -1,
			cacheCreation: 20,
			cacheRead:     30,
			want: TokenMetrics{
				CacheCreationTokens: 20,
				CacheReadTokens:     30,
			},
		},
		{
			name:          "maximum cache counts do not overflow",
			input:         1,
			cacheCreation: math.MaxInt64,
			cacheRead:     math.MaxInt64,
			want: TokenMetrics{
				CacheCreationTokens: math.MaxInt64,
				CacheReadTokens:     math.MaxInt64,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, NormalizeTokenMetrics(
				test.input,
				test.output,
				test.cacheCreation,
				test.cacheRead,
				test.inputAlreadySeparated,
			))
		})
	}
}

func TestQuotaDataHasUserCreatedAtIndex(t *testing.T) {
	assert.True(t, DB.Migrator().HasIndex(&QuotaData{}, "idx_qdt_user_created_at"))
}

func TestGetTokenTrendExcludesLegacyRowsAndFillsMeasuredRange(t *testing.T) {
	truncateTables(t)
	rows := []QuotaData{
		{
			UserID: 1, Username: "alice", ModelName: "legacy", CreatedAt: 3600,
			Count: 10, TokenUsed: 1000, Quota: 5000,
		},
		{
			UserID: 1, Username: "alice", ModelName: "gpt-a", TokenName: "primary",
			UseGroup: "vip", ChannelID: 11, CreatedAt: 7200,
			InputTokens: 100, OutputTokens: 20, CacheCreationTokens: 10,
			CacheReadTokens: 50, TokenMetricsCount: 2, Quota: 120,
		},
		{
			UserID: 1, Username: "alice", ModelName: "gpt-a", TokenName: "primary",
			UseGroup: "vip", ChannelID: 11, CreatedAt: 14400,
			InputTokens: 40, OutputTokens: 10, TokenMetricsCount: 1, Quota: 30,
		},
		{
			UserID: 2, Username: "bob", ModelName: "gpt-a", TokenName: "primary",
			UseGroup: "vip", ChannelID: 11, CreatedAt: 7200,
			InputTokens: 999, OutputTokens: 999, TokenMetricsCount: 1, Quota: 999,
		},
	}
	require.NoError(t, DB.Create(&rows).Error)

	trend, err := GetTokenTrend(TokenTrendQuery{
		StartTimestamp: 3700,
		EndTimestamp:   15000,
		Granularity:    TokenTrendGranularityHour,
		UserID:         1,
		ModelName:      "gpt-a",
		TokenName:      "primary",
		UseGroup:       "vip",
		ChannelID:      11,
	})
	require.NoError(t, err)
	require.True(t, trend.Available)
	require.NotNil(t, trend.TrackingStartedAt)
	assert.Equal(t, int64(7200), *trend.TrackingStartedAt)
	require.Len(t, trend.Points, 3)
	assert.Equal(t, int64(7200), trend.Points[0].Timestamp)
	assert.Equal(t, int64(100), trend.Points[0].InputTokens)
	assert.Equal(t, int64(120), trend.Points[0].ConsumedQuota)
	assert.Equal(t, int64(2), trend.Points[0].TrackedRequests)
	require.NotNil(t, trend.Points[0].CacheHitRate)
	assert.InDelta(t, 100.0/3.0, *trend.Points[0].CacheHitRate, 0.0001)

	assert.Equal(t, int64(10800), trend.Points[1].Timestamp)
	assert.Zero(t, trend.Points[1].TrackedRequests)
	assert.Nil(t, trend.Points[1].CacheHitRate)
	assert.Equal(t, int64(14400), trend.Points[2].Timestamp)
	assert.Equal(t, int64(140), trend.Totals.InputTokens)
	assert.Equal(t, int64(30), trend.Totals.OutputTokens)
	assert.Equal(t, int64(150), trend.Totals.ConsumedQuota)
	assert.Equal(t, int64(10), trend.Totals.CacheCreationTokens)
	assert.Equal(t, int64(50), trend.Totals.CacheReadTokens)
	assert.Equal(t, int64(3), trend.Totals.TrackedRequests)
	require.NotNil(t, trend.Totals.CacheHitRate)
	assert.InDelta(t, 50.0/190.0*100, *trend.Totals.CacheHitRate, 0.0001)
}

func TestGetTokenTrendDailyAggregationAndEmptyLegacyData(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&[]QuotaData{
		{UserID: 1, Username: "alice", CreatedAt: 3600, TokenMetricsCount: 1, InputTokens: 10, Quota: 15},
		{UserID: 1, Username: "alice", CreatedAt: 7200, TokenMetricsCount: 1, InputTokens: 20, Quota: 25},
		{UserID: 1, Username: "alice", CreatedAt: 90000, TokenMetricsCount: 1, InputTokens: 30, Quota: 35},
		{UserID: 2, Username: "legacy", CreatedAt: 3600, Count: 1, TokenUsed: 50, Quota: 5000},
	}).Error)

	trend, err := GetTokenTrend(TokenTrendQuery{
		StartTimestamp: 1,
		EndTimestamp:   100000,
		Granularity:    TokenTrendGranularityDay,
		Username:       "alice",
	})
	require.NoError(t, err)
	require.Len(t, trend.Points, 2)
	assert.Equal(t, int64(30), trend.Points[0].InputTokens)
	assert.Equal(t, int64(40), trend.Points[0].ConsumedQuota)
	assert.Equal(t, int64(30), trend.Points[1].InputTokens)
	assert.Equal(t, int64(35), trend.Points[1].ConsumedQuota)
	assert.Equal(t, int64(75), trend.Totals.ConsumedQuota)

	legacy, err := GetTokenTrend(TokenTrendQuery{
		StartTimestamp: 1,
		EndTimestamp:   100000,
		Granularity:    TokenTrendGranularityDay,
		Username:       "legacy",
	})
	require.NoError(t, err)
	assert.Nil(t, legacy.TrackingStartedAt)
	assert.Empty(t, legacy.Points)
	assert.Zero(t, legacy.Totals.TrackedRequests)
}

func TestGetTokenTrendAlignsDailyBucketsToClientTimezone(t *testing.T) {
	tests := []struct {
		name           string
		timezoneOffset int
		locationOffset int
	}{
		{name: "UTC plus 8", timezoneOffset: -480, locationOffset: 8 * 60 * 60},
		{name: "UTC minus 5", timezoneOffset: 300, locationOffset: -5 * 60 * 60},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			truncateTables(t)
			location := time.FixedZone(test.name, test.locationOffset)
			firstDay := time.Date(2026, time.February, 3, 0, 0, 0, 0, location).Unix()
			secondDay := firstDay + 24*60*60
			require.NoError(t, DB.Create(&[]QuotaData{
				{UserID: 1, Username: "alice", CreatedAt: firstDay - 1, InputTokens: 99, TokenMetricsCount: 1},
				{UserID: 1, Username: "alice", CreatedAt: firstDay + 60, InputTokens: 10, TokenMetricsCount: 1},
				{UserID: 1, Username: "alice", CreatedAt: secondDay + 60, InputTokens: 20, TokenMetricsCount: 1},
			}).Error)

			trend, err := GetTokenTrend(TokenTrendQuery{
				StartTimestamp: firstDay,
				EndTimestamp:   secondDay + 60*60,
				Granularity:    TokenTrendGranularityDay,
				TimezoneOffset: test.timezoneOffset,
				UserID:         1,
			})
			require.NoError(t, err)
			require.Len(t, trend.Points, 2)
			assert.Equal(t, firstDay, trend.Points[0].Timestamp)
			assert.Equal(t, int64(10), trend.Points[0].InputTokens)
			assert.Equal(t, secondDay, trend.Points[1].Timestamp)
			assert.Equal(t, int64(20), trend.Points[1].InputTokens)
			assert.Equal(t, int64(30), trend.Totals.InputTokens)
		})
	}
}

func TestGetTokenTrendAlignsHourlyBucketsToNonWholeHourTimezone(t *testing.T) {
	truncateTables(t)
	location := time.FixedZone("UTC+5:30", 5*60*60+30*60)
	firstHour := time.Date(2026, time.February, 3, 10, 0, 0, 0, location).Unix()
	secondHour := firstHour + 60*60
	require.NoError(t, DB.Create(&[]QuotaData{
		{UserID: 1, Username: "alice", CreatedAt: firstHour + 15*60, InputTokens: 10, TokenMetricsCount: 1},
		{UserID: 1, Username: "alice", CreatedAt: secondHour + 10*60, InputTokens: 20, TokenMetricsCount: 1},
	}).Error)

	trend, err := GetTokenTrend(TokenTrendQuery{
		StartTimestamp: firstHour,
		EndTimestamp:   secondHour + 10*60,
		Granularity:    TokenTrendGranularityHour,
		TimezoneOffset: -330,
		UserID:         1,
	})
	require.NoError(t, err)
	require.Len(t, trend.Points, 2)
	assert.Equal(t, firstHour, trend.Points[0].Timestamp)
	assert.Equal(t, secondHour, trend.Points[1].Timestamp)
}

func TestGetTokenTrendRejectsInvalidTimezoneOffset(t *testing.T) {
	for _, timezoneOffset := range []int{-841, 841} {
		trend, err := GetTokenTrend(TokenTrendQuery{
			StartTimestamp: 1,
			EndTimestamp:   2,
			Granularity:    TokenTrendGranularityHour,
			TimezoneOffset: timezoneOffset,
		})
		require.EqualError(t, err, "invalid timezone_offset")
		assert.Empty(t, trend.Points)
	}
}

func TestGetTokenTrendRejectsTimestampBucketOverflow(t *testing.T) {
	trend, err := GetTokenTrend(TokenTrendQuery{
		StartTimestamp: math.MaxInt64 - 60,
		EndTimestamp:   math.MaxInt64,
		Granularity:    TokenTrendGranularityHour,
	})
	require.EqualError(t, err, "timestamp range cannot be bucketed safely")
	assert.Empty(t, trend.Points)
}

func TestGetTokenTrendTimezoneArithmeticHandlesTimestampExtremes(t *testing.T) {
	truncateTables(t)

	t.Run("small timestamp with western offset", func(t *testing.T) {
		trend, err := GetTokenTrend(TokenTrendQuery{
			StartTimestamp: 1,
			EndTimestamp:   1,
			Granularity:    TokenTrendGranularityHour,
			TimezoneOffset: 840,
		})
		require.NoError(t, err)
		assert.Empty(t, trend.Points)
	})

	t.Run("shifting maximum timestamp overflows", func(t *testing.T) {
		trend, err := GetTokenTrend(TokenTrendQuery{
			StartTimestamp: math.MaxInt64,
			EndTimestamp:   math.MaxInt64,
			Granularity:    TokenTrendGranularityHour,
			TimezoneOffset: -840,
		})
		require.EqualError(t, err, "timestamp range cannot be bucketed safely")
		assert.Empty(t, trend.Points)
	})

	t.Run("exclusive end overflows after eastern shift", func(t *testing.T) {
		trend, err := GetTokenTrend(TokenTrendQuery{
			StartTimestamp: math.MaxInt64,
			EndTimestamp:   math.MaxInt64,
			Granularity:    TokenTrendGranularityHour,
			TimezoneOffset: 840,
		})
		require.EqualError(t, err, "timestamp range cannot be bucketed safely")
		assert.Empty(t, trend.Points)
	})
}

func TestLogQuotaDataPersistsTokenMetrics(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	first := TokenMetrics{InputTokens: 70, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 30}
	second := TokenMetrics{InputTokens: 5, OutputTokens: 3, CacheReadTokens: 2}
	base := QuotaDataLogParams{
		UserID: 1, Username: "alice", ModelName: "gpt-a", TokenID: 7,
		TokenName: "primary", UseGroup: "vip", ChannelID: 11, NodeName: "node-a",
		CreatedAt: 3661, Quota: 10, TokenMetrics: &first,
	}
	LogQuotaData(base)
	base.CreatedAt = 3700
	base.Quota = 20
	base.TokenMetrics = &second
	LogQuotaData(base)
	SaveQuotaDataCache()

	var row QuotaData
	require.NoError(t, DB.Where("user_id = ?", 1).First(&row).Error)
	assert.Equal(t, "primary", row.TokenName)
	assert.Equal(t, int64(75), row.InputTokens)
	assert.Equal(t, int64(23), row.OutputTokens)
	assert.Equal(t, int64(10), row.CacheCreationTokens)
	assert.Equal(t, int64(32), row.CacheReadTokens)
	assert.Equal(t, int64(2), row.TokenMetricsCount)
	assert.Equal(t, 30, row.Quota)
}

func TestAdminLogQueriesFilterByStableUserID(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 1, Username: "old-alice", Type: LogTypeConsume, CreatedAt: now, ModelName: "gpt-a", Quota: 10, PromptTokens: 20, CompletionTokens: 5},
		{UserId: 2, Username: "bob", Type: LogTypeConsume, CreatedAt: now, ModelName: "gpt-a", Quota: 99, PromptTokens: 100, CompletionTokens: 50},
	}).Error)

	logs, total, err := GetAllLogs(LogTypeConsume, now-1, now+1, "", "", 1, "", 0, 10, 0, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, 1, logs[0].UserId)

	stat, err := SumUsedQuota(LogTypeConsume, now-1, now+1, "", "", 1, "", 0, "")
	require.NoError(t, err)
	assert.Equal(t, 10, stat.Quota)
	assert.Equal(t, 1, stat.Rpm)
	assert.Equal(t, 25, stat.Tpm)
}

func TestRecordConsumeLogFallsBackForLegacyTokenCallers(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	originalDataExportEnabled := common.DataExportEnabled
	common.DataExportEnabled = true
	t.Cleanup(func() {
		common.DataExportEnabled = originalDataExportEnabled
		CacheQuotaDataLock.Lock()
		CacheQuotaData = make(map[string]*QuotaData)
		CacheQuotaDataLock.Unlock()
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("username", "alice")
	RecordConsumeLog(ctx, 1, RecordConsumeLogParams{
		PromptTokens:     30,
		CompletionTokens: 7,
		ModelName:        "gpt-a",
		TokenName:        "primary",
	})

	CacheQuotaDataLock.Lock()
	require.Len(t, CacheQuotaData, 1)
	for _, row := range CacheQuotaData {
		assert.Equal(t, int64(30), row.InputTokens)
		assert.Equal(t, int64(7), row.OutputTokens)
		assert.Equal(t, int64(1), row.TokenMetricsCount)
	}
	CacheQuotaDataLock.Unlock()
}
