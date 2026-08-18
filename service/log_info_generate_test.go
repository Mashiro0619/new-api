package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTextOtherInfoIncludesForcedOutboundAuditMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	now := time.Unix(1_700_000_000, 0)
	relayInfo := &relaycommon.RelayInfo{
		StartTime:                   now,
		FirstResponseTime:           now,
		RelayFormat:                 types.RelayFormatClaude,
		AppliedForcedOutboundFormat: types.RelayFormatOpenAIResponses,
		RequestConversionChain:      []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI, types.RelayFormatGemini},
		FinalRequestRelayFormat:     types.RelayFormatGemini,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "must-not-appear-in-log",
			ChannelSetting: dto.ChannelSettings{
				ForcedOutboundFormat: types.RelayFormatOpenAIResponses,
			},
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 1, 1, 1, 0, 0, 0, 1)

	assert.Equal(t, string(types.RelayFormatClaude), other["inbound_relay_format"])
	assert.Equal(t, string(types.RelayFormatOpenAIResponses), other["forced_outbound_format"])
	assert.Equal(t, string(types.RelayFormatGemini), other["final_request_format"])
	assert.Equal(t, []string{"Claude Messages", "OpenAI Compatible", "Google Gemini"}, other["request_conversion"])
	encoded, err := common.Marshal(other)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), relayInfo.ApiKey)
}

func TestGenerateTextOtherInfoOmitsForcedOutboundAuditMetadataWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	now := time.Unix(1_700_000_000, 0)
	relayInfo := &relaycommon.RelayInfo{
		StartTime:              now,
		FirstResponseTime:      now,
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{
				ForcedOutboundFormat: types.RelayFormatClaude,
			},
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 1, 1, 1, 0, 0, 0, 1)

	assert.NotContains(t, other, "inbound_relay_format")
	assert.NotContains(t, other, "forced_outbound_format")
	assert.NotContains(t, other, "final_request_format")
	assert.Equal(t, []string{"OpenAI Compatible"}, other["request_conversion"])
}

func TestGenerateTextOtherInfoOmitsForcedOutboundAuditMetadataWhenConfiguredButNotApplied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	now := time.Unix(1_700_000_000, 0)
	relayInfo := &relaycommon.RelayInfo{
		StartTime:              now,
		FirstResponseTime:      now,
		RelayFormat:            types.RelayFormatEmbedding,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatEmbedding},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{
				ForcedOutboundFormat: types.RelayFormatClaude,
			},
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 1, 1, 1, 0, 0, 0, 1)

	assert.NotContains(t, other, "inbound_relay_format")
	assert.NotContains(t, other, "forced_outbound_format")
	assert.NotContains(t, other, "final_request_format")
}
