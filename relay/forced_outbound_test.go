package relay

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForcedOutboundAdaptorStandardURLs(t *testing.T) {
	tests := []struct {
		name     string
		target   types.RelayFormat
		stream   bool
		expected string
	}{
		{name: "openai chat", target: types.RelayFormatOpenAI, expected: "https://upstream.example/v1/chat/completions"},
		{name: "openai responses", target: types.RelayFormatOpenAIResponses, expected: "https://upstream.example/v1/responses"},
		{name: "anthropic messages", target: types.RelayFormatClaude, expected: "https://upstream.example/v1/messages"},
		{name: "gemini non-stream", target: types.RelayFormatGemini, expected: "https://upstream.example/v1beta/models/gemini-2.5-pro:generateContent"},
		{name: "gemini stream", target: types.RelayFormatGemini, stream: true, expected: "https://upstream.example/v1beta/models/gemini-2.5-pro:streamGenerateContent?alt=sse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				IsStream: tt.stream,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       constant.ChannelTypeNewAPI,
					ChannelBaseUrl:    "https://upstream.example",
					UpstreamModelName: "models/gemini-2.5-pro",
				},
			}
			url, err := (&forcedOutboundAdaptor{target: tt.target}).GetRequestURL(info)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestForcedOutboundAdaptorUsesBearerWithoutProtocolCredentialHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("anthropic-version", "2024-01-01")

	info := &relaycommon.RelayInfo{
		OriginModelName: "claude-test",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeNewAPI,
			ApiKey:      "secret-key",
		},
	}

	for _, target := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatGemini} {
		headers := http.Header{}
		err := (&forcedOutboundAdaptor{target: target}).SetupRequestHeader(c, &headers, info)
		require.NoError(t, err)
		assert.Equal(t, "Bearer secret-key", headers.Get("Authorization"))
		assert.Empty(t, headers.Get("x-api-key"))
		assert.Empty(t, headers.Get("x-goog-api-key"))
		if target == types.RelayFormatClaude {
			assert.Equal(t, "2024-01-01", headers.Get("anthropic-version"))
		} else {
			assert.Empty(t, headers.Get("anthropic-version"))
		}
	}
}

func TestForcedOutboundOnlyAppliesToSupportedTextEndpoints(t *testing.T) {
	setting := dto.ChannelSettings{ForcedOutboundFormat: types.RelayFormatClaude}
	tests := []struct {
		name   string
		format types.RelayFormat
		mode   int
		want   bool
	}{
		{name: "chat completions", format: types.RelayFormatOpenAI, mode: relayconstant.RelayModeChatCompletions, want: true},
		{name: "responses", format: types.RelayFormatOpenAIResponses, mode: relayconstant.RelayModeResponses, want: true},
		{name: "claude messages", format: types.RelayFormatClaude, mode: relayconstant.RelayModeUnknown, want: true},
		{name: "gemini generate content", format: types.RelayFormatGemini, mode: relayconstant.RelayModeGemini, want: true},
		{name: "legacy completions", format: types.RelayFormatOpenAI, mode: relayconstant.RelayModeCompletions},
		{name: "embedding", format: types.RelayFormatEmbedding, mode: relayconstant.RelayModeEmbeddings},
		{name: "image", format: types.RelayFormatOpenAIImage, mode: relayconstant.RelayModeImagesGenerations},
		{name: "audio", format: types.RelayFormatOpenAIAudio, mode: relayconstant.RelayModeAudioSpeech},
		{name: "realtime", format: types.RelayFormatOpenAIRealtime, mode: relayconstant.RelayModeRealtime},
		{name: "responses compact", format: types.RelayFormatOpenAIResponsesCompaction, mode: relayconstant.RelayModeResponsesCompact},
		{name: "alpha search", format: types.RelayFormatOpenAIAlphaSearch, mode: relayconstant.RelayModeAlphaSearch},
		{name: "gemini embedding", format: types.RelayFormatGemini, mode: relayconstant.RelayModeEmbeddings},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				RelayFormat: tt.format,
				RelayMode:   tt.mode,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:    constant.ChannelTypeOpenAI,
					ChannelSetting: setting,
				},
			}
			_, enabled, err := forcedOutboundTarget(info)
			require.NoError(t, err)
			assert.Equal(t, tt.want, enabled)
		})
	}
}

