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
