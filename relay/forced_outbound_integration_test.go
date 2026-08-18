package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const forcedOutboundIntegrationText = "forced outbound integration response"

type forcedOutboundCapturedRequest struct {
	method           string
	path             string
	rawQuery         string
	authorization    string
	contentType      string
	anthropicVersion string
	xAPIKey          string
	xGoogAPIKey      string
	body             []byte
}

func TestForcedOutboundHTTPRoundTripUsesTargetProtocolForSupportedChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targets := []struct {
		name             string
		format           types.RelayFormat
		wantPath         string
		wantQuery        string
		upstreamResponse string
	}{
		{
			name:     "openai_chat_completions",
			format:   types.RelayFormatOpenAI,
			wantPath: "/v1/chat/completions",
			upstreamResponse: `{
				"id":"chatcmpl-forced","object":"chat.completion","created":1700000000,"model":"forced-model",
				"choices":[{"index":0,"message":{"role":"assistant","content":"forced outbound integration response"},"finish_reason":"stop"}],
				"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}
			}`,
		},
		{
			name:     "openai_responses",
			format:   types.RelayFormatOpenAIResponses,
			wantPath: "/v1/responses",
			upstreamResponse: `{
				"id":"resp-forced","object":"response","created_at":1700000000,"model":"forced-model","status":"completed",
				"output":[{"id":"msg-forced","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"forced outbound integration response"}]}],
				"usage":{"input_tokens":5,"output_tokens":3,"total_tokens":8}
			}`,
		},
		{
			name:     "anthropic_messages",
			format:   types.RelayFormatClaude,
			wantPath: "/v1/messages",
			upstreamResponse: `{
				"id":"msg-forced","type":"message","role":"assistant","model":"forced-model",
				"content":[{"type":"text","text":"forced outbound integration response"}],
				"stop_reason":"end_turn","usage":{"input_tokens":5,"output_tokens":3}
			}`,
		},
		{
			name:      "gemini_generate_content",
			format:    types.RelayFormatGemini,
			wantPath:  "/v1beta/models/forced-model:generateContent",
			wantQuery: "",
			upstreamResponse: `{
				"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"forced outbound integration response"}]},"finishReason":"STOP"}],
				"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3,"totalTokenCount":8}
			}`,
		},
	}

	channelTypes := []struct {
		name string
		id   int
	}{
		{name: "openai_channel", id: constant.ChannelTypeOpenAI},
		{name: "new_api_channel", id: constant.ChannelTypeNewAPI},
	}

	for _, channelType := range channelTypes {
		for _, target := range targets {
			t.Run(channelType.name+"/"+target.name, func(t *testing.T) {
				captured := make(chan forcedOutboundCapturedRequest, 1)
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					captured <- forcedOutboundCapturedRequest{
						method:           r.Method,
						path:             r.URL.Path,
						rawQuery:         r.URL.RawQuery,
						authorization:    r.Header.Get("Authorization"),
						contentType:      r.Header.Get("Content-Type"),
						anthropicVersion: r.Header.Get("anthropic-version"),
						xAPIKey:          r.Header.Get("x-api-key"),
						xGoogAPIKey:      r.Header.Get("x-goog-api-key"),
						body:             body,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(target.upstreamResponse))
				}))
				t.Cleanup(upstream.Close)

				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
				c.Request.Header.Set("Content-Type", "application/json")

				maxTokens := uint(64)
				request := &dto.GeneralOpenAIRequest{
					Model:     "forced-model",
					MaxTokens: &maxTokens,
					Messages: []dto.Message{
						{Role: "user", Content: "forced outbound integration request"},
					},
				}
				info := &relaycommon.RelayInfo{
					RelayFormat:            types.RelayFormatOpenAI,
					RelayMode:              relayconstant.RelayModeChatCompletions,
					OriginModelName:        "forced-model",
					RequestURLPath:         "/v1/chat/completions",
					RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI},
					ChannelMeta: &relaycommon.ChannelMeta{
						ChannelType:       channelType.id,
						ChannelBaseUrl:    upstream.URL,
						ApiKey:            "forced-secret",
						UpstreamModelName: "forced-model",
						ChannelSetting: dto.ChannelSettings{
							ForcedOutboundFormat: target.format,
						},
					},
				}

				usage, enabled, relayErr := tryForcedOutbound(c, info, request)
				require.True(t, enabled)
				require.Nil(t, relayErr)
				require.NotNil(t, usage)
				assert.Equal(t, 5, usage.PromptTokens)
				assert.Equal(t, 3, usage.CompletionTokens)
				assert.Equal(t, 8, usage.TotalTokens)
				assert.Equal(t, target.format, info.FinalRequestRelayFormat)

				gotRequest := <-captured
				assert.Equal(t, http.MethodPost, gotRequest.method)
				assert.Equal(t, target.wantPath, gotRequest.path)
				assert.Equal(t, target.wantQuery, gotRequest.rawQuery)
				assert.Equal(t, "Bearer forced-secret", gotRequest.authorization)
				assert.Equal(t, "application/json", gotRequest.contentType)
				assert.Empty(t, gotRequest.xAPIKey)
				assert.Empty(t, gotRequest.xGoogAPIKey)
				if target.format == types.RelayFormatClaude {
					assert.Equal(t, "2023-06-01", gotRequest.anthropicVersion)
				} else {
					assert.Empty(t, gotRequest.anthropicVersion)
				}
				assertForcedOutboundRequestShape(t, target.format, gotRequest.body)

				assertForcedOutboundOpenAIResponse(t, recorder.Body.Bytes())
			})
		}
	}
}

