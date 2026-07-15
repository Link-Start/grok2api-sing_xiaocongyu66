package web

import (
	"testing"

	inferencedomain "github.com/chenyme/grok2api/backend/internal/domain/inference"
)

func TestEstimatePromptCacheTokensFirstTurn(t *testing.T) {
	input, cached := estimatePromptCacheTokens("hello world from user", nil)
	if input <= 0 || cached != 0 {
		t.Fatalf("first turn input=%d cached=%d", input, cached)
	}
}

func TestEstimatePromptCacheTokensMultiTurnUsesPriorUsage(t *testing.T) {
	previous := &inferencedomain.WebResponseState{
		ResponseJSON: `{"usage":{"input_tokens":100,"output_tokens":40,"input_tokens_details":{"cached_tokens":0}}}`,
	}
	input, cached := estimatePromptCacheTokens("follow up question", previous)
	if cached != 140 {
		t.Fatalf("cached = %d, want 140 (prior input+output)", cached)
	}
	if input <= cached {
		t.Fatalf("total input %d should exceed cached %d by current prompt", input, cached)
	}
}

func TestBuildOpenAIResultReportsCachedTokens(t *testing.T) {
	parsed := parsedChat{InputTokens: 200, CachedInputTokens: 150}
	parsed.Text.WriteString("answer text")

	responses := buildOpenAIResult("responses", "resp_1", "grok-chat-fast", parsed, false)
	usage := responses["usage"].(map[string]any)
	if usage["input_tokens"] != int64(200) {
		t.Fatalf("input_tokens = %#v", usage["input_tokens"])
	}
	details := usage["input_tokens_details"].(map[string]any)
	if details["cached_tokens"] != int64(150) {
		t.Fatalf("cached_tokens = %#v", details["cached_tokens"])
	}

	chat := buildOpenAIResult("chat", "resp_1", "grok-chat-fast", parsed, false)
	chatUsage := chat["usage"].(map[string]any)
	promptDetails := chatUsage["prompt_tokens_details"].(map[string]any)
	if promptDetails["cached_tokens"] != int64(150) {
		t.Fatalf("chat cached_tokens = %#v", promptDetails["cached_tokens"])
	}

	messages := buildOpenAIResult("messages", "resp_1", "grok-chat-fast", parsed, false)
	msgUsage := messages["usage"].(map[string]any)
	if msgUsage["cache_read_input_tokens"] != int64(150) {
		t.Fatalf("messages cache_read = %#v", msgUsage["cache_read_input_tokens"])
	}
}

func TestPriorConversationTokensFromNestedUsage(t *testing.T) {
	previous := &inferencedomain.WebResponseState{
		ResponseJSON: `{"response":{"usage":{"prompt_tokens":10,"completion_tokens":5}}}`,
	}
	if got := priorConversationTokens(previous); got != 15 {
		t.Fatalf("prior = %d", got)
	}
}
