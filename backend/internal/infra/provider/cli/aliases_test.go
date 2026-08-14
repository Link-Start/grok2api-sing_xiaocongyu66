package cli

import (
	"strings"
	"testing"

	"github.com/chenyme/grok2api/backend/internal/domain/account"
	"github.com/chenyme/grok2api/backend/internal/infra/provider"
)

func TestBuildReasoningEffortAliasesRegistered(t *testing.T) {
	registry := provider.NewRegistry(NewAdapter(Config{}, nil))
	for _, name := range []string{"grok-4.5-low", "grok-4.5-medium", "grok-4.5-high", "grok-4.5-xhigh"} {
		alias, ok := registry.ResolveModelAlias(name)
		if !ok {
			t.Fatalf("alias %q missing", name)
		}
		if alias.Provider != account.ProviderBuild || alias.UpstreamModel != "grok-4.5" {
			t.Fatalf("alias %q = %#v", name, alias)
		}
		if !strings.HasSuffix(name, alias.ReasoningEffort) {
			t.Fatalf("alias %q effort %q mismatch", name, alias.ReasoningEffort)
		}
	}
	routes := AliasRoutes()
	if len(routes) != 4 {
		t.Fatalf("AliasRoutes = %d, want 4", len(routes))
	}
	for _, route := range routes {
		if route.Provider != account.ProviderBuild || route.UpstreamModel != "grok-4.5" || route.ID != 0 {
			t.Fatalf("alias route = %#v", route)
		}
		if !strings.HasPrefix(route.PublicID, "Build/") {
			t.Fatalf("public id = %q", route.PublicID)
		}
	}
}
