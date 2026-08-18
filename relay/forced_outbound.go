package relay

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"
	claudechannel "github.com/QuantumNous/new-api/relay/channel/claude"
	geminichannel "github.com/QuantumNous/new-api/relay/channel/gemini"
	openaichannel "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// forcedOutboundAdaptor deliberately ignores the channel's default API type.
// It keeps channel transport settings while selecting URL, headers and response
// decoding from the explicitly configured outbound protocol.
type forcedOutboundAdaptor struct {
	target types.RelayFormat
}

const forcedOutboundResponseStartedKey = "forced_outbound_response_started"

// forcedUsageAccumulator keeps the best-known usage while a protocol is being
// converted. Some upstreams (notably Anthropic) split input/cache and output
// usage across different events, so replacing the previous value would lose
// billable fields.
type forcedUsageAccumulator struct {
	usage *dto.Usage
}

func (a *forcedUsageAccumulator) add(usage *dto.Usage) {
	if usage == nil {
		return
	}
	if a.usage == nil {
		copy := *usage
		copy.BillingUsage = dto.CloneBillingUsage(usage.BillingUsage)
		if usage.InputTokensDetails != nil {
			details := *usage.InputTokensDetails
			copy.InputTokensDetails = &details
		}
		a.usage = &copy
		return
	}
	mergeForcedUsage(a.usage, usage)
}

func mergeForcedUsage(dst, src *dto.Usage) {
	if dst == nil || src == nil {
		return
	}
	if src.PromptTokens > dst.PromptTokens {
		dst.PromptTokens = src.PromptTokens
	}
	if src.CompletionTokens > dst.CompletionTokens {
		dst.CompletionTokens = src.CompletionTokens
	}
	if src.TotalTokens > dst.TotalTokens {
		dst.TotalTokens = src.TotalTokens
	}
	if src.PromptTokensDetails.CachedTokens > dst.PromptTokensDetails.CachedTokens {
		dst.PromptTokensDetails.CachedTokens = src.PromptTokensDetails.CachedTokens
	}
	if src.PromptTokensDetails.CachedCreationTokens > dst.PromptTokensDetails.CachedCreationTokens {
		dst.PromptTokensDetails.CachedCreationTokens = src.PromptTokensDetails.CachedCreationTokens
	}
	if src.PromptTokensDetails.CacheWriteTokens > dst.PromptTokensDetails.CacheWriteTokens {
		dst.PromptTokensDetails.CacheWriteTokens = src.PromptTokensDetails.CacheWriteTokens
	}
	if src.PromptTokensDetails.TextTokens > dst.PromptTokensDetails.TextTokens {
		dst.PromptTokensDetails.TextTokens = src.PromptTokensDetails.TextTokens
	}
	if src.PromptTokensDetails.AudioTokens > dst.PromptTokensDetails.AudioTokens {
		dst.PromptTokensDetails.AudioTokens = src.PromptTokensDetails.AudioTokens
	}
	if src.PromptTokensDetails.ImageTokens > dst.PromptTokensDetails.ImageTokens {
		dst.PromptTokensDetails.ImageTokens = src.PromptTokensDetails.ImageTokens
	}
	if src.CompletionTokenDetails.ReasoningTokens > dst.CompletionTokenDetails.ReasoningTokens {
		dst.CompletionTokenDetails.ReasoningTokens = src.CompletionTokenDetails.ReasoningTokens
	}
	if src.CompletionTokenDetails.TextTokens > dst.CompletionTokenDetails.TextTokens {
		dst.CompletionTokenDetails.TextTokens = src.CompletionTokenDetails.TextTokens
	}
	if src.CompletionTokenDetails.AudioTokens > dst.CompletionTokenDetails.AudioTokens {
		dst.CompletionTokenDetails.AudioTokens = src.CompletionTokenDetails.AudioTokens
	}
	if src.CompletionTokenDetails.ImageTokens > dst.CompletionTokenDetails.ImageTokens {
		dst.CompletionTokenDetails.ImageTokens = src.CompletionTokenDetails.ImageTokens
	}
	if src.UsageSemantic != "" {
		dst.UsageSemantic = src.UsageSemantic
	}
	if src.UsageSource != "" {
		dst.UsageSource = src.UsageSource
	}
	mergeForcedBillingUsage(dst, src)
	if src.ClaudeCacheCreation5mTokens > dst.ClaudeCacheCreation5mTokens {
		dst.ClaudeCacheCreation5mTokens = src.ClaudeCacheCreation5mTokens
	}
	if src.ClaudeCacheCreation1hTokens > dst.ClaudeCacheCreation1hTokens {
		dst.ClaudeCacheCreation1hTokens = src.ClaudeCacheCreation1hTokens
	}
}

func mergeForcedBillingUsage(dst, src *dto.Usage) {
	if dst == nil || src == nil || src.BillingUsage == nil {
		return
	}
	if dst.BillingUsage == nil {
		dst.BillingUsage = dto.CloneBillingUsage(src.BillingUsage)
		return
	}

	dstBilling := dst.BillingUsage
	srcBilling := src.BillingUsage
	if dstBilling.ClaudeUsage != nil && srcBilling.ClaudeUsage != nil {
		mergeForcedClaudeUsage(dstBilling.ClaudeUsage, srcBilling.ClaudeUsage)
		return
	}
	if dstBilling.OpenAIUsage != nil && srcBilling.OpenAIUsage != nil {
		mergeForcedUsage(dstBilling.OpenAIUsage, srcBilling.OpenAIUsage)
		dstBilling.OpenAIUsage.BillingUsage = nil
		return
	}
	if dstBilling.GeminiUsageMetadata != nil && srcBilling.GeminiUsageMetadata != nil {
		mergeForcedGeminiUsage(dstBilling.GeminiUsageMetadata, srcBilling.GeminiUsageMetadata)
		return
	}
	dst.BillingUsage = dto.CloneBillingUsage(srcBilling)
}

