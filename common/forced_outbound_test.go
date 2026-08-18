package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
)

func TestChannelTypeSupportsForcedOutboundFormat(t *testing.T) {
	supportedFormats := []types.RelayFormat{
		types.RelayFormatOpenAI,
		types.RelayFormatOpenAIResponses,
		types.RelayFormatClaude,
		types.RelayFormatGemini,
	}
	for _, channelType := range []int{constant.ChannelTypeOpenAI, constant.ChannelTypeNewAPI} {
		for _, format := range supportedFormats {
			assert.True(t, ChannelTypeSupportsForcedOutboundFormat(channelType, format))
		}
	}

	assert.False(t, ChannelTypeSupportsForcedOutboundFormat(constant.ChannelTypeDeepSeek, types.RelayFormatOpenAI))
	assert.False(t, ChannelTypeSupportsForcedOutboundFormat(constant.ChannelTypeAdvancedCustom, types.RelayFormatGemini))
	assert.False(t, ChannelTypeSupportsForcedOutboundFormat(constant.ChannelTypeOpenAI, types.RelayFormatOpenAIResponsesCompaction))
	assert.False(t, ChannelTypeSupportsForcedOutboundFormat(constant.ChannelTypeNewAPI, ""))
}

func TestAddForcedOutboundEndpointTypes(t *testing.T) {
	base := []constant.EndpointType{
		constant.EndpointTypeImageGeneration,
		constant.EndpointTypeOpenAI,
	}

	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeImageGeneration,
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
		constant.EndpointTypeAnthropic,
		constant.EndpointTypeGemini,
	}, AddForcedOutboundEndpointTypes(base, constant.ChannelTypeOpenAI, types.RelayFormatClaude))

	assert.Equal(t, base, AddForcedOutboundEndpointTypes(base, constant.ChannelTypeGemini, types.RelayFormatOpenAI))
}
