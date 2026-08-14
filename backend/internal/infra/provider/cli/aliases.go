package cli

import (
	"github.com/chenyme/grok2api/backend/internal/domain/account"
	modeldomain "github.com/chenyme/grok2api/backend/internal/domain/model"
	"github.com/chenyme/grok2api/backend/internal/infra/provider"
)

// Build reasoning-effort aliases (client cannot always pass reasoning fields).
// Same pattern as Console multi-agent-low/medium/high/xhigh: one upstream model,
// fixed effort via gateway rewriteAliasedModel.
//
// Only appear in GET /v1/models when the target upstream exists in model_routes
// (after Build account sync discovers grok-4.5).
var buildAliases = []provider.ModelAlias{
	buildAlias("grok-4.5-low", "grok-4.5", "grok-4.5", "low"),
	buildAlias("grok-4.5-medium", "grok-4.5", "grok-4.5", "medium"),
	buildAlias("grok-4.5-high", "grok-4.5", "grok-4.5", "high"),
	buildAlias("grok-4.5-xhigh", "grok-4.5", "grok-4.5", "xhigh"),
}

func buildAlias(alias, publicModel, upstreamModel, effort string) provider.ModelAlias {
	canonical, _ := modeldomain.NormalizePublicID(account.ProviderBuild, publicModel)
	return provider.ModelAlias{
		Alias: alias, PublicModel: canonical, Provider: account.ProviderBuild,
		UpstreamModel: upstreamModel, ReasoningEffort: effort,
	}
}

// Aliases returns Build client-facing effort shortcuts.
func Aliases() []provider.ModelAlias {
	return append([]provider.ModelAlias(nil), buildAliases...)
}

// AliasRoutes returns model_routes rows for Build effort shortcuts so each
// client-facing name has a stable DB id for client-key ACL (not id=0 virtual rows).
// Upstream stays the real model (e.g. grok-4.5); gateway injects effort on rewrite.
func AliasRoutes() []modeldomain.Route {
	values := make([]modeldomain.Route, 0, len(buildAliases))
	for _, alias := range buildAliases {
		publicID, ok := modeldomain.NormalizePublicID(account.ProviderBuild, alias.Alias)
		if !ok || publicID == "" {
			continue
		}
		values = append(values, modeldomain.Route{
			PublicID: publicID, Provider: account.ProviderBuild, UpstreamModel: alias.UpstreamModel,
			Capability: modeldomain.CapabilityResponses, Origin: modeldomain.OriginCatalog, Enabled: true,
		})
	}
	return values
}

// ModelAliases implements provider.ModelAliasAdapter.
func (a *Adapter) ModelAliases() []provider.ModelAlias { return Aliases() }