func mergeForcedClaudeUsage(dst, src *dto.ClaudeUsage) {
	if dst == nil || src == nil {
		return
	}
	dst.InputTokens = max(dst.InputTokens, src.InputTokens)
	dst.OutputTokens = max(dst.OutputTokens, src.OutputTokens)
	dst.CacheReadInputTokens = max(dst.CacheReadInputTokens, src.CacheReadInputTokens)
	dst.CacheCreationInputTokens = max(dst.CacheCreationInputTokens, src.CacheCreationInputTokens)
	dst.ClaudeCacheCreation5mTokens = max(dst.ClaudeCacheCreation5mTokens, src.ClaudeCacheCreation5mTokens)
	dst.ClaudeCacheCreation1hTokens = max(dst.ClaudeCacheCreation1hTokens, src.ClaudeCacheCreation1hTokens)
	if src.CacheCreation != nil {
		if dst.CacheCreation == nil {
			dst.CacheCreation = &dto.ClaudeCacheCreationUsage{}
		}
		dst.CacheCreation.Ephemeral5mInputTokens = max(dst.CacheCreation.Ephemeral5mInputTokens, src.CacheCreation.Ephemeral5mInputTokens)
		dst.CacheCreation.Ephemeral1hInputTokens = max(dst.CacheCreation.Ephemeral1hInputTokens, src.CacheCreation.Ephemeral1hInputTokens)
	}
	if src.ServerToolUse != nil {
		if dst.ServerToolUse == nil {
			dst.ServerToolUse = &dto.ClaudeServerToolUse{}
		}
		dst.ServerToolUse.WebSearchRequests = max(dst.ServerToolUse.WebSearchRequests, src.ServerToolUse.WebSearchRequests)
	}
}

func mergeForcedGeminiUsage(dst, src *dto.GeminiUsageMetadata) {
	if dst == nil || src == nil {
		return
	}
	dst.PromptTokenCount = max(dst.PromptTokenCount, src.PromptTokenCount)
	dst.ToolUsePromptTokenCount = max(dst.ToolUsePromptTokenCount, src.ToolUsePromptTokenCount)
	dst.CandidatesTokenCount = max(dst.CandidatesTokenCount, src.CandidatesTokenCount)
	dst.ThoughtsTokenCount = max(dst.ThoughtsTokenCount, src.ThoughtsTokenCount)
	dst.CachedContentTokenCount = max(dst.CachedContentTokenCount, src.CachedContentTokenCount)
	dst.TotalTokenCount = max(dst.TotalTokenCount, src.TotalTokenCount)
	if len(src.PromptTokensDetails) > 0 {
		dst.PromptTokensDetails = append([]dto.GeminiPromptTokensDetails{}, src.PromptTokensDetails...)
	}
	if len(src.ToolUsePromptTokensDetails) > 0 {
		dst.ToolUsePromptTokensDetails = append([]dto.GeminiPromptTokensDetails{}, src.ToolUsePromptTokensDetails...)
	}
	if len(src.CandidatesTokensDetails) > 0 {
		dst.CandidatesTokensDetails = append([]dto.GeminiPromptTokensDetails{}, src.CandidatesTokensDetails...)
	}
}

func finalizeForcedUsage(c *gin.Context, usage *dto.Usage, outputText, model string, promptTokens int) *dto.Usage {
	if usage == nil {
		usage = &dto.Usage{}
	}
	if !service.ValidUsage(usage) || usage.PromptTokens == 0 || usage.CompletionTokens == 0 {
		estimate := service.ResponseText2Usage(c, outputText, model, promptTokens)
		if usage.PromptTokens == 0 {
			usage.PromptTokens = estimate.PromptTokens
		}
		if usage.CompletionTokens == 0 && outputText != "" {
			usage.CompletionTokens = estimate.CompletionTokens
		}
	}
	usage.TotalTokens = max(usage.TotalTokens, usage.PromptTokens+usage.CompletionTokens)
	return usage
}

func forcedUsageNeedsFallback(usage *dto.Usage, outputText string) bool {
	return !service.ValidUsage(usage) || usage.PromptTokens == 0 || (outputText != "" && usage.CompletionTokens == 0)
}

type forcedResponseObserver struct {
	usage                   forcedUsageAccumulator
	outputText              strings.Builder
	seenToolCalls           map[string]struct{}
	openAIStreamToolNames   map[string]string
	imageCounter            relaycommon.ImageGenerationCallCounter
	responsesImagesObserved bool
	responsesNonBillable    bool
	streamChunks            int
}

func (o *forcedResponseObserver) observe(c *gin.Context, info *relaycommon.RelayInfo, source types.RelayFormat, value any, stream bool) {
	if o == nil || info == nil {
		return
	}
	o.usage.add(forcedUsageFromResponse(value))
	switch response := value.(type) {
	case *dto.OpenAITextResponse:
		o.observeOpenAIResponse(c, info, response)
	case *dto.ChatCompletionsStreamResponse:
		o.observeOpenAIStreamResponse(c, info, response)
	case *dto.OpenAIResponsesResponse:
		o.observeResponsesResponse(info, response)
	case *dto.ResponsesStreamResponse:
		o.observeResponsesStreamResponse(info, response)
	case *dto.ClaudeResponse:
		o.observeClaudeResponse(c, info, response, stream)
	case *dto.GeminiChatResponse:
		o.observeGeminiResponse(c, info, response, stream)
	}
	if stream {
		o.streamChunks++
	}
}

func forcedUsageFromResponse(value any) *dto.Usage {
	switch response := value.(type) {
	case *dto.OpenAITextResponse:
		return relayconvert.UsageFromChatUsage(&response.Usage)
	case *dto.ChatCompletionsStreamResponse:
		return relayconvert.UsageFromChatUsage(response.Usage)
	case *dto.OpenAIResponsesResponse:
		return relayconvert.UsageFromResponsesUsage(response.Usage)
	case *dto.ResponsesStreamResponse:
		if response.Response != nil {
			return relayconvert.UsageFromResponsesUsage(response.Response.Usage)
		}
	case *dto.ClaudeResponse:
		if response.Usage != nil {
			return relayconvert.UsageFromClaudeAPIUsage(response.Usage)
		}
		if response.Message != nil && response.Message.Usage != nil {
			return relayconvert.UsageFromClaudeAPIUsage(response.Message.Usage)
		}
	case *dto.GeminiChatResponse:
		return relayconvert.UsageFromGeminiMetadata(response.GetUsageMetadata(), 0)
	}
	return nil
}

