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
		{PublicID: "Build/grok-shared", Provider: account.ProviderBuild, Enabled: true, CreatedAt: now},
		{PublicID: "Console/grok-shared", Provider: account.ProviderConsole, Enabled: true, CreatedAt: now.Add(time.Second)},
		{PublicID: "Web/grok-chat-fast", Provider: account.ProviderWeb, Enabled: true, CreatedAt: now},
		// Disabled routes must not appear even if configured.
		{PublicID: "Build/disabled-model", Provider: account.ProviderBuild, Enabled: false, CreatedAt: now},
	}, nil)
	if len(items) != 2 || items[0].ID != "grok-shared" || items[1].ID != "grok-chat-fast" {
		t.Fatalf("model list = %#v", items)
	}
}

func TestNewModelListItemsIncludesConsoleAliases(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	items := newModelListItems([]modeldomain.Route{
		{PublicID: "Console/grok-4.20-multi-agent-0309", Provider: account.ProviderConsole, Enabled: true, CreatedAt: now},
		// Configured but not yet capability-synced still lists (dynamic exposure).
		{PublicID: "Console/grok-4.3", Provider: account.ProviderConsole, Enabled: true, CreatedAt: now},
	}, []string{"grok-4.20-multi-agent-xhigh", "grok-4.20-multi-agent-high", "grok-4.20-multi-agent-0309"})
	ids := make(map[string]bool, len(items))
	for _, item := range items {
		ids[item.ID] = true
	}
	for _, want := range []string{
		"grok-4.20-multi-agent-0309",
		"grok-4.20-multi-agent-xhigh",
		"grok-4.20-multi-agent-high",
		"grok-4.3",
	} {
		if !ids[want] {
			t.Fatalf("missing %q in %#v", want, items)
		}
	}
	if len(items) != 4 {
		t.Fatalf("items = %#v", items)
	}
}
