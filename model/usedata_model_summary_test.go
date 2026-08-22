package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetModelTokenMetricsSummariesAggregatesMeasuredRows(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaData["pending-gpt-a"] = &QuotaData{
		ModelName:           "gpt-a",
		InputTokens:         7,
		OutputTokens:        2,
		CacheCreationTokens: 3,
		CacheReadTokens:     4,
		TokenMetricsCount:   1,
	}
	CacheQuotaData["pending-memory-only"] = &QuotaData{
		ModelName:         "memory-only",
		InputTokens:       20,
		OutputTokens:      5,
		TokenMetricsCount: 1,
	}
	CacheQuotaData["pending-legacy"] = &QuotaData{
		ModelName:         "legacy",
		InputTokens:       20,
		TokenMetricsCount: 0,
	}
	CacheQuotaDataLock.Unlock()
	t.Cleanup(func() {
		CacheQuotaDataLock.Lock()
		CacheQuotaData = make(map[string]*QuotaData)
		CacheQuotaDataLock.Unlock()
	})

	rows := []QuotaData{
		{ModelName: "gpt-a", InputTokens: 100, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 50, TokenMetricsCount: 2},
		{ModelName: "gpt-a", InputTokens: 40, OutputTokens: 10, CacheReadTokens: 5, TokenMetricsCount: 1},
		{ModelName: "legacy", InputTokens: 999, OutputTokens: 999, TokenMetricsCount: 0},
	}
	require.NoError(t, DB.Create(&rows).Error)

	summaries, err := GetModelTokenMetricsSummaries()
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	require.Equal(t, "gpt-a", summaries[0].ModelName)
	require.EqualValues(t, 147, summaries[0].InputTokens)
	require.EqualValues(t, 32, summaries[0].OutputTokens)
	require.EqualValues(t, 13, summaries[0].CacheCreationTokens)
	require.EqualValues(t, 59, summaries[0].CacheReadTokens)
	require.EqualValues(t, 4, summaries[0].TokenMetricsCount)
	require.EqualValues(t, 251, summaries[0].TotalTokens())
	rate := summaries[0].CacheHitRate()
	require.NotNil(t, rate)
	require.InDelta(t, 59.0/206.0*100, *rate, 0.0001)
	require.Equal(t, "memory-only", summaries[1].ModelName)
	require.EqualValues(t, 1, summaries[1].TokenMetricsCount)
	require.Nil(t, summaries[1].CacheHitRate())
	names, err := GetModelTokenMetricNames()
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a", "memory-only"}, names)
}

func TestIncrementProhibitedWordHitsDeduplicatesAndIncrements(t *testing.T) {
	truncateTables(t)
	user := User{Username: "prohibited-user", Password: "password", Role: 1, Status: 1}
	require.NoError(t, DB.Create(&user).Error)

	require.NoError(t, IncrementProhibitedWordHits(user.Id, []string{"Alpha", "alpha", "Beta"}))
	require.NoError(t, IncrementProhibitedWordHits(user.Id, []string{"alpha"}))

	var hits []ProhibitedWordHit
	require.NoError(t, DB.Where("user_id = ?", user.Id).Order("keyword ASC").Find(&hits).Error)
	require.Len(t, hits, 2)
	require.Equal(t, "alpha", hits[0].Keyword)
	require.EqualValues(t, 2, hits[0].HitCount)
	require.Equal(t, "beta", hits[1].Keyword)
	require.EqualValues(t, 1, hits[1].HitCount)
}

func TestGetProhibitedWordSummaryIncludesUsersWithoutHits(t *testing.T) {
	truncateTables(t)
	users := []User{
		{Username: "hit-user", Password: "password", Role: 1, Status: 1, AffCode: "hit-user-code"},
		{Username: "quiet-user", Password: "password", Role: 1, Status: 1, AffCode: "quiet-user-code"},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, IncrementProhibitedWordHits(users[0].Id, []string{"alpha"}))

	page, err := GetProhibitedWordSummary(1, 50, []string{"alpha", "beta"})
	require.NoError(t, err)
	require.EqualValues(t, 2, page.Total)
	require.Len(t, page.Items, 2)
	require.EqualValues(t, 1, page.Items[0].Counts["alpha"])
	require.EqualValues(t, 0, page.Items[0].Counts["beta"])
	require.EqualValues(t, 0, page.Items[1].Counts["alpha"])
	require.EqualValues(t, 0, page.Items[1].Counts["beta"])
}