func (o *forcedResponseObserver) markTool(info *relaycommon.RelayInfo, key, itemType, name string) {
	if o.seenToolCalls == nil {
		o.seenToolCalls = make(map[string]struct{})
	}
	if _, exists := o.seenToolCalls[key]; exists {
		return
	}
	o.seenToolCalls[key] = struct{}{}
	info.CountBillableToolCall(itemType, name)
}

func (o *forcedResponseObserver) observeOpenAIResponse(c *gin.Context, info *relaycommon.RelayInfo, response *dto.OpenAITextResponse) {
	if response == nil {
		return
	}
	for choiceIndex, choice := range response.Choices {
		if choice.FinishReason == constant.FinishReasonContentFilter {
			common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "openai_finish_reason=content_filter")
		}
		o.outputText.WriteString(choice.Message.StringContent())
		o.outputText.WriteString(choice.Message.GetReasoningContent())
		for toolIndex, tool := range choice.Message.ParseToolCalls() {
			o.outputText.WriteString(tool.Function.Name)
			o.outputText.WriteString(tool.Function.Arguments)
			o.markTool(info, fmt.Sprintf("openai:%d:%d", choiceIndex, toolIndex), dto.BuildInCallFunctionCall, tool.Function.Name)
		}
	}
}

func (o *forcedResponseObserver) observeOpenAIStreamResponse(c *gin.Context, info *relaycommon.RelayInfo, response *dto.ChatCompletionsStreamResponse) {
	if response == nil {
		return
	}
	for _, choice := range response.Choices {
		if choice.FinishReason != nil && *choice.FinishReason == constant.FinishReasonContentFilter {
			common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "openai_finish_reason=content_filter")
		}
		o.outputText.WriteString(choice.Delta.GetContentString())
		o.outputText.WriteString(choice.Delta.GetReasoningContent())
		for toolIndex, tool := range choice.Delta.ToolCalls {
			o.outputText.WriteString(tool.Function.Name)
			o.outputText.WriteString(tool.Function.Arguments)
			index := toolIndex
			if tool.Index != nil {
				index = *tool.Index
			}
			key := fmt.Sprintf("openai:%d:%d", choice.Index, index)
			if o.openAIStreamToolNames == nil {
				o.openAIStreamToolNames = make(map[string]string)
			}
			if tool.Function.Name != "" {
				o.openAIStreamToolNames[key] = tool.Function.Name
			}
		}
	}
}

func (o *forcedResponseObserver) observeResponsesOutput(info *relaycommon.RelayInfo, output *dto.ResponsesOutput, index int, includeOutputText bool) {
	if output == nil {
		return
	}
	key := output.ID
	if key == "" {
		key = output.CallId
	}
	if key == "" {
		key = fmt.Sprintf("index:%d", index)
	}
	switch output.Type {
	case dto.BuildInCallWebSearchCall:
		o.markTool(info, "responses:web:"+key, dto.BuildInCallWebSearchCall, "")
	case dto.BuildInCallFileSearchCall:
		o.markTool(info, "responses:file:"+key, dto.BuildInCallFileSearchCall, "")
	case dto.BuildInCallFunctionCall:
		if includeOutputText {
			o.outputText.WriteString(output.Name)
			o.outputText.Write(output.Arguments)
		}
		o.markTool(info, "responses:function:"+key, dto.BuildInCallFunctionCall, output.Name)
	case dto.ResponsesOutputTypeImageGenerationCall:
		o.responsesImagesObserved = true
		o.imageCounter.Observe(output, &index)
	}
	if includeOutputText {
		for _, content := range output.Content {
			o.outputText.WriteString(content.Text)
		}
	}
}

func (o *forcedResponseObserver) observeResponsesResponse(info *relaycommon.RelayInfo, response *dto.OpenAIResponsesResponse) {
	if response == nil {
		return
	}
	if relaycommon.IsNonBillableResponsesStatus(response.Status) {
		o.responsesNonBillable = true
	}
	for index := range response.Output {
		o.observeResponsesOutput(info, &response.Output[index], index, true)
	}
}

func (o *forcedResponseObserver) observeResponsesStreamResponse(info *relaycommon.RelayInfo, response *dto.ResponsesStreamResponse) {
	if response == nil {
		return
	}
	switch response.Type {
	case "response.output_text.delta", "response.reasoning_summary_text.delta", "response.reasoning_text.delta",
		"response.function_call_arguments.delta", "response.custom_tool_call_input.delta":
		o.outputText.WriteString(response.Delta)
	case dto.ResponsesOutputTypeItemDone:
		index := 0
		if response.OutputIndex != nil {
			index = *response.OutputIndex
		}
		// A Responses stream already emitted its textual content through delta
		// events. Item-done and terminal snapshots are still needed for tool and
		// image accounting, but including their full payload would double the
		// text used by the no-usage token estimator.
		o.observeResponsesOutput(info, response.Item, index, false)
	case "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		o.responsesNonBillable = true
	}
	if response.Response != nil {
		if relaycommon.IsNonBillableResponsesStatus(response.Response.Status) {
			o.responsesNonBillable = true
		}
		for index := range response.Response.Output {
			o.observeResponsesOutput(info, &response.Response.Output[index], index, false)
		}
	}
}

