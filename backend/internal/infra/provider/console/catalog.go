package console

import (
	"github.com/chenyme/grok2api/backend/internal/domain/account"
	modeldomain "github.com/chenyme/grok2api/backend/internal/domain/model"
	"github.com/chenyme/grok2api/backend/internal/infra/provider"
)

const (
	QuotaMode = "console"
	// DefaultQuotaLimit is the per-window request budget for console.x.ai.
	DefaultQuotaLimit = 20
	// DefaultQuotaWindow is the recovery window in seconds once the rotate timer starts.
	DefaultQuotaWindow = 3600
	// RotateThreshold starts the recovery timer only after remaining drops to this
	// value (inclusive). Mirrors jiujiu532/grok2api delayed rotation so high-balance
	// accounts stay preferred and timers are not started on first use.
	RotateThreshold = 12
)

type ModelSpec struct {
	PublicID               string
	UpstreamModel          string
	SupportsReasoning      bool
	DefaultReasoningEffort string
	MaxOutputTokens        int
	SearchTools            bool
}

// catalog is the built-in console.x.ai model directory (aligned with
// jiujiu532/grok2api app/control/model/registry.py Console Chat section).
// These are seeded into model_routes at startup and on admin「同步模型」;
// they are NOT discovered from a remote /models API.
var catalog = []ModelSpec{
	{PublicID: "grok-4.3", UpstreamModel: "grok-4.3", SupportsReasoning: true, DefaultReasoningEffort: "medium", MaxOutputTokens: 1_000_000, SearchTools: true},
	{PublicID: "grok-4.20-0309", UpstreamModel: "grok-4.20-0309", MaxOutputTokens: 1_000_000, SearchTools: true},
	{PublicID: "grok-4.20-0309-reasoning", UpstreamModel: "grok-4.20-0309-reasoning", MaxOutputTokens: 1_000_000, SearchTools: true},
	{PublicID: "grok-4.20-0309-non-reasoning", UpstreamModel: "grok-4.20-0309-non-reasoning", MaxOutputTokens: 1_000_000, SearchTools: true},
	{PublicID: "grok-4.20-multi-agent-0309", UpstreamModel: "grok-4.20-multi-agent-0309", SupportsReasoning: true, DefaultReasoningEffort: "medium", MaxOutputTokens: 2_000_000, SearchTools: true},
	{PublicID: "grok-build-0.1", UpstreamModel: "grok-build-0.1", MaxOutputTokens: 256_000, SearchTools: true},
}

// aliases are client-facing IDs (same names as jiujiu MODELS console entries).
// They resolve to a catalog upstream + optional fixed reasoning effort — no separate DB row.
var aliases = []provider.ModelAlias{
	// *-console names (jiujiu public names for the same upstream models)
	consoleAlias("grok-4.3-console", "grok-4.3", "grok-4.3", ""),
	consoleAlias("grok-4.20-0309-console", "grok-4.20-0309", "grok-4.20-0309", ""),
	consoleAlias("grok-4.20-0309-reasoning-console", "grok-4.20-0309-reasoning", "grok-4.20-0309-reasoning", ""),
	consoleAlias("grok-4.20-0309-non-reasoning-console", "grok-4.20-0309-non-reasoning", "grok-4.20-0309-non-reasoning", ""),
	consoleAlias("grok-4.20-multi-agent-console", "grok-4.20-multi-agent-0309", "grok-4.20-multi-agent-0309", ""),
	consoleAlias("grok-build-console", "grok-build-0.1", "grok-build-0.1", ""),
	// fixed reasoning effort shortcuts
	consoleAlias("grok-4.3-low", "grok-4.3", "grok-4.3", "low"),
	consoleAlias("grok-4.3-medium", "grok-4.3", "grok-4.3", "medium"),
	consoleAlias("grok-4.3-high", "grok-4.3", "grok-4.3", "high"),
	consoleAlias("grok-4.20-multi-agent-low", "grok-4.20-multi-agent-0309", "grok-4.20-multi-agent-0309", "low"),
	consoleAlias("grok-4.20-multi-agent-medium", "grok-4.20-multi-agent-0309", "grok-4.20-multi-agent-0309", "medium"),
	consoleAlias("grok-4.20-multi-agent-high", "grok-4.20-multi-agent-0309", "grok-4.20-multi-agent-0309", "high"),
	consoleAlias("grok-4.20-multi-agent-xhigh", "grok-4.20-multi-agent-0309", "grok-4.20-multi-agent-0309", "xhigh"),
}

// ClientFacingIDs returns every name clients may put in request.model / GET /v1/models
// for Console: catalog public IDs + aliases (jiujiu-style pre-registered list).
func ClientFacingIDs() []string {
	out := make([]string, 0, len(catalog)+len(aliases))
	for _, spec := range catalog {
		out = append(out, spec.PublicID)
	}
	for _, alias := range aliases {
		out = append(out, alias.Alias)
	}
	return out
}

func consoleAlias(alias, publicModel, upstreamModel, effort string) provider.ModelAlias {
	canonical, _ := modeldomain.NormalizePublicID(account.ProviderConsole, publicModel)
	return provider.ModelAlias{
		Alias: alias, PublicModel: canonical, Provider: account.ProviderConsole,
		UpstreamModel: upstreamModel, ReasoningEffort: effort,
	}
}

func Catalog() []ModelSpec { return append([]ModelSpec(nil), catalog...) }

func Routes() []modeldomain.Route {
	values := make([]modeldomain.Route, 0, len(catalog))
	for _, spec := range catalog {
		publicID, _ := modeldomain.NormalizePublicID(account.ProviderConsole, spec.PublicID)
		values = append(values, modeldomain.Route{
			PublicID: publicID, Provider: account.ProviderConsole, UpstreamModel: spec.UpstreamModel,
			Capability: modeldomain.CapabilityResponses, Enabled: true,
		})
	}
	return values
}

func Resolve(upstreamModel string) (ModelSpec, bool) {
	for _, spec := range catalog {
		if spec.UpstreamModel == upstreamModel {
			return spec, true
		}
	}
	return ModelSpec{}, false
}

func Aliases() []provider.ModelAlias { return append([]provider.ModelAlias(nil), aliases...) }