func assertForcedOutboundRequestShape(t *testing.T, target types.RelayFormat, body []byte) {
	t.Helper()

	var payload map[string]any
	require.NoError(t, rootcommon.Unmarshal(body, &payload))

	switch target {
	case types.RelayFormatOpenAI:
		assert.Equal(t, "forced-model", payload["model"])
		assert.Equal(t, float64(64), payload["max_tokens"])
		assertForcedOutboundMessageText(t, payload["messages"], "forced outbound integration request")
		assert.NotContains(t, payload, "input")
		assert.NotContains(t, payload, "contents")
	case types.RelayFormatOpenAIResponses:
		assert.Equal(t, "forced-model", payload["model"])
		assert.Equal(t, float64(64), payload["max_output_tokens"])
		assertForcedOutboundMessageText(t, payload["input"], "forced outbound integration request")
		assert.NotContains(t, payload, "messages")
		assert.NotContains(t, payload, "contents")
	case types.RelayFormatClaude:
		assert.Equal(t, "forced-model", payload["model"])
		assert.Equal(t, float64(64), payload["max_tokens"])
		assertForcedOutboundMessageText(t, payload["messages"], "forced outbound integration request")
		assert.NotContains(t, payload, "input")
		assert.NotContains(t, payload, "contents")
		assert.NotContains(t, payload, "max_output_tokens")
	case types.RelayFormatGemini:
		assertForcedOutboundGeminiText(t, payload["contents"], "forced outbound integration request")
		generationConfig, ok := payload["generationConfig"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(64), generationConfig["maxOutputTokens"])
		assert.NotContains(t, payload, "model")
		assert.NotContains(t, payload, "messages")
		assert.NotContains(t, payload, "input")
	default:
		t.Fatalf("unsupported target %q", target)
	}
}

func assertForcedOutboundMessageText(t *testing.T, raw any, want string) {
	t.Helper()

	messages, ok := raw.([]any)
	require.True(t, ok)
	require.NotEmpty(t, messages)
	message, ok := messages[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", message["role"])

	switch content := message["content"].(type) {
	case string:
		assert.Equal(t, want, content)
	case []any:
		require.NotEmpty(t, content)
		part, ok := content[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, want, part["text"])
	default:
		t.Fatalf("unexpected message content type %T", message["content"])
	}
}

func assertForcedOutboundGeminiText(t *testing.T, raw any, want string) {
	t.Helper()

	contents, ok := raw.([]any)
	require.True(t, ok)
	require.NotEmpty(t, contents)
	content, ok := contents[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", content["role"])
	parts, ok := content["parts"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, parts)
	part, ok := parts[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, want, part["text"])
}

func assertForcedOutboundOpenAIResponse(t *testing.T, body []byte) {
	t.Helper()

	var payload map[string]any
	require.NoError(t, rootcommon.Unmarshal(body, &payload))
	assert.Equal(t, "chat.completion", payload["object"])
	choices, ok := payload["choices"].([]any)
	require.True(t, ok)
	require.Len(t, choices, 1)
	choice, ok := choices[0].(map[string]any)
	require.True(t, ok)
	message, ok := choice["message"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "assistant", message["role"])
	assert.Equal(t, forcedOutboundIntegrationText, message["content"])
}