func (o *forcedResponseObserver) observeClaudeResponse(c *gin.Context, info *relaycommon.RelayInfo, response *dto.ClaudeResponse, stream bool) {
	if response == nil {
		return
	}
	if response.StopReason == "refusal" || (response.Delta != nil && response.Delta.StopReason != nil && *response.Delta.StopReason == "refusal") {
		common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "claude_stop_reason=refusal")
	}
	if response.Usage != nil && response.Usage.ServerToolUse != nil {
		c.Set("claude_web_search_requests", max(c.GetInt("claude_web_search_requests"), response.Usage.ServerToolUse.WebSearchRequests))
	}
	if response.Message != nil && response.Message.Usage != nil && response.Message.Usage.ServerToolUse != nil {
		c.Set("claude_web_search_requests", max(c.GetInt("claude_web_search_requests"), response.Message.Usage.ServerToolUse.WebSearchRequests))
	}
	for index := range response.Content {
		block := &response.Content[index]
		o.observeClaudeBlock(info, block, fmt.Sprintf("content:%d", index))
	}
	if response.ContentBlock != nil {
		o.observeClaudeBlock(info, response.ContentBlock, fmt.Sprintf("stream:%d", response.GetIndex()))
	}
	if response.Delta != nil {
		if response.Delta.Text != nil {
			o.outputText.WriteString(*response.Delta.Text)
		}
		if response.Delta.Thinking != nil {
			o.outputText.WriteString(*response.Delta.Thinking)
		}
		if response.Delta.PartialJson != nil {
			o.outputText.WriteString(*response.Delta.PartialJson)
		}
	}
	if !stream {
		o.outputText.WriteString(response.Completion)
	}
}

func (o *forcedResponseObserver) observeClaudeBlock(info *relaycommon.RelayInfo, block *dto.ClaudeMediaMessage, key string) {
	if block == nil {
		return
	}
	if block.Text != nil {
		o.outputText.WriteString(*block.Text)
	}
	if block.Thinking != nil {
		o.outputText.WriteString(*block.Thinking)
	}
	if block.Type == "tool_use" {
		o.outputText.WriteString(block.Name)
		if input, err := common.Marshal(block.Input); err == nil {
			o.outputText.Write(input)
		}
		identity := block.Id
		if identity == "" {
			identity = key
		}
		o.markTool(info, "claude:"+identity, dto.BuildInCallToolUse, block.Name)
	}
}

func (o *forcedResponseObserver) observeGeminiResponse(c *gin.Context, info *relaycommon.RelayInfo, response *dto.GeminiChatResponse, stream bool) {
	if response == nil {
		return
	}
	for candidateIndex := range response.Candidates {
		candidate := &response.Candidates[candidateIndex]
		if candidate.GroundingMetadata != nil && len(candidate.GroundingMetadata.WebSearchQueries) > 0 {
			c.Set("gemini_google_search_call", true)
		}
		for partIndex := range candidate.Content.Parts {
			part := &candidate.Content.Parts[partIndex]
			o.outputText.WriteString(part.Text)
			if part.FunctionCall == nil {
				continue
			}
			o.outputText.WriteString(part.FunctionCall.FunctionName)
			if arguments, err := common.Marshal(part.FunctionCall.Arguments); err == nil {
				o.outputText.Write(arguments)
			}
			key := fmt.Sprintf("gemini:%d:%d:%s", candidate.Index, partIndex, part.FunctionCall.FunctionName)
			if !stream {
				key = fmt.Sprintf("gemini:%d:%d:%s", candidateIndex, partIndex, part.FunctionCall.FunctionName)
			}
			o.markTool(info, key, dto.BuildInCallFunctionCall, part.FunctionCall.FunctionName)
		}
	}
}

func (o *forcedResponseObserver) finish(info *relaycommon.RelayInfo) {
	if o == nil || info == nil {
		return
	}
	for key, name := range o.openAIStreamToolNames {
		o.markTool(info, key, dto.BuildInCallFunctionCall, name)
	}
	if !o.responsesImagesObserved {
		return
	}
	if o.responsesNonBillable {
		o.imageCounter.Reset()
	}
	o.imageCounter.Commit(info)
}

func refreshForcedBillingUsage(source types.RelayFormat, usage *dto.Usage, estimated bool) {
	if usage == nil {
		return
	}
	switch source {
	case types.RelayFormatOpenAI:
		raw := *usage
		raw.InputTokens = max(raw.InputTokens, raw.PromptTokens)
		raw.OutputTokens = max(raw.OutputTokens, raw.CompletionTokens)
		raw.BillingUsage = nil
		usage.BillingUsage = dto.NewOpenAIChatBillingUsage(&raw)
	case types.RelayFormatOpenAIResponses:
		raw := *usage
		raw.InputTokens = max(raw.InputTokens, raw.PromptTokens)
		raw.OutputTokens = max(raw.OutputTokens, raw.CompletionTokens)
		details := raw.PromptTokensDetails
		raw.InputTokensDetails = &details
		raw.BillingUsage = nil
		usage.BillingUsage = dto.NewOpenAIResponsesBillingUsage(&raw)
	case types.RelayFormatClaude:
		raw := &dto.ClaudeUsage{}
		if usage.BillingUsage != nil && usage.BillingUsage.ClaudeUsage != nil {
			cloned := dto.CloneBillingUsage(usage.BillingUsage)
			raw = cloned.ClaudeUsage
		}
		cacheCreation := max(usage.PromptTokensDetails.CachedCreationTokens, usage.PromptTokensDetails.CacheWriteTokens)
		cacheCreation = max(cacheCreation, usage.ClaudeCacheCreation5mTokens+usage.ClaudeCacheCreation1hTokens)
		raw.CacheReadInputTokens = max(raw.CacheReadInputTokens, usage.PromptTokensDetails.CachedTokens)
		raw.CacheCreationInputTokens = max(raw.CacheCreationInputTokens, cacheCreation)
		if raw.InputTokens == 0 {
			raw.InputTokens = max(usage.PromptTokens-raw.CacheReadInputTokens-raw.CacheCreationInputTokens, 0)
		}
		raw.OutputTokens = max(raw.OutputTokens, usage.CompletionTokens)
		if usage.ClaudeCacheCreation5mTokens > 0 || usage.ClaudeCacheCreation1hTokens > 0 {
			if raw.CacheCreation == nil {
				raw.CacheCreation = &dto.ClaudeCacheCreationUsage{}
			}
			raw.CacheCreation.Ephemeral5mInputTokens = max(raw.CacheCreation.Ephemeral5mInputTokens, usage.ClaudeCacheCreation5mTokens)
			raw.CacheCreation.Ephemeral1hInputTokens = max(raw.CacheCreation.Ephemeral1hInputTokens, usage.ClaudeCacheCreation1hTokens)
		}
		usage.BillingUsage = dto.NewClaudeMessagesBillingUsage(raw)
	case types.RelayFormatGemini:
		metadata := &dto.GeminiUsageMetadata{}
		if usage.BillingUsage != nil && usage.BillingUsage.GeminiUsageMetadata != nil {
			cloned := dto.CloneBillingUsage(usage.BillingUsage)
			metadata = cloned.GeminiUsageMetadata
		}
		metadata.PromptTokenCount = max(metadata.PromptTokenCount, usage.PromptTokens-metadata.ToolUsePromptTokenCount)
		metadata.CandidatesTokenCount = max(metadata.CandidatesTokenCount, usage.CompletionTokens-metadata.ThoughtsTokenCount)
		metadata.CachedContentTokenCount = max(metadata.CachedContentTokenCount, usage.PromptTokensDetails.CachedTokens)
		metadata.TotalTokenCount = max(metadata.TotalTokenCount, usage.TotalTokens)
		usage.BillingUsage = dto.NewGeminiChatBillingUsage(metadata)
	}
	if estimated && usage.BillingUsage != nil {
		usage.BillingUsage.Estimated = true
	}
}

