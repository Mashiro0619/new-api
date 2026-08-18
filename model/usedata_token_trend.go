package model

import (
	"database/sql"
	"errors"
	"math"

	"gorm.io/gorm"
)

const (
	TokenTrendGranularityHour = "hour"
	TokenTrendGranularityDay  = "day"

	MinTokenTrendTimezoneOffset = -14 * 60
	MaxTokenTrendTimezoneOffset = 14 * 60
)

type TokenTrendQuery struct {
	StartTimestamp int64
	EndTimestamp   int64
	Granularity    string
	TimezoneOffset int
	UserID         int
	Username       string
	ModelName      string
	TokenName      string
	UseGroup       string
	ChannelID      int
}

type TokenTrendMetrics struct {
	InputTokens         int64    `json:"input_tokens"`
	OutputTokens        int64    `json:"output_tokens"`
	ConsumedQuota       int64    `json:"consumed_quota"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
	CacheHitRate        *float64 `json:"cache_hit_rate"`
	TrackedRequests     int64    `json:"tracked_requests"`
}

type TokenTrendPoint struct {
	Timestamp int64 `json:"timestamp"`
	TokenTrendMetrics
}

type TokenTrendData struct {
	Available         bool              `json:"available"`
	Reason            string            `json:"reason"`
	TrackingStartedAt *int64            `json:"tracking_started_at"`
	StartTimestamp    int64             `json:"start_timestamp"`
	EndTimestamp      int64             `json:"end_timestamp"`
	Granularity       string            `json:"granularity"`
	Totals            TokenTrendMetrics `json:"totals"`
	Points            []TokenTrendPoint `json:"points"`
}

type TokenTrendUser struct {
	ID           int            `json:"id"`
	Username     string         `json:"username"`
	DisplayName  string         `json:"display_name"`
	Role         int            `json:"role"`
	Status       int            `json:"status"`
	Group        string         `json:"group"`
	UsedQuota    int            `json:"used_quota"`
	RequestCount int            `json:"request_count"`
	DeletedAt    gorm.DeletedAt `json:"DeletedAt"`
}

type tokenTrendDatabaseRow struct {
	CreatedAt           int64 `gorm:"column:created_at"`
	InputTokens         int64 `gorm:"column:input_tokens"`
	OutputTokens        int64 `gorm:"column:output_tokens"`
	ConsumedQuota       int64 `gorm:"column:consumed_quota"`
	CacheCreationTokens int64 `gorm:"column:cache_creation_tokens"`
	CacheReadTokens     int64 `gorm:"column:cache_read_tokens"`
	TrackedRequests     int64 `gorm:"column:tracked_requests"`
}

func newEmptyTokenTrendData(query TokenTrendQuery) TokenTrendData {
	return TokenTrendData{
		Available:      true,
		StartTimestamp: query.StartTimestamp,
		EndTimestamp:   query.EndTimestamp,
		Granularity:    query.Granularity,
		Points:         make([]TokenTrendPoint, 0),
	}
}

func applyTokenTrendFilters(tx *gorm.DB, query TokenTrendQuery) *gorm.DB {
	if query.UserID > 0 {
		tx = tx.Where("user_id = ?", query.UserID)
	}
	if query.Username != "" {
		tx = tx.Where("username = ?", query.Username)
	}
	if query.ModelName != "" {
		tx = tx.Where("model_name = ?", query.ModelName)
	}
	if query.TokenName != "" {
		tx = tx.Where("token_name = ?", query.TokenName)
	}
	if query.UseGroup != "" {
		tx = tx.Where("use_group = ?", query.UseGroup)
	}
	if query.ChannelID > 0 {
		tx = tx.Where("channel_id = ?", query.ChannelID)
	}
	return tx
}

func tokenTrendBucketSeconds(granularity string) (int64, error) {
	switch granularity {
	case TokenTrendGranularityHour:
		return 3600, nil
	case TokenTrendGranularityDay:
		return 86400, nil
	default:
		return 0, errors.New("invalid granularity")
	}
}

func tokenTrendTimezoneOffsetSeconds(timezoneOffset int) (int64, error) {
	if timezoneOffset < MinTokenTrendTimezoneOffset || timezoneOffset > MaxTokenTrendTimezoneOffset {
		return 0, errors.New("invalid timezone_offset")
	}
	return int64(timezoneOffset) * 60, nil
}

func checkedAddTokenTrendTimestamp(value, delta int64) (int64, bool) {
	if delta > 0 && value > math.MaxInt64-delta {
		return 0, false
	}
	if delta < 0 && value < math.MinInt64-delta {
		return 0, false
	}
	return value + delta, true
}

func checkedSubtractTokenTrendTimestamp(value, subtrahend int64) (int64, bool) {
	if subtrahend > 0 && value < math.MinInt64+subtrahend {
		return 0, false
	}
	if subtrahend < 0 && value > math.MaxInt64+subtrahend {
		return 0, false
	}
	return value - subtrahend, true
}

func tokenTrendBucketTimestamp(timestamp, bucketSeconds, timezoneOffsetSeconds int64) (int64, error) {
	shiftedTimestamp, ok := checkedSubtractTokenTrendTimestamp(timestamp, timezoneOffsetSeconds)
	if !ok {
		return 0, errors.New("timestamp range cannot be bucketed safely")
	}
	remainder := shiftedTimestamp % bucketSeconds
	if remainder < 0 {
		remainder += bucketSeconds
	}
	localBucket, ok := checkedSubtractTokenTrendTimestamp(shiftedTimestamp, remainder)
	if !ok {
		return 0, errors.New("timestamp range cannot be bucketed safely")
	}
	bucket, ok := checkedAddTokenTrendTimestamp(localBucket, timezoneOffsetSeconds)
	if !ok {
		return 0, errors.New("timestamp range cannot be bucketed safely")
	}
	return bucket, nil
}

func tokenTrendBucketRange(query TokenTrendQuery, bucketSeconds, timezoneOffsetSeconds int64) (int64, int64, int64, error) {
	if query.StartTimestamp <= 0 || query.EndTimestamp <= 0 || query.EndTimestamp < query.StartTimestamp {
		return 0, 0, 0, errors.New("invalid time range")
	}
	startBucket, err := tokenTrendBucketTimestamp(query.StartTimestamp, bucketSeconds, timezoneOffsetSeconds)
	if err != nil {
		return 0, 0, 0, err
	}
	endBucket, err := tokenTrendBucketTimestamp(query.EndTimestamp, bucketSeconds, timezoneOffsetSeconds)
	if err != nil {
		return 0, 0, 0, err
	}
	endExclusive, ok := checkedAddTokenTrendTimestamp(endBucket, bucketSeconds)
	if !ok {
		return 0, 0, 0, errors.New("timestamp range cannot be bucketed safely")
	}
	return startBucket, endBucket, endExclusive, nil
}

func tokenTrendCacheHitRate(inputTokens, cacheReadTokens int64) *float64 {
	denominator := float64(inputTokens) + float64(cacheReadTokens)
	if denominator <= 0 {
		return nil
	}
	rate := float64(cacheReadTokens) / denominator * 100
	return &rate
}

func addTokenTrendMetrics(target *TokenTrendMetrics, source tokenTrendDatabaseRow) {
	target.InputTokens += source.InputTokens
	target.OutputTokens += source.OutputTokens
	target.ConsumedQuota += source.ConsumedQuota
	target.CacheCreationTokens += source.CacheCreationTokens
	target.CacheReadTokens += source.CacheReadTokens
	target.TrackedRequests += source.TrackedRequests
}

func finalizeTokenTrendMetrics(metrics *TokenTrendMetrics) {
	metrics.CacheHitRate = tokenTrendCacheHitRate(metrics.InputTokens, metrics.CacheReadTokens)
}

func GetTokenTrendUser(userID int) (*TokenTrendUser, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}

	var user TokenTrendUser
	err := DB.Unscoped().
		Model(&User{}).
		Select("id", "username", "display_name", "role", "status", commonGroupCol, "used_quota", "request_count", "deleted_at").
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetTokenTrend(query TokenTrendQuery) (TokenTrendData, error) {
	result := newEmptyTokenTrendData(query)
	bucketSeconds, err := tokenTrendBucketSeconds(query.Granularity)
	if err != nil {
		return result, err
	}
	timezoneOffsetSeconds, err := tokenTrendTimezoneOffsetSeconds(query.TimezoneOffset)
	if err != nil {
		return result, err
	}
	startBucket, endBucket, endExclusive, err := tokenTrendBucketRange(query, bucketSeconds, timezoneOffsetSeconds)
	if err != nil {
		return result, err
	}
	if query.UserID > 0 && query.Username != "" {
		return result, errors.New("user_id and username cannot be used together")
	}

	baseQuery := applyTokenTrendFilters(
		DB.Table("quota_data").Where("token_metrics_count > 0"),
		query,
	)
	var trackingStartedAt sql.NullInt64
	if err := baseQuery.Select("MIN(created_at)").Scan(&trackingStartedAt).Error; err != nil {
		return result, err
	}
	if !trackingStartedAt.Valid {
		return result, nil
	}
	result.TrackingStartedAt = &trackingStartedAt.Int64

	rows := make([]tokenTrendDatabaseRow, 0)
	err = baseQuery.
		Select("created_at, sum(input_tokens) as input_tokens, sum(output_tokens) as output_tokens, sum(quota) as consumed_quota, sum(cache_creation_tokens) as cache_creation_tokens, sum(cache_read_tokens) as cache_read_tokens, sum(token_metrics_count) as tracked_requests").
		Where("created_at >= ? and created_at < ?", startBucket, endExclusive).
		Group("created_at").
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return result, err
	}

	trackingBucket, err := tokenTrendBucketTimestamp(trackingStartedAt.Int64, bucketSeconds, timezoneOffsetSeconds)
	if err != nil {
		return result, err
	}
	fillStart := max(startBucket, trackingBucket)
	if fillStart > endBucket {
		return result, nil
	}

	metricsByBucket := make(map[int64]*TokenTrendMetrics, len(rows))
	for _, row := range rows {
		bucket, err := tokenTrendBucketTimestamp(row.CreatedAt, bucketSeconds, timezoneOffsetSeconds)
		if err != nil {
			return result, err
		}
		metrics := metricsByBucket[bucket]
		if metrics == nil {
			metrics = &TokenTrendMetrics{}
			metricsByBucket[bucket] = metrics
		}
		addTokenTrendMetrics(metrics, row)
		addTokenTrendMetrics(&result.Totals, row)
	}
	finalizeTokenTrendMetrics(&result.Totals)

	bucketSpan, ok := checkedSubtractTokenTrendTimestamp(endBucket, fillStart)
	if !ok {
		return result, errors.New("timestamp range cannot be bucketed safely")
	}
	pointCount64 := bucketSpan/bucketSeconds + 1
	maxInt := int64(^uint(0) >> 1)
	if pointCount64 > maxInt {
		return result, errors.New("timestamp range cannot be bucketed safely")
	}
	pointCount := int(pointCount64)
	result.Points = make([]TokenTrendPoint, 0, pointCount)
	for index := 0; index < pointCount; index++ {
		timestamp := fillStart + int64(index)*bucketSeconds
		metrics := TokenTrendMetrics{}
		if measured := metricsByBucket[timestamp]; measured != nil {
			metrics = *measured
			finalizeTokenTrendMetrics(&metrics)
		}
		result.Points = append(result.Points, TokenTrendPoint{
			Timestamp:         timestamp,
			TokenTrendMetrics: metrics,
		})
	}

	return result, nil
}
