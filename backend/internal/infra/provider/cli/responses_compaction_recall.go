package cli

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// gatewayCompactRecall remembers gateway-emulated compact responses so a later
// previous_response_id can be rewritten into portable summary input instead of
// being forwarded to Grok (which never stored the synthetic compact response).
type gatewayCompactRecall struct {
	mu      sync.Mutex
	entries map[string]gatewayCompactEntry
	ttl     time.Duration
}

type gatewayCompactEntry struct {
	summary string
	session string
	expires time.Time
}

func newGatewayCompactRecall() *gatewayCompactRecall {
	return &gatewayCompactRecall{
		entries: make(map[string]gatewayCompactEntry),
		ttl:     24 * time.Hour,
	}
}

func (r *gatewayCompactRecall) remember(responseID, session, summary string) {
	if r == nil {
		return
	}
	responseID = strings.TrimSpace(responseID)
	summary = strings.TrimSpace(summary)
	if responseID == "" || summary == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entries == nil {
		r.entries = make(map[string]gatewayCompactEntry)
	}
	// Bound memory: drop expired entries opportunistically.
	now := time.Now().UTC()
	if len(r.entries) > 4096 {
		for id, entry := range r.entries {
			if now.After(entry.expires) {
				delete(r.entries, id)
			}
		}
	}
	r.entries[responseID] = gatewayCompactEntry{
		summary: summary,
		session: strings.TrimSpace(session),
		expires: now.Add(r.ttl),
	}
}

