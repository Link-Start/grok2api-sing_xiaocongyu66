package inference

import (
	"testing"
	"time"

	"github.com/chenyme/grok2api/backend/internal/domain/account"
	modeldomain "github.com/chenyme/grok2api/backend/internal/domain/model"
)

func TestNewModelListItemsDeduplicatesSharedPublicName(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	items := newModelListItems([]modeldomain.Route{
		{PublicID: "Build/grok-shared", Provider: account.ProviderBuild, CreatedAt: now},
		{PublicID: "Console/grok-shared", Provider: account.ProviderConsole, CreatedAt: now.Add(time.Second)},
		{PublicID: "Web/grok-chat-fast", Provider: account.ProviderWeb, CreatedAt: now},
	}, nil)
	if len(items) != 2 || items[0].ID != "grok-shared" || items[1].ID != "grok-chat-fast" {
		t.Fatalf("model list = %#v", items)
	}
}

func TestNewModelListItemsIncludesConsoleAliases(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	items := newModelListItems([]modeldomain.Route{
		{PublicID: "Console/grok-4.20-multi-agent-0309", Provider: account.ProviderConsole, CreatedAt: now},
	}, []string{"grok-4.20-multi-agent-xhigh", "grok-4.20-multi-agent-high", "grok-4.20-multi-agent-0309"})
	// Canonical external name of the route is multi-agent-0309; aliases add xhigh/high without duplicating 0309.
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	want := map[string]bool{
		"grok-4.20-multi-agent-0309":  true,
		"grok-4.20-multi-agent-xhigh": true,
		"grok-4.20-multi-agent-high":  true,
	}
	if len(ids) != 3 {
		t.Fatalf("items = %#v", items)
	}
	for _, id := range ids {
		if !want[id] {
			t.Fatalf("unexpected id %q in %#v", id, ids)
		}
	}
}
