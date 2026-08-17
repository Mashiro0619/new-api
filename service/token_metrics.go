package service

import (
	"math"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
)

func isLegacyClaudeDerivedOpenAIUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if usage == nil {
		return false
	}
	if relayInfo != nil && relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return false
	}
	if usage.UsageSource != "" || usage.UsageSemantic != "" {
		return false
	}
	return usage.ClaudeCacheCreation5mTokens > 0 || usage.ClaudeCacheCreation1hTokens > 0
}

func tokenMetricsInputAlreadySeparated(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	return usageSemanticFromUsage(relayInfo, usage) == dto.BillingUsageSemanticAnthropic ||
		isLegacyClaudeDerivedOpenAIUsage(relayInfo, usage)
}

func saturatingNonNegativeTokenSum(left, right int) int {
	left = max(left, 0)
	right = max(right, 0)
	if right > math.MaxInt-left {
		return math.MaxInt
	}
	return left + right
}

func tokenMetricsCacheCreationTokens(usage *dto.Usage) int64 {
	if usage == nil {
		return 0
	}

	aggregate := int64(usage.PromptTokensDetails.CacheCreationTokensTotal())
	split := int64(saturatingNonNegativeTokenSum(
		usage.ClaudeCacheCreation5mTokens,
		usage.ClaudeCacheCreation1hTokens,
	))
	return max(aggregate, split)
}

func normalizeTokenMetricsForUsage(
	relayInfo *relaycommon.RelayInfo,
	usage *dto.Usage,
	inputTokens int64,
	outputTokens int64,
	cacheCreationTokens int64,
	cacheReadTokens int64,
) model.TokenMetrics {
	return model.NormalizeTokenMetrics(
		inputTokens,
		outputTokens,
		cacheCreationTokens,
		cacheReadTokens,
		tokenMetricsInputAlreadySeparated(relayInfo, usage),
	)
}

// BuildTokenMetrics returns non-overlapping token metrics using the same
// protocol semantics as text settlement. Structured BillingUsage takes
// precedence over compatibility fields when it is available.
func BuildTokenMetrics(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) model.TokenMetrics {
	usage = effectiveBillingUsage(usage)
	if usage == nil {
		return model.TokenMetrics{}
	}
	return normalizeTokenMetricsForUsage(
		relayInfo,
		usage,
		int64(usage.PromptTokens),
		int64(usage.CompletionTokens),
		tokenMetricsCacheCreationTokens(usage),
		int64(usage.PromptTokensDetails.CachedTokens),
	)
}
