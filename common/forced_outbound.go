package common

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
)

var forcedOutboundInboundEndpointTypes = [...]constant.EndpointType{
	constant.EndpointTypeOpenAI,
	constant.EndpointTypeOpenAIResponse,
	constant.EndpointTypeAnthropic,
	constant.EndpointTypeGemini,
}

// ChannelTypeSupportsForcedOutboundFormat is the shared channel-type
// capability whitelist for the forced text protocol override.
func ChannelTypeSupportsForcedOutboundFormat(channelType int, format types.RelayFormat) bool {
	switch channelType {
	case constant.ChannelTypeOpenAI, constant.ChannelTypeNewAPI:
		return dto.IsForcedOutboundFormatSupported(format)
	default:
		return false
	}
}

// AddForcedOutboundEndpointTypes advertises every inbound text protocol that
// can be converted to the configured target while retaining native non-text
// endpoints such as Responses Compact, Alpha Search, and image generation.
func AddForcedOutboundEndpointTypes(
	endpointTypes []constant.EndpointType,
	channelType int,
	format types.RelayFormat,
) []constant.EndpointType {
	if !ChannelTypeSupportsForcedOutboundFormat(channelType, format) {
		return endpointTypes
	}

	result := append([]constant.EndpointType(nil), endpointTypes...)
	seen := make(map[constant.EndpointType]struct{}, len(result)+len(forcedOutboundInboundEndpointTypes))
	for _, endpointType := range result {
		seen[endpointType] = struct{}{}
	}
	for _, endpointType := range forcedOutboundInboundEndpointTypes {
		if _, exists := seen[endpointType]; exists {
			continue
		}
		seen[endpointType] = struct{}{}
		result = append(result, endpointType)
	}
	return result
}