func (r *gatewayCompactRecall) lookup(responseID string) (summary, session string, ok bool) {
	if r == nil {
		return "", "", false
	}
	responseID = strings.TrimSpace(responseID)
	if responseID == "" {
		return "", "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, found := r.entries[responseID]
	if !found {
		return "", "", false
	}
	if time.Now().UTC().After(entry.expires) {
		delete(r.entries, responseID)
		return "", "", false
	}
	return entry.summary, entry.session, true
}

// resolveGatewayPreviousResponse rewrites previous_response_id that points at a
// gateway-emulated compact response into a portable user summary message.
// Grok Build never saw that response id (store=false sampling + synthetic body).
func resolveGatewayPreviousResponse(body []byte, recall *gatewayCompactRecall) ([]byte, bool) {
	if len(body) == 0 || recall == nil {
		return body, false
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return body, false
	}
	prev, _ := payload["previous_response_id"].(string)
	prev = strings.TrimSpace(prev)
	if prev == "" {
		prev, _ = payload["previousResponseId"].(string)
		prev = strings.TrimSpace(prev)
	}
	if prev == "" {
		return body, false
	}
	summary, _, ok := recall.lookup(prev)
	if !ok {
		return body, false
	}
	delete(payload, "previous_response_id")
	delete(payload, "previousResponseId")
	items, _ := payload["input"].([]any)
	prefix := gatewayCompactionSummaryMessage(summary)
	if items == nil {
		if raw, ok := payload["input"].(string); ok && strings.TrimSpace(raw) != "" {
			items = []any{prefix, map[string]any{"type": "message", "role": "user", "content": raw}}
		} else {
			items = []any{prefix}
		}
	} else {
		items = append([]any{prefix}, items...)
	}
	payload["input"] = items
	encoded, err := json.Marshal(payload)
	if err != nil {
		return body, false
	}
	slog.Warn("compaction_previous_response_expanded", "response_id_set", true)
	return encoded, true
}

// stripPreviousResponseID removes previous_response_id for a one-shot retry
// after Grok rejects compact state.
func stripPreviousResponseID(body []byte) ([]byte, bool) {
	if len(body) == 0 {
		return body, false
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return body, false
	}
	_, a := payload["previous_response_id"]
	_, b := payload["previousResponseId"]
	if !a && !b {
		return body, false
	}
	delete(payload, "previous_response_id")
	delete(payload, "previousResponseId")
	encoded, err := json.Marshal(payload)
	if err != nil {
		return body, false
	}
	return encoded, true
}

// sanitizeBodyAfterCompactionDecodeError is the aggressive recovery path when
// Grok returns "Could not decode the compaction blob". Mild scrub often no-ops
// (CLI already sent non-type=compaction shapes, or only previous_response_id).
// This rewrites the outbound body so a second attempt can continue the chat
// from portable text history instead of account-scoped compact state.
func sanitizeBodyAfterCompactionDecodeError(body []byte, recall *gatewayCompactRecall) ([]byte, map[string]any) {
	stats := map[string]any{"changed": false}
	if len(body) == 0 {
		return body, stats
	}
	out := body
	if rewritten, ok := resolveGatewayPreviousResponse(out, recall); ok {
		out = rewritten
		stats["previous_response_expanded"] = true
		stats["changed"] = true
	}
	if stripped, ok := stripPreviousResponseID(out); ok {
		out = stripped
		stats["previous_response_stripped"] = true
		stats["changed"] = true
	}
	if scrubbed, n := scrubUpstreamCompactionBlobs(out); n > 0 {
		out = scrubbed
		stats["compaction_objects_removed"] = n
		stats["changed"] = true
	}
	// Nuclear pass: drop any remaining compact-like payloads Grok would try to
	// decode (opaque encrypted_content on non-reasoning items, nested parts).
	if cleaned, n := stripOpaqueCompactPayloads(out); n > 0 {
		out = cleaned
		stats["opaque_payloads_removed"] = n
		stats["changed"] = true
	}
	return out, stats
}

// stripOpaqueCompactPayloads removes compact-like encrypted blobs that are not
// ordinary reasoning replay items. Reasoning.encrypted_content is kept (Build
// multi-turn needs it); type=compaction and bare opaque blobs are replaced.
func stripOpaqueCompactPayloads(body []byte) ([]byte, int) {
	if len(body) == 0 {
		return body, 0
	}
	var payload any
	if json.Unmarshal(body, &payload) != nil {
		return body, 0
	}
	cleaned, removed := stripOpaqueCompactValue(payload)
	if removed == 0 {
		return body, 0
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return body, 0
	}
	return encoded, removed
}

func stripOpaqueCompactValue(value any) (any, int) {
	switch typed := value.(type) {
	case map[string]any:
		if isCompactionInputItem(typed) || looksLikeCompactionObject(typed) {
			return foreignCompactionBoundaryMessage(), 1
		}
		typ := strings.ToLower(strings.TrimSpace(stringField(typed, "type")))
		// Keep reasoning encrypted_content (not a compaction blob).
		if typ == "reasoning" {
			removed := 0
			for key, nested := range typed {
				if key == "encrypted_content" || key == "encryptedContent" {
					continue
				}
				next, n := stripOpaqueCompactValue(nested)
				if n > 0 {
					typed[key] = next
					removed += n
				}
			}
			return typed, removed
		}
		removed := 0
		// Non-reasoning objects: drop long opaque encrypted_content fields that
		// Grok may treat as compact state even without type=compaction.
		for _, key := range []string{"encrypted_content", "encryptedContent"} {
			if blob, ok := typed[key].(string); ok && looksLikeOpaqueCompactBlob(blob) {
				delete(typed, key)
				removed++
			}
		}
		for key, nested := range typed {
			next, n := stripOpaqueCompactValue(nested)
			if n > 0 {
				typed[key] = next
				removed += n
			}
		}
		if isCompactionInputItem(typed) || looksLikeCompactionObject(typed) {
			return foreignCompactionBoundaryMessage(), removed + 1
		}
		return typed, removed
	case []any:
		removed := 0
		for index, nested := range typed {
			next, n := stripOpaqueCompactValue(nested)
			if n > 0 {
				typed[index] = next
				removed += n
			}
		}
		return typed, removed
	default:
		return value, 0
	}
}

func looksLikeOpaqueCompactBlob(blob string) bool {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return false
	}
	if strings.HasPrefix(blob, gatewayCompactionPrefix) {
		return true
	}
	// Native Grok / foreign compact payloads are long opaque tokens.
	return len(blob) >= 64
}

func isCompactionBlobDecodeError(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "compaction blob") ||
		strings.Contains(lower, "could not decode the compaction") ||
		(strings.Contains(lower, "invalid-argument") && strings.Contains(lower, "compaction"))
}
