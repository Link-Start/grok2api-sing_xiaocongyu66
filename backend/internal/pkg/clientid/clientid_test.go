package clientid

import "testing"

func TestDetectClaudeCode(t *testing.T) {
	if got := Detect("claude-code/2.1.0", nil); got != ClaudeCode {
		t.Fatalf("ua = %s", got)
	}
	if got := Detect("Mozilla/5.0", map[string]string{"x-claude-code-session-id": "sess"}); got != ClaudeCode {
		t.Fatalf("header = %s", got)
	}
}

func TestDetectCodex(t *testing.T) {
	if got := Detect("openai-codex/0.1", nil); got != Codex {
		t.Fatalf("ua = %s", got)
	}
	if got := Detect("", map[string]string{"x-codex-session-id": "x"}); got != Codex {
		t.Fatalf("header = %s", got)
	}
}

func TestDetectHermesOpenCodeGrok(t *testing.T) {
	if got := Detect("hermes-agent/1.0", nil); got != Hermes {
		t.Fatalf("hermes = %s", got)
	}
	if got := Detect("opencode/0.9", nil); got != OpenCode {
		t.Fatalf("opencode = %s", got)
	}
	if got := Detect("grok-cli/1.0", nil); got != GrokCLI {
		t.Fatalf("grok = %s", got)
	}
}

func TestDetectUnknown(t *testing.T) {
	if got := Detect("", nil); got != Unknown {
		t.Fatalf("empty = %s", got)
	}
	if got := Detect("SomeCustomBot/1.0", nil); got != Unknown {
		t.Fatalf("custom = %s", got)
	}
}