func TestPrepareForcedOutboundOverridesPassThroughAndReserializesSameFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test","messages":[],"unknown_passthrough_field":true}`))
	c.Request.Header.Set("Content-Type", "application/json")

	global := model_setting.GetGlobalSettings()
	originalPassThrough := global.PassThroughRequestEnabled
	global.PassThroughRequestEnabled = true
	t.Cleanup(func() {
		global.PassThroughRequestEnabled = originalPassThrough
	})

	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-test",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	info := &relaycommon.RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RelayMode:              relayconstant.RelayModeChatCompletions,
		OriginModelName:        "gpt-test",
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeNewAPI,
			ChannelBaseUrl:    "https://upstream.example",
			ApiKey:            "secret-key",
			UpstreamModelName: "gpt-test",
			ChannelSetting: dto.ChannelSettings{
				PassThroughBodyEnabled: true,
				ForcedOutboundFormat:   types.RelayFormatOpenAI,
			},
		},
	}

	adaptor, body, enabled, err := PrepareForcedOutboundRequest(c, info, request)
	require.NoError(t, err)
	require.True(t, enabled)
	require.NotNil(t, adaptor)
	assert.NotContains(t, string(body), "unknown_passthrough_field")
	assert.Contains(t, string(body), `"messages"`)
	assert.Equal(t, types.RelayFormat(types.RelayFormatOpenAI), info.FinalRequestRelayFormat)
	assert.Equal(t, []types.RelayFormat{types.RelayFormatOpenAI}, info.RequestConversionChain)
}

func TestPrepareForcedOutboundAppliesSystemPromptAfterTargetConversion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	request := &dto.GeneralOpenAIRequest{
		Model: "claude-test",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	info := &relaycommon.RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RelayMode:              relayconstant.RelayModeChatCompletions,
		OriginModelName:        "claude-test",
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeNewAPI,
			ChannelBaseUrl:    "https://upstream.example",
			ApiKey:            "secret-key",
			UpstreamModelName: "claude-test",
			ChannelSetting: dto.ChannelSettings{
				ForcedOutboundFormat: types.RelayFormatClaude,
				SystemPrompt:         "channel system prompt",
			},
		},
	}

	_, body, enabled, err := PrepareForcedOutboundRequest(c, info, request)
	require.NoError(t, err)
	require.True(t, enabled)

	var outbound dto.ClaudeRequest
	require.NoError(t, rootcommon.Unmarshal(body, &outbound))
	assert.Equal(t, "channel system prompt", outbound.GetStringSystem())
	assert.Equal(t, []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude}, info.RequestConversionChain)
}

func TestWriteForcedOutboundResponsesStreamAcceptsRegistryEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	event := relayconvert.ChatToResponsesStreamEvent{
		Type: "response.created",
		Payload: dto.ResponsesStreamResponse{
			Type: "response.created",
		},
	}
	err := writeForcedOutboundStreamResult(c, types.RelayFormatOpenAIResponses, event)
	require.NoError(t, err)
	assert.Contains(t, recorder.Body.String(), "event: response.created")
	assert.Contains(t, recorder.Body.String(), `"type":"response.created"`)
}

func TestInitChannelMetaResetsForcedOutboundConversionState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rootcommon.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeNewAPI)
	rootcommon.SetContextKey(c, constant.ContextKeyChannelId, 42)
	rootcommon.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://upstream.example")
	rootcommon.SetContextKey(c, constant.ContextKeyChannelKey, "secret-key")
	rootcommon.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-test")
	rootcommon.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{
		ForcedOutboundFormat: types.RelayFormatClaude,
	})
	c.Set("forced_outbound_response_started", true)
	c.Set("claude_web_search_requests", 3)
	c.Set("gemini_google_search_call", true)
	rootcommon.SetContextKey(c, constant.ContextKeyAdminRejectReason, "previous_attempt_rejected")

	request := &dto.GeneralOpenAIRequest{Model: "gpt-test"}
	info := &relaycommon.RelayInfo{
		RelayFormat:                 types.RelayFormatOpenAI,
		OriginModelName:             "gpt-test",
		Request:                     request,
		RequestConversionChain:      []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatGemini},
		FinalRequestRelayFormat:     types.RelayFormatGemini,
		AppliedForcedOutboundFormat: types.RelayFormatGemini,
	}

	info.InitChannelMeta(c)
	assert.Equal(t, []types.RelayFormat{types.RelayFormatOpenAI}, info.RequestConversionChain)
	assert.Empty(t, info.FinalRequestRelayFormat)
	assert.Empty(t, info.AppliedForcedOutboundFormat)
	assert.False(t, c.GetBool("forced_outbound_response_started"))
	assert.Zero(t, c.GetInt("claude_web_search_requests"))
	assert.False(t, c.GetBool("gemini_google_search_call"))
	assert.Empty(t, rootcommon.GetContextKeyString(c, constant.ContextKeyAdminRejectReason))
}

