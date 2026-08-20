package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/require"
)

func TestBuildTestRequestLongPrompt(t *testing.T) {
	request := buildTestRequest("gpt-4o-mini", "openai", nil, false, true)
	openAIRequest, ok := request.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, openAIRequest.Messages, 1)

	prompt, ok := openAIRequest.Messages[0].Content.(string)
	require.True(t, ok)
	require.GreaterOrEqual(t, strings.Count(prompt, "Please verify"), longTestPromptRepeatCount)
}

func TestBuildTestRequestDefaultPromptStaysShort(t *testing.T) {
	request := buildTestRequest("gpt-4o-mini", "openai", nil, false, false)
	openAIRequest, ok := request.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, openAIRequest.Messages, 1)
	require.Equal(t, "hi", openAIRequest.Messages[0].Content)
}