func validateForcedGeminiResponse(c *gin.Context, response *dto.GeminiChatResponse, firstStreamChunk bool) *types.NewAPIError {
	if response == nil || len(response.Candidates) > 0 {
		return nil
	}
	if response.PromptFeedback != nil && response.PromptFeedback.BlockReason != nil {
		reason := *response.PromptFeedback.BlockReason
		common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "gemini_block_reason="+reason)
		return types.NewOpenAIError(
			errors.New("request blocked by Gemini API: "+reason),
			types.ErrorCodePromptBlocked,
			http.StatusBadRequest,
		)
	}
	if !firstStreamChunk {
		return nil
	}
	common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "gemini_empty_candidates")
	return types.NewOpenAIError(
		errors.New("empty response from Gemini API"),
		types.ErrorCodeEmptyResponse,
		http.StatusInternalServerError,
	)
}

func markForcedOutboundResponseStarted(c *gin.Context) {
	if c != nil {
		c.Set(forcedOutboundResponseStartedKey, true)
	}
}

func protectForcedOutboundStreamError(c *gin.Context, err *types.NewAPIError) *types.NewAPIError {
	if err == nil || c == nil || !c.GetBool(forcedOutboundResponseStartedKey) {
		return err
	}
	return types.NewError(err, err.GetErrorCode(), types.ErrOptionWithSkipRetry())
}

func (a *forcedOutboundAdaptor) Init(info *relaycommon.RelayInfo) {
	if info != nil {
		info.FinalRequestRelayFormat = a.target
	}
}

func (a *forcedOutboundAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil {
		return "", errors.New("relay info is nil")
	}

	var path string
	switch a.target {
	case types.RelayFormatOpenAI:
		path = "/v1/chat/completions"
	case types.RelayFormatOpenAIResponses:
		path = "/v1/responses"
	case types.RelayFormatClaude:
		path = "/v1/messages"
	case types.RelayFormatGemini:
		model := strings.TrimPrefix(info.UpstreamModelName, "models/")
		action := "generateContent"
		if info.IsStream {
			action = "streamGenerateContent?alt=sse"
			if info.RelayMode == relayconstant.RelayModeGemini {
				info.DisablePing = true
			}
		}
		path = fmt.Sprintf("/v1beta/models/%s:%s", model, action)
	default:
		return "", fmt.Errorf("unsupported forced outbound format %q", a.target)
	}

	return relaycommon.GetFullRequestURL(strings.TrimRight(info.ChannelBaseUrl, "/"), path, info.ChannelType), nil
}

