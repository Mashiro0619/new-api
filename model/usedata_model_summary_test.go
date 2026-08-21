package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetModelTokenMetricsSummariesAggregatesMeasuredRows(t *testing.T) {
	truncateTables(t)
	rows := []QuotaData{
		{ModelName: "gpt-a", InputTokens: 100, OutputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 50, TokenMetricsCount: 2},
		{ModelName: "gpt-a", InputTokens: 40, OutputTokens: 10, CacheReadTokens: 5, TokenMetricsCount: 1},
		{ModelName: "legacy", InputTokens: 999, OutputTokens: 999, TokenMetricsCount: 0},
	}
	require.NoError(t, DB.Create(&rows).Error)

	summaries, err := GetModelTokenMetricsSummaries()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, "gpt-a", summaries[0].ModelName)
	require.EqualValues(t, 140, summaries[0].InputTokens)
	require.EqualValues(t, 30, summaries[0].OutputTokens)
	require.EqualValues(t, 10, summaries[0].CacheCreationTokens)
	require.EqualValues(t, 55, summaries[0].CacheReadTokens)
	require.EqualValues(t, 3, summaries[0].TokenMetricsCount)
}