func TestForcedResponsesStreamObserverDoesNotDoubleCountTerminalOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{}
	observer := &forcedResponseObserver{}
	outputIndex := 0
	output := dto.ResponsesOutput{
		Type:   "message",
		ID:     "message-1",
		Role:   "assistant",
		Status: "completed",
		Content: []dto.ResponsesOutputContent{
			{Type: "output_text", Text: "hello world"},
		},
	}

	observer.observe(c, info, types.RelayFormatOpenAIResponses, &dto.ResponsesStreamResponse{
		Type:        "response.output_text.delta",
		Delta:       "hello world",
		OutputIndex: &outputIndex,
		ItemID:      output.ID,
	}, true)
	observer.observe(c, info, types.RelayFormatOpenAIResponses, &dto.ResponsesStreamResponse{
		Type:        dto.ResponsesOutputTypeItemDone,
		OutputIndex: &outputIndex,
		Item:        &output,
	}, true)
	observer.observe(c, info, types.RelayFormatOpenAIResponses, &dto.ResponsesStreamResponse{
		Type: "response.completed",
		Response: &dto.OpenAIResponsesResponse{
			Output: []dto.ResponsesOutput{output},
		},
	}, true)

	assert.Equal(t, "hello world", observer.outputText.String())
}

func TestForcedUsageAccumulatorMergesSplitClaudeUsage(t *testing.T) {
	accumulator := &forcedUsageAccumulator{}
	accumulator.add(relayconvert.UsageFromClaudeAPIUsage(&dto.ClaudeUsage{
		InputTokens:              7,
		CacheReadInputTokens:     3,
		CacheCreationInputTokens: 5,
		CacheCreation: &dto.ClaudeCacheCreationUsage{
			Ephemeral5mInputTokens: 2,
			Ephemeral1hInputTokens: 3,
		},
	}))
	accumulator.add(relayconvert.UsageFromClaudeAPIUsage(&dto.ClaudeUsage{OutputTokens: 11}))

	require.NotNil(t, accumulator.usage)
	assert.Equal(t, 15, accumulator.usage.PromptTokens)
	assert.Equal(t, 11, accumulator.usage.CompletionTokens)
	assert.Equal(t, 3, accumulator.usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 5, accumulator.usage.PromptTokensDetails.CachedCreationTokens)
	require.NotNil(t, accumulator.usage.BillingUsage)
	require.NotNil(t, accumulator.usage.BillingUsage.ClaudeUsage)
	assert.Equal(t, 7, accumulator.usage.BillingUsage.ClaudeUsage.InputTokens)
	assert.Equal(t, 11, accumulator.usage.BillingUsage.ClaudeUsage.OutputTokens)
	assert.Equal(t, 3, accumulator.usage.BillingUsage.ClaudeUsage.CacheReadInputTokens)
	assert.Equal(t, 5, accumulator.usage.BillingUsage.ClaudeUsage.CacheCreationInputTokens)
}

func TestValidateForcedGeminiResponseRejectsBlockedAndEmptyResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("blocked prompt", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		reason := "SAFETY"
		err := validateForcedGeminiResponse(c, &dto.GeminiChatResponse{
			PromptFeedback: &dto.GeminiChatPromptFeedback{BlockReason: &reason},
		}, true)

		require.NotNil(t, err)
		assert.Equal(t, http.StatusBadRequest, err.StatusCode)
		assert.Equal(t, "gemini_block_reason=SAFETY", rootcommon.GetContextKeyString(c, constant.ContextKeyAdminRejectReason))
	})

	t.Run("empty first stream chunk", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		err := validateForcedGeminiResponse(c, &dto.GeminiChatResponse{}, true)

		require.NotNil(t, err)
		assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
		assert.Equal(t, "gemini_empty_candidates", rootcommon.GetContextKeyString(c, constant.ContextKeyAdminRejectReason))
	})

	t.Run("usage-only later stream chunk", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		assert.Nil(t, validateForcedGeminiResponse(c, &dto.GeminiChatResponse{}, false))
	})
}

func TestProtectForcedOutboundStreamErrorSkipsRetryAfterResponseStarted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	original := types.NewError(errors.New("stream conversion failed"), types.ErrorCodeBadResponseBody)

	assert.False(t, types.IsSkipRetryError(protectForcedOutboundStreamError(c, original)))
	markForcedOutboundResponseStarted(c)
	assert.True(t, types.IsSkipRetryError(protectForcedOutboundStreamError(c, original)))
}