func (a *forcedOutboundAdaptor) SetupRequestHeader(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error {
	if info == nil {
		return errors.New("relay info is nil")
	}
	channel.SetupApiRequestHeader(info, c, header)
	header.Set("Authorization", "Bearer "+info.ApiKey)
	if info.Organization != "" {
		header.Set("OpenAI-Organization", info.Organization)
	}
	if a.target == types.RelayFormatClaude {
		anthropicVersion := c.GetHeader("anthropic-version")
		if anthropicVersion == "" {
			anthropicVersion = "2023-06-01"
		}
		header.Set("anthropic-version", anthropicVersion)
		claudechannel.CommonClaudeHeadersOperation(c, header, info)
	}
	return nil
}

func (a *forcedOutboundAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return convertForcedOutboundRequest(c, info, a.target, request)
}

func (a *forcedOutboundAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return convertForcedOutboundRequest(c, info, a.target, &request)
}

func (a *forcedOutboundAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return convertForcedOutboundRequest(c, info, a.target, request)
}

func (a *forcedOutboundAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return convertForcedOutboundRequest(c, info, a.target, request)
}

func (a *forcedOutboundAdaptor) ConvertRerankRequest(*gin.Context, int, dto.RerankRequest) (any, error) {
	return nil, errors.New("forced outbound protocol does not support rerank requests")
}

func (a *forcedOutboundAdaptor) ConvertEmbeddingRequest(*gin.Context, *relaycommon.RelayInfo, dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("forced outbound protocol does not support embedding requests")
}

func (a *forcedOutboundAdaptor) ConvertAudioRequest(*gin.Context, *relaycommon.RelayInfo, dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("forced outbound protocol does not support audio requests")
}

func (a *forcedOutboundAdaptor) ConvertImageRequest(*gin.Context, *relaycommon.RelayInfo, dto.ImageRequest) (any, error) {
	return nil, errors.New("forced outbound protocol does not support image requests")
}

func (a *forcedOutboundAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *forcedOutboundAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	if info == nil {
		return nil, types.NewError(errors.New("relay info is nil"), types.ErrorCodeBadResponseBody)
	}
	if resp == nil || resp.Body == nil {
		return nil, types.NewError(errors.New("upstream response is nil"), types.ErrorCodeBadResponseBody)
	}

	if info.IsStream {
		return forcedOutboundStreamResponse(c, resp, info, a.target)
	}
	return forcedOutboundNonStreamResponse(c, resp, info, a.target)
}

func (a *forcedOutboundAdaptor) GetModelList() []string { return nil }

func (a *forcedOutboundAdaptor) GetChannelName() string { return "forced-outbound" }

func convertForcedOutboundRequest(c *gin.Context, info *relaycommon.RelayInfo, target types.RelayFormat, request any) (any, error) {
	result, err := relayconvert.ConvertRequest(c, info, target, request)
	if err != nil {
		return nil, err
	}

	value := result.Value
	switch target {
	case types.RelayFormatOpenAI:
		typed, ok := value.(*dto.GeneralOpenAIRequest)
		if !ok {
			return nil, fmt.Errorf("expected OpenAI chat completions request, got %T", value)
		}
		// The forced protocol is standard OpenAI even when the physical channel
		// is New API. Normalize with OpenAI semantics without permanently
		// changing the selected channel identity.
		savedChannelType := info.ChannelType
		info.ChannelType = constant.ChannelTypeOpenAI
		value, err = (&openaichannel.Adaptor{}).ConvertOpenAIRequest(c, info, typed)
		info.ChannelType = savedChannelType
	case types.RelayFormatOpenAIResponses:
		typed, ok := value.(*dto.OpenAIResponsesRequest)
		if !ok {
			return nil, fmt.Errorf("expected OpenAI responses request, got %T", value)
		}
		value, err = (&openaichannel.Adaptor{}).ConvertOpenAIResponsesRequest(c, info, *typed)
	case types.RelayFormatClaude:
		if _, ok := value.(*dto.ClaudeRequest); !ok {
			return nil, fmt.Errorf("expected Anthropic messages request, got %T", value)
		}
	case types.RelayFormatGemini:
		typed, ok := value.(*dto.GeminiChatRequest)
		if !ok {
			return nil, fmt.Errorf("expected Gemini generateContent request, got %T", value)
		}
		value, err = (&geminichannel.Adaptor{}).ConvertGeminiRequest(c, info, typed)
	default:
		return nil, fmt.Errorf("unsupported forced outbound format %q", target)
	}
	if err != nil {
		return nil, err
	}
	info.FinalRequestRelayFormat = target
	return value, nil
}

func isForcedOutboundTextRequest(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	switch info.RelayFormat {
	case types.RelayFormatOpenAI:
		return info.RelayMode == relayconstant.RelayModeChatCompletions
	case types.RelayFormatOpenAIResponses:
		return info.RelayMode == relayconstant.RelayModeResponses
	case types.RelayFormatClaude:
		return info.RelayMode != relayconstant.RelayModeResponsesCompact &&
			info.RelayMode != relayconstant.RelayModeAlphaSearch
	case types.RelayFormatGemini:
		return info.RelayMode == relayconstant.RelayModeGemini
	default:
		return false
	}
}

func forcedOutboundTarget(info *relaycommon.RelayInfo) (types.RelayFormat, bool, error) {
	if info == nil || !isForcedOutboundTextRequest(info) {
		return "", false, nil
	}
	target := info.ChannelSetting.ForcedOutboundFormat
	if target == "" {
		return "", false, nil
	}
	if !common.ChannelTypeSupportsForcedOutboundFormat(info.ChannelType, target) {
		return "", true, fmt.Errorf("channel type %d does not support forced outbound format %q", info.ChannelType, target)
	}
	return target, true, nil
}

// PrepareForcedOutboundRequest is shared by normal relay traffic and channel
// tests. Callers must perform model mapping before invoking it.
func PrepareForcedOutboundRequest(c *gin.Context, info *relaycommon.RelayInfo, request any) (channel.Adaptor, []byte, bool, error) {
	target, enabled, err := forcedOutboundTarget(info)
	if err != nil || !enabled {
		return nil, nil, enabled, err
	}
	info.AppliedForcedOutboundFormat = target

	adaptor := &forcedOutboundAdaptor{target: target}
	adaptor.Init(info)
	convertedRequest, err := convertForcedOutboundRequest(c, info, target, request)
	if err != nil {
		return nil, nil, true, err
	}
	if err := applyForcedOutboundSystemPrompt(c, info, convertedRequest); err != nil {
		return nil, nil, true, err
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return nil, nil, true, err
	}
	jsonData, err = relaycommon.RemoveDisabledFieldsForced(jsonData, info.ChannelOtherSettings)
	if err != nil {
		return nil, nil, true, err
	}
	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return nil, nil, true, err
		}
	}

	logger.LogDebug(c, "forced outbound request: inbound=%s target=%s chain=%v body=%s", info.RelayFormat, target, info.RequestConversionChain, jsonData)
	return adaptor, jsonData, true, nil
}

func tryForcedOutbound(c *gin.Context, info *relaycommon.RelayInfo, request any) (*dto.Usage, bool, *types.NewAPIError) {
	adaptor, jsonData, enabled, err := PrepareForcedOutboundRequest(c, info, request)
	if !enabled {
		return nil, false, nil
	}
	if err != nil {
		if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
			return nil, true, relaycommon.NewAPIErrorFromParamOverride(fixedErr)
		}
		return nil, true, types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}

	body, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return nil, true, types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}
	defer closer.Close()

	respAny, err := adaptor.DoRequest(c, info, body)
	if err != nil {
		return nil, true, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	resp, ok := respAny.(*http.Response)
	if !ok || resp == nil {
		return nil, true, types.NewError(errors.New("invalid upstream response"), types.ErrorCodeBadResponse)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")
	upstreamStream := strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream")
	if upstreamStream != info.IsStream {
		service.CloseResponseBodyGracefully(resp)
		return nil, true, types.NewError(
			fmt.Errorf("forced outbound stream mismatch: request stream=%t, upstream content-type=%q", info.IsStream, resp.Header.Get("Content-Type")),
			types.ErrorCodeBadResponse,
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		newAPIError := service.RelayErrorHandler(c.Request.Context(), resp, false)
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return nil, true, newAPIError
	}

	usageAny, newAPIError := adaptor.DoResponse(c, resp, info)
	if newAPIError != nil {
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return nil, true, protectForcedOutboundStreamError(c, newAPIError)
	}
	usage, ok := usageAny.(*dto.Usage)
	if !ok || usage == nil {
		return nil, true, types.NewError(fmt.Errorf("invalid forced outbound usage type %T", usageAny), types.ErrorCodeBadResponseBody)
	}
	return usage, true, nil
}

func forcedOutboundNonStreamResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, source types.RelayFormat) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeReadResponseBodyFailed)
	}
	logger.LogDebug(c, "forced outbound response: source=%s body=%s", source, responseBody)

	parsed, err := parseForcedOutboundResponse(source, responseBody, false)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	if source == types.RelayFormatGemini {
		geminiResponse, ok := parsed.(*dto.GeminiChatResponse)
		if !ok {
			return nil, types.NewError(fmt.Errorf("expected Gemini response, got %T", parsed), types.ErrorCodeBadResponseBody)
		}
		if newAPIError := validateForcedGeminiResponse(c, geminiResponse, true); newAPIError != nil {
			return nil, newAPIError
		}
	}
	observer := &forcedResponseObserver{}
	observer.observe(c, info, source, parsed, false)
	observer.finish(info)
	result, err := relayconvert.ConvertResponse(c, info, info.RelayFormat, parsed)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	observer.usage.add(result.Usage)
	needsFallback := forcedUsageNeedsFallback(observer.usage.usage, observer.outputText.String())
	usage := finalizeForcedUsage(c, observer.usage.usage, observer.outputText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	refreshForcedBillingUsage(source, usage, needsFallback)
	output, err := common.Marshal(result.Value)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	service.IOCopyBytesGracefully(c, resp, output)
	return usage, nil
}

func forcedOutboundStreamResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, source types.RelayFormat) (*dto.Usage, *types.NewAPIError) {
	responseID := helper.GetResponseID(c)
	created := common.GetTimestamp()
	state, err := relayconvert.NewResponseStreamState(source, info.RelayFormat, relayconvert.ResponseStreamOptions{
		ID:           responseID,
		Model:        info.UpstreamModelName,
		Created:      created,
		IncludeUsage: info.ShouldIncludeUsage,
	})
	if err != nil {
		service.CloseResponseBodyGracefully(resp)
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	observer := &forcedResponseObserver{}
	var streamErr error
	var streamAPIError *types.NewAPIError
	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		parsed, parseErr := parseForcedOutboundResponse(source, []byte(data), true)
		if parseErr != nil {
			streamErr = parseErr
			sr.Stop(parseErr)
			return
		}
		if source == types.RelayFormatGemini {
			geminiResponse, ok := parsed.(*dto.GeminiChatResponse)
			if !ok {
				streamErr = fmt.Errorf("expected Gemini stream response, got %T", parsed)
				sr.Stop(streamErr)
				return
			}
			if newAPIError := validateForcedGeminiResponse(c, geminiResponse, observer.streamChunks == 0); newAPIError != nil {
				streamAPIError = newAPIError
				sr.Stop(newAPIError)
				return
			}
		}
		observer.observe(c, info, source, parsed, true)
		results, convertErr := relayconvert.ConvertStreamResponseChunk(c, info, state, parsed)
		if convertErr != nil {
			streamErr = convertErr
			sr.Stop(convertErr)
			return
		}
		for _, result := range results {
			observer.usage.add(result.Usage)
			if writeErr := writeForcedOutboundStreamResult(c, info.RelayFormat, result.Value); writeErr != nil {
				streamErr = writeErr
				sr.Stop(writeErr)
				return
			}
			markForcedOutboundResponseStarted(c)
		}
	})
	observer.finish(info)
	if streamAPIError != nil {
		return nil, protectForcedOutboundStreamError(c, streamAPIError)
	}
	if streamErr != nil {
		return nil, types.NewError(streamErr, types.ErrorCodeBadResponseBody)
	}

	observer.usage.add(state.Usage())
	needsFallback := forcedUsageNeedsFallback(observer.usage.usage, observer.outputText.String())
	usage := finalizeForcedUsage(c, observer.usage.usage, observer.outputText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	refreshForcedBillingUsage(source, usage, needsFallback)
	state.SetUsage(usage)
	finalResults, err := relayconvert.FinalizeStreamResponse(c, info, state)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	for _, result := range finalResults {
		observer.usage.add(result.Usage)
		if err := writeForcedOutboundStreamResult(c, info.RelayFormat, result.Value); err != nil {
			return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
		}
		markForcedOutboundResponseStarted(c)
	}
	if info.RelayFormat == types.RelayFormatOpenAI {
		if info.ShouldIncludeUsage && needsFallback {
			if err := helper.ObjectData(c, helper.GenerateFinalUsageResponse(responseID, created, info.UpstreamModelName, *usage)); err != nil {
				return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
			}
			markForcedOutboundResponseStarted(c)
		}
		helper.Done(c)
		markForcedOutboundResponseStarted(c)
	}
	return usage, nil
}

func parseForcedOutboundResponse(format types.RelayFormat, data []byte, stream bool) (any, error) {
	switch format {
	case types.RelayFormatOpenAI:
		if stream {
			var value dto.ChatCompletionsStreamResponse
			if err := common.Unmarshal(data, &value); err != nil {
				return nil, err
			}
			return &value, nil
		}
		var value dto.OpenAITextResponse
		if err := common.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		if apiErr := value.GetOpenAIError(); apiErr != nil && apiErr.Type != "" {
			return nil, fmt.Errorf("upstream OpenAI error: %s", apiErr.Message)
		}
		return &value, nil
	case types.RelayFormatOpenAIResponses:
		if stream {
			var value dto.ResponsesStreamResponse
			if err := common.Unmarshal(data, &value); err != nil {
				return nil, err
			}
			return &value, nil
		}
		var value dto.OpenAIResponsesResponse
		if err := common.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		if apiErr := value.GetOpenAIError(); apiErr != nil && apiErr.Type != "" {
			return nil, fmt.Errorf("upstream Responses error: %s", apiErr.Message)
		}
		return &value, nil
	case types.RelayFormatClaude:
		var value dto.ClaudeResponse
		if err := common.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		if apiErr := value.GetClaudeError(); apiErr != nil && apiErr.Type != "" {
			return nil, fmt.Errorf("upstream Anthropic error: %s", apiErr.Message)
		}
		return &value, nil
	case types.RelayFormatGemini:
		var value dto.GeminiChatResponse
		if err := common.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		return &value, nil
	default:
		return nil, fmt.Errorf("unsupported forced outbound response format %q", format)
	}
}

func writeForcedOutboundStreamResult(c *gin.Context, format types.RelayFormat, value any) error {
	switch format {
	case types.RelayFormatOpenAI, types.RelayFormatGemini:
		return helper.ObjectData(c, value)
	case types.RelayFormatOpenAIResponses:
		var response *dto.ResponsesStreamResponse
		switch typed := value.(type) {
		case *dto.ResponsesStreamResponse:
			response = typed
		case dto.ResponsesStreamResponse:
			response = &typed
		case *relayconvert.ChatToResponsesStreamEvent:
			if typed != nil {
				payload := typed.Payload
				if payload.Type == "" {
					payload.Type = typed.Type
				}
				response = &payload
			}
		case relayconvert.ChatToResponsesStreamEvent:
			payload := typed.Payload
			if payload.Type == "" {
				payload.Type = typed.Type
			}
			response = &payload
		}
		if response == nil {
			return fmt.Errorf("expected Responses stream event, got %T", value)
		}
		data, err := common.Marshal(response)
		if err != nil {
			return err
		}
		return helper.ResponseChunkData(c, *response, string(data))
	case types.RelayFormatClaude:
		response, ok := value.(*dto.ClaudeResponse)
		if !ok {
			if nonPointer, ok := value.(dto.ClaudeResponse); ok {
				response = &nonPointer
			}
		}
		if response == nil {
			return fmt.Errorf("expected Anthropic stream event, got %T", value)
		}
		return helper.ClaudeData(c, *response)
	default:
		return fmt.Errorf("unsupported client stream format %q", format)
	}
}

func applyForcedOutboundSystemPrompt(c *gin.Context, info *relaycommon.RelayInfo, request any) error {
	if info == nil || info.ChannelSetting.SystemPrompt == "" {
		return nil
	}
	switch typed := request.(type) {
	case *dto.GeneralOpenAIRequest:
		applySystemPromptIfNeeded(c, info, typed)
	case *dto.OpenAIResponsesRequest:
		return applyResponsesSystemPrompt(info, typed)
	case *dto.ClaudeRequest:
		applyClaudeSystemPrompt(c, info, typed)
	case *dto.GeminiChatRequest:
		applyGeminiSystemPrompt(c, info, typed)
	default:
		return fmt.Errorf("unsupported forced outbound request type %T", request)
	}
	return nil
}

func applyResponsesSystemPrompt(info *relaycommon.RelayInfo, request *dto.OpenAIResponsesRequest) error {
	if request == nil || info == nil || info.ChannelSetting.SystemPrompt == "" {
		return nil
	}
	if len(request.Instructions) == 0 {
		encoded, err := common.Marshal(info.ChannelSetting.SystemPrompt)
		if err != nil {
			return err
		}
		request.Instructions = encoded
		return nil
	}
	if !info.ChannelSetting.SystemPromptOverride {
		return nil
	}
	var existing string
	if err := common.Unmarshal(request.Instructions, &existing); err != nil {
		return fmt.Errorf("invalid Responses instructions: %w", err)
	}
	encoded, err := common.Marshal(info.ChannelSetting.SystemPrompt + "\n" + existing)
	if err != nil {
		return err
	}
	request.Instructions = encoded
	return nil
}

func applyClaudeSystemPrompt(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) {
	if request == nil || info == nil || info.ChannelSetting.SystemPrompt == "" {
		return
	}
	if request.System == nil {
		request.SetStringSystem(info.ChannelSetting.SystemPrompt)
		return
	}
	if !info.ChannelSetting.SystemPromptOverride {
		return
	}
	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, true)
	if request.IsStringSystem() {
		existing := strings.TrimSpace(request.GetStringSystem())
		if existing == "" {
			request.SetStringSystem(info.ChannelSetting.SystemPrompt)
		} else {
			request.SetStringSystem(info.ChannelSetting.SystemPrompt + "\n" + existing)
		}
		return
	}
	contents := request.ParseSystem()
	newSystem := dto.ClaudeMediaMessage{Type: dto.ContentTypeText}
	newSystem.SetText(info.ChannelSetting.SystemPrompt)
	request.System = append([]dto.ClaudeMediaMessage{newSystem}, contents...)
}

func applyGeminiSystemPrompt(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) {
	if request == nil || info == nil || info.ChannelSetting.SystemPrompt == "" {
		return
	}
	if request.SystemInstructions == nil {
		request.SystemInstructions = &dto.GeminiChatContent{Parts: []dto.GeminiPart{{Text: info.ChannelSetting.SystemPrompt}}}
		return
	}
	if len(request.SystemInstructions.Parts) == 0 {
		request.SystemInstructions.Parts = []dto.GeminiPart{{Text: info.ChannelSetting.SystemPrompt}}
		return
	}
	if !info.ChannelSetting.SystemPromptOverride {
		return
	}
	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, true)
	for i := range request.SystemInstructions.Parts {
		if request.SystemInstructions.Parts[i].Text == "" {
			continue
		}
		request.SystemInstructions.Parts[i].Text = info.ChannelSetting.SystemPrompt + "\n" + request.SystemInstructions.Parts[i].Text
		return
	}
	request.SystemInstructions.Parts = append([]dto.GeminiPart{{Text: info.ChannelSetting.SystemPrompt}}, request.SystemInstructions.Parts...)
}
