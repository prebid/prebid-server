package tmp

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/prebid-server/v4/logger"
)

// Config is the JSON configuration for the module. See README.md.
type Config struct {
	// SellerAgentURL identifies this Prebid Server deployment as a seller
	// agent. The value is sent on every outbound context / identity
	// request AND is folded into the Ed25519 signing preimage — so it is
	// the receiving agent's handle on this deployment for both package
	// evaluation and signature verification. Same value for every user
	// on a given placement; carries no user identity.
	//
	// Operational contract — the same canonical URL MUST resolve in all
	// three of these places, or the receiving agent has no way to
	// verify us:
	//
	//   1. The publisher's adagents.json `authorized_agents[].url` for
	//      every property served by this deployment (spec `agent_url`
	//      per adcp docs/reference/url-canonicalization).
	//   2. A row in the AdCP registry's `/api/registry/authorizations`
	//      endpoint whose `agent_url` matches this value and whose
	//      `signing_keys[]` carries the Ed25519 pubkey paired with
	//      Signing.KeyID below (adcp-go tmproto.LazyAuthorizationKeyStore
	//      queries `?agent_url=<this>` to fetch it).
	//   3. This config field.
	//
	// All comparisons are AdCP URL canonicalization, not byte-equality
	// (see adcp docs/reference/url-canonicalization and
	// tmproto.NormalizeProviderEndpointURL) — case, default ports, and
	// trailing-slash normalization are transparent, but path is
	// significant, so e.g. `https://seller.example.com/` and
	// `https://seller.example.com/agent` are different agents. A
	// mismatch in any of the three surfaces yields a hard 401 at the
	// receiving agent (ErrSignatureKeyUnknown), because seller_agent_url
	// is bound into the signed input — a request that reports one URL
	// but is signed under another cannot be verified.
	SellerAgentURL string `json:"seller_agent_url"`

	// PropertyType default when the registry does not return one. Optional.
	DefaultPropertyType string `json:"default_property_type"`

	// TimeoutMs is the overall budget for the TMP fan-out. Individual providers
	// can override with their own Timeout field. Default 300 ms.
	TimeoutMs int `json:"timeout_ms"`

	// DecorrelationMaxDelayMs, when > 0, jitters the second of a provider's
	// context / identity outbound calls by a random duration in
	// [0, DecorrelationMaxDelayMs] milliseconds. The pair is also spawned in
	// a randomized order regardless of this value. Set to 0 to disable the
	// delay (order randomization remains on — it is free). Default 0.
	//
	// Recommended by the TMP spec as a MAY to break timing correlation
	// between the two calls at a passive observer. Costs latency on the
	// auction hot path — operators trade privacy for speed by tuning this.
	DecorrelationMaxDelayMs int `json:"decorrelation_max_delay_ms"`

	// Signing holds the Ed25519 key used to authenticate outbound requests to
	// TMP providers. Required.
	Signing SigningConfig `json:"signing"`

	// PropertyRegistry configures the adcp property catalog client used to
	// resolve domain → property_rid.
	PropertyRegistry PropertyRegistryConfig `json:"property_registry"`

	// Providers is the list of downstream TMP providers to fan out to. At least
	// one is required. Each provider must have at least one of IdentityURL or
	// ContextURL configured.
	Providers []ProviderConfig `json:"providers"`

	// Masking optionally coarsens the ContextMatch payload before it leaves the
	// server. Identity payloads never carry the fields Masking operates on.
	Masking MaskingConfig `json:"masking"`

	// TargetingKey is the ext key on the bid response under which we surface
	// the raw merged TMP segment list (a []string of "key=value" pairs, useful
	// for callers that consume the response.ext directly). Defaults to "adcp".
	TargetingKey string `json:"targeting_key"`

	// PackageTargetingKey is the single custom key under which the module
	// emits all matched package_ids on prebid.targeting, comma-joined and
	// deduplicated across every provider that responded. Ad-server line items
	// target on this key with IN semantics (e.g. GAM: adcp_package_id ∈
	// {pkg_a, pkg_b}). Defaults to "adcp_package_id"; set to "" to disable
	// package emission on prebid.targeting.
	PackageTargetingKey string `json:"package_targeting_key"`

	// AddToTargeting mirrors merged package IDs, context response signals,
	// per-offer creative macros, and identity TMPX chunks into
	// prebid.targeting so downstream ad servers (GAM, VAST URL macros, DOOH
	// play-log fields) can consume them. Signal and offer-macro keys are
	// emitted as the agents return them. Identity TMPX values are keyed on
	// the publisher-local macro names resolved from TmpxMacroMapping —
	// providers never name the destination.
	AddToTargeting bool `json:"add_to_targeting"`

	// TmpxMacroMapping is the publisher-owned deployment configuration that
	// resolves each provider's ordered TMPX chunks to local ad-server macro
	// names (or targeting keys, VAST URL substitutions, DOOH play-log
	// fields) on this Prebid Server surface. Outer map key is the
	// provider's Name in this module's Providers[] (used as `provider_id`
	// in the adcp TMP spec); inner map key is the provider-local `slot_id`
	// the provider registered in `tmpx_slots`; value is the ad-server
	// destination the publisher trafficks against.
	//
	// The map is authored by the same operator who trafficks the
	// corresponding ad-server line items; it never travels on the wire
	// between identity provider and this module. This keeps macro naming a
	// deployment concern rather than a protocol identifier: a hostile or
	// misconfigured identity provider cannot pick a macro name the
	// publisher did not intend.
	//
	// At serve time, the router iterates
	// ProviderIdentityMatchResponse.tmpx_chunks[] for each provider and
	// emits macro=value for every chunk whose slot_id is present in the
	// inner map. A chunk whose (provider, slot_id) is absent causes the
	// whole provider's chunks to be dropped atomically for that impression
	// (fail-closed per adcp publisher-tmpx-config.json). Providers with no
	// entry in the outer map produce no TMPX targeting on this surface.
	//
	// See docs/trusted-match/specification.mdx and publisher-tmpx-config.json
	// in the adcontextprotocol/adcp repo for the wire-level model.
	TmpxMacroMapping map[string]map[string]string `json:"tmpx_macro_mapping"`

	// MaxSegments caps the total number of segments emitted onto the
	// response ext, regardless of how many providers respond or how many
	// offers/signals they include. Default 128. A hostile-or-buggy
	// provider cannot bloat the bid response past this bound.
	MaxSegments int `json:"max_segments"`

	// MaxSegmentValueLen bounds each emitted segment string (name +
	// separator + value). Default 256. Excess is truncated. A cap of 0
	// disables truncation.
	MaxSegmentValueLen int `json:"max_segment_value_len"`
}

// SigningConfig carries the private-key material used to sign outbound TMP
// requests. Ed25519 per the TMP spec.
type SigningConfig struct {
	// KeyID is echoed in the X-AdCP-Key-Id header so verifiers can look up the
	// matching public key in the property registry.
	KeyID string `json:"key_id"`
	// PrivateKeyPEM holds the PEM-encoded PKCS#8 Ed25519 private key. Deployments
	// substitute this from the environment via yaml env expansion (e.g.
	// ${ADCP_TMP_SIGNING_KEY_PEM}) — the module itself receives it as a string.
	PrivateKeyPEM string `json:"private_key_pem"`

	// Disabled, when true, skips signing every outbound context / identity
	// request — the module omits both the X-AdCP-Signature and
	// X-AdCP-Key-Id headers. Intended for pre-production and rollout
	// scenarios where the verifier runs with TMP_ALLOW_UNSIGNED=true and
	// the seller's adagents.json / registry authorization is still
	// propagating. When Disabled=true, key_id and private_key_pem are
	// optional; otherwise both are required.
	//
	// Setting this in production leaks the seller identity's signing
	// guarantee — the verifier can no longer prove the request came from
	// this deployment, so relay attackers and misconfigured peers become
	// indistinguishable from legitimate traffic. Builder logs a loud WARN
	// at every startup so an unintended flag is visible in ops logs.
	// DO NOT use in production.
	Disabled bool `json:"disabled"`
}

// PropertyRegistryConfig configures the domain → property_rid resolver.
type PropertyRegistryConfig struct {
	// Endpoint is the POST resolve endpoint of the property catalog, e.g.
	// https://agenticadvertising.org/api/registry/resolve. The module sends
	// `{identifiers: [{type, value}], provenance: {...}, mode}` and parses
	// resolved[].property_rid out of the response. Default:
	// https://agenticadvertising.org/api/registry/resolve.
	Endpoint string `json:"endpoint"`
	// AuthBearer is the optional bearer token sent as Authorization: Bearer …
	// on registry calls. Required for Mode="resolve" (contributes and
	// creates missing catalog entries); optional for Mode="lookup" (pure
	// read). May be substituted from env in deployment YAML.
	AuthBearer string `json:"auth_bearer"`
	// Mode selects the resolve verb: "resolve" (default; requires
	// AuthBearer; contributes new identifiers to the catalog) or "lookup"
	// (pure read; no auth needed; returns null property_rid for unknown
	// identifiers). Publishers running Prebid Server against a shared
	// catalog SHOULD use "resolve" so their inventory lights up on the
	// catalog even when a domain is new.
	Mode string `json:"mode"`
	// ProvenanceType is the FactProvenance.type envelope on every request.
	// Enum, from adcp catalog-openapi.ts: agency_allowlist,
	// publisher_declaration, impression_log, ssp_inventory, deal_history,
	// data_partner, member_assertion. `crawl` is reserved for server-side
	// pipelines and rejected by the registry. Default: member_assertion —
	// a Prebid Server operator resolving on behalf of their own member org.
	ProvenanceType string `json:"provenance_type"`
	// ProvenanceContext is an optional free-text annotation attached to
	// FactProvenance.context (e.g. "prebid-server:staging"). Ignored when
	// empty.
	ProvenanceContext string `json:"provenance_context"`
	// CacheTTLSeconds is how long a successful lookup is memoized. Default 3600.
	CacheTTLSeconds int `json:"cache_ttl_seconds"`
	// NegativeCacheTTLSeconds is how long a "not found" answer is memoized. Default 300.
	NegativeCacheTTLSeconds int `json:"negative_cache_ttl_seconds"`
	// CacheSize is the max number of entries kept in memory. Default 4096.
	CacheSize int `json:"cache_size"`
	// TimeoutMs bounds a single registry HTTP call. Default 500.
	TimeoutMs int `json:"timeout_ms"`
}

// signingBase returns the provider's registered base URL — what the TMP
// signing preimage binds to. Derived by stripping the /identity or
// /context suffix from whichever dispatch URL is set (validated() has
// already asserted both derive to the same base when both are set) and
// applying adcp URL canonicalization via
// tmproto.NormalizeProviderEndpointURL.
//
// Aligned with adcp provider-registration.json: providers register a
// single `endpoint` base URL; the router appends /identity and /context
// only when dispatching. See adcp-go tmproto/own_endpoint_test.go for
// the regression that pinned this convention.
func (p ProviderConfig) signingBase() string {
	switch {
	case p.IdentityURL != "":
		return tmproto.NormalizeProviderEndpointURL(strings.TrimSuffix(tmproto.NormalizeProviderEndpointURL(p.IdentityURL), "/identity"))
	case p.ContextURL != "":
		return tmproto.NormalizeProviderEndpointURL(strings.TrimSuffix(tmproto.NormalizeProviderEndpointURL(p.ContextURL), "/context"))
	}
	return ""
}

// ProviderConfig describes a single downstream TMP provider (identity agent,
// context agent, or both). At least one of IdentityURL or ContextURL MUST
// be set — the two fields are independently optional but not both empty;
// validated() rejects an all-empty pair. Both must resolve to the same
// registered base URL under adcp URL canonicalization, since the TMP
// signing preimage binds to the provider's single `endpoint` from
// provider-registration.json — validated() rejects mismatched bases.
type ProviderConfig struct {
	Name string `json:"name"`
	// IdentityURL is the provider's /identity endpoint. Empty means this
	// provider does not serve identity — the router skips its identity
	// call and any offers pass through the eligibility gate unfiltered.
	// Full path is significant: the module POSTs here verbatim, but signs
	// against the derived base URL (this value minus the /identity
	// suffix, or itself when it doesn't end in /identity).
	IdentityURL string `json:"identity_url"`
	// ContextURL is the provider's /context endpoint. Empty means this
	// provider does not serve context — the router does not fetch offers
	// from it, and its identity eligibility (if configured) contributes
	// nothing on its own. Full path is significant: the module POSTs
	// here verbatim, but signs against the derived base URL (this value
	// minus the /context suffix, or itself when it doesn't end in
	// /context).
	ContextURL string `json:"context_url"`
	// TimeoutMs overrides the module-level timeout for this provider. Optional.
	TimeoutMs int `json:"timeout_ms"`

	// TmpxSlots is the ordered list of provider-local slot IDs this
	// provider registered in its `tmpx_slots` field per adcp
	// provider-registration.json. The module uses it to enforce the
	// ordered-prefix invariant on incoming responses (adcp#5971): a
	// provider's emitted tmpx_chunks[].slot_id sequence MUST be a
	// non-empty ordered prefix of this list; any other sequence
	// (reordered, sparse, unregistered, over-cap) causes the whole
	// provider's chunks to be dropped atomically for that impression.
	//
	// Required only when the provider will emit TMPX. Providers that do
	// not populate tmpx_chunks omit this list. Cap of 2 slots in v1
	// mirrors the schema's maxItems=2.
	TmpxSlots []string `json:"tmpx_slots"`
}

// MaskingConfig mirrors the categories the previous RTD module exposed, so
// operators can migrate configuration in-place.
type MaskingConfig struct {
	Enabled bool                `json:"enabled"`
	Geo     GeoMaskingConfig    `json:"geo"`
	User    UserMaskingConfig   `json:"user"`
	Device  DeviceMaskingConfig `json:"device"`
}

type GeoMaskingConfig struct {
	PreserveMetro    bool `json:"preserve_metro"`
	PreserveZip      bool `json:"preserve_zip"`
	PreserveCity     bool `json:"preserve_city"`
	LatLongPrecision int  `json:"lat_long_precision"`
}

type UserMaskingConfig struct {
	PreserveEids []string `json:"preserve_eids"`
}

type DeviceMaskingConfig struct {
	PreserveMobileIds bool `json:"preserve_mobile_ids"`
}

// providerNameRE matches the adcp `provider_id` charset from
// provider-registration.json (`^[A-Za-z0-9_]+$`, 1–64 chars). The name
// appears verbatim in logs, metrics, and as the outer key of
// TmpxMacroMapping — matching the spec charset avoids quoting or
// normalization mismatches when the same identifier flows across
// registration and mapping.
var providerNameRE = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

// tmpxSlotIDRE matches the adcp `slot_id` charset from tmpx-chunk.json
// (`^[a-zA-Z][a-zA-Z0-9_]*$`, 1–64 chars). Applied to both provider
// TmpxSlots entries and TmpxMacroMapping inner keys.
var tmpxSlotIDRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// tmpxMaxSlots mirrors the v1 slot cap from provider-registration.json
// and tmpx-chunk-derived response schemas.
const tmpxMaxSlots = 2

// tmpxMaxSlotIDLen mirrors the schema's `slot_id` maxLength.
const tmpxMaxSlotIDLen = 64

// tmpxMaxMacroLen mirrors publisher-tmpx-config.json's inner-value
// maxLength for ad-server destination strings.
const tmpxMaxMacroLen = 128

// validated returns a Config with defaults filled in, along with the parsed
// Ed25519 private key. Invalid configuration is rejected here rather than at
// call sites.
func (c *Config) validated() (ed25519.PrivateKey, error) {
	if c.SellerAgentURL == "" {
		return nil, errors.New("seller_agent_url is required")
	}
	var priv ed25519.PrivateKey
	if c.Signing.Disabled {
		// Reject stale key material rather than silently ignore it. If an
		// operator flips signing.disabled back to false in a later
		// deployment, they should re-supply the key material at the same
		// time, not rely on whatever was left behind in the YAML from a
		// previous run.
		if c.Signing.KeyID != "" || c.Signing.PrivateKeyPEM != "" {
			return nil, errors.New("signing.disabled=true is incompatible with signing.key_id or signing.private_key_pem being set; clear both so a later re-enable is deliberate")
		}
	} else {
		if c.Signing.KeyID == "" {
			return nil, errors.New("signing.key_id is required (set signing.disabled=true to opt out, non-production only)")
		}
		if c.Signing.PrivateKeyPEM == "" {
			return nil, errors.New("signing.private_key_pem is required (set signing.disabled=true to opt out, non-production only)")
		}
		var err error
		priv, err = tmproto.LoadEd25519PrivateKeyPEM([]byte(c.Signing.PrivateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("signing.private_key_pem: %w", err)
		}
	}
	if len(c.Providers) == 0 {
		return nil, errors.New("at least one provider is required")
	}
	seenNames := make(map[string]bool, len(c.Providers))
	for i := range c.Providers {
		p := &c.Providers[i]
		if p.Name == "" {
			return nil, fmt.Errorf("providers[%d].name is required", i)
		}
		if !providerNameRE.MatchString(p.Name) {
			return nil, fmt.Errorf("providers[%d].name %q must match adcp provider_id charset %s", i, p.Name, providerNameRE)
		}
		if seenNames[p.Name] {
			return nil, fmt.Errorf("providers[%d].name %q is duplicated", i, p.Name)
		}
		seenNames[p.Name] = true
		if p.IdentityURL == "" && p.ContextURL == "" {
			return nil, fmt.Errorf("providers[%d] (%s): at least one of identity_url or context_url is required", i, p.Name)
		}
		// Both dispatch URLs, when set, must resolve to the same base
		// URL — the TMP signing preimage binds to that single base (per
		// adcp provider-registration.json `endpoint`, verified by
		// adcp-go tmproto/own_endpoint_test.go). A mismatch would make
		// context signatures verify but identity signatures fail (or
		// vice versa) at the same provider, so reject at startup.
		if p.IdentityURL != "" && p.ContextURL != "" {
			idBase := (ProviderConfig{IdentityURL: p.IdentityURL}).signingBase()
			ctxBase := (ProviderConfig{ContextURL: p.ContextURL}).signingBase()
			if idBase != ctxBase {
				return nil, fmt.Errorf("providers[%d] (%s): identity_url and context_url must share a base URL (got %q and %q) — signing binds to the provider's registered endpoint", i, p.Name, idBase, ctxBase)
			}
		}
		if len(p.TmpxSlots) > tmpxMaxSlots {
			return nil, fmt.Errorf("providers[%d] (%s): tmpx_slots holds %d entries; adcp v1 caps registered slots at %d", i, p.Name, len(p.TmpxSlots), tmpxMaxSlots)
		}
		seenSlots := make(map[string]bool, len(p.TmpxSlots))
		for j, slotID := range p.TmpxSlots {
			if slotID == "" {
				return nil, fmt.Errorf("providers[%d] (%s): tmpx_slots[%d] must be non-empty", i, p.Name, j)
			}
			if len(slotID) > tmpxMaxSlotIDLen {
				return nil, fmt.Errorf("providers[%d] (%s): tmpx_slots[%d] %q exceeds %d chars", i, p.Name, j, slotID, tmpxMaxSlotIDLen)
			}
			if !tmpxSlotIDRE.MatchString(slotID) {
				return nil, fmt.Errorf("providers[%d] (%s): tmpx_slots[%d] %q must match adcp slot_id charset %s", i, p.Name, j, slotID, tmpxSlotIDRE)
			}
			if seenSlots[slotID] {
				return nil, fmt.Errorf("providers[%d] (%s): tmpx_slots[%d] %q is duplicated (schema requires uniqueItems)", i, p.Name, j, slotID)
			}
			seenSlots[slotID] = true
		}
	}
	if c.PropertyRegistry.Endpoint == "" {
		c.PropertyRegistry.Endpoint = "https://agenticadvertising.org/api/registry/resolve"
	}
	switch c.PropertyRegistry.Mode {
	case "":
		c.PropertyRegistry.Mode = "resolve"
	case "resolve", "lookup":
		// ok
	default:
		return nil, fmt.Errorf("property_registry.mode %q must be one of resolve, lookup", c.PropertyRegistry.Mode)
	}
	if c.PropertyRegistry.Mode == "resolve" && c.PropertyRegistry.AuthBearer == "" {
		return nil, errors.New("property_registry.auth_bearer is required when mode is resolve; use mode: lookup for the unauthenticated read")
	}
	if c.PropertyRegistry.ProvenanceType == "" {
		c.PropertyRegistry.ProvenanceType = "member_assertion"
	}
	switch c.PropertyRegistry.ProvenanceType {
	case "agency_allowlist", "publisher_declaration", "impression_log",
		"ssp_inventory", "deal_history", "data_partner", "member_assertion":
		// ok — enum from adcp catalog-openapi.ts FactProvenance.type
	default:
		return nil, fmt.Errorf("property_registry.provenance_type %q is not a valid adcp FactProvenance.type", c.PropertyRegistry.ProvenanceType)
	}

	if c.TimeoutMs <= 0 {
		c.TimeoutMs = 300
	}
	if c.DecorrelationMaxDelayMs < 0 {
		return nil, errors.New("decorrelation_max_delay_ms cannot be negative")
	}
	if c.PropertyRegistry.CacheTTLSeconds <= 0 {
		c.PropertyRegistry.CacheTTLSeconds = 3600
	}
	if c.PropertyRegistry.NegativeCacheTTLSeconds <= 0 {
		c.PropertyRegistry.NegativeCacheTTLSeconds = 300
	}
	if c.PropertyRegistry.CacheSize <= 0 {
		c.PropertyRegistry.CacheSize = 4096
	}
	if c.PropertyRegistry.TimeoutMs <= 0 {
		c.PropertyRegistry.TimeoutMs = 500
	}
	if c.TargetingKey == "" {
		c.TargetingKey = "adcp"
	}
	if c.PackageTargetingKey == "" {
		c.PackageTargetingKey = "adcp_package_id"
	}
	if c.MaxSegments <= 0 {
		c.MaxSegments = 128
	}
	if c.MaxSegmentValueLen < 0 {
		return nil, errors.New("max_segment_value_len cannot be negative")
	}
	if c.MaxSegmentValueLen == 0 {
		c.MaxSegmentValueLen = 256
	}
	if c.Masking.Enabled {
		if c.Masking.Geo.LatLongPrecision > 4 {
			return nil, errors.New("masking.geo.lat_long_precision cannot exceed 4")
		}
		if c.Masking.Geo.LatLongPrecision < 0 {
			return nil, errors.New("masking.geo.lat_long_precision cannot be negative")
		}
		if len(c.Masking.User.PreserveEids) == 0 {
			c.Masking.User.PreserveEids = []string{"liveramp.com", "uidapi.com", "id5-sync.com"}
		}
	}
	providerSlots := make(map[string]map[string]bool, len(c.Providers))
	for i := range c.Providers {
		p := &c.Providers[i]
		set := make(map[string]bool, len(p.TmpxSlots))
		for _, s := range p.TmpxSlots {
			set[s] = true
		}
		providerSlots[p.Name] = set
	}
	for providerID, slotMap := range c.TmpxMacroMapping {
		if !providerNameRE.MatchString(providerID) {
			return nil, fmt.Errorf("tmpx_macro_mapping key %q must match adcp provider_id charset %s", providerID, providerNameRE)
		}
		if !seenNames[providerID] {
			return nil, fmt.Errorf("tmpx_macro_mapping refers to provider %q that is not in providers[]", providerID)
		}
		if len(slotMap) == 0 {
			return nil, fmt.Errorf("tmpx_macro_mapping[%q] is empty; omit the provider to disable its TMPX targeting", providerID)
		}
		if len(slotMap) > tmpxMaxSlots {
			return nil, fmt.Errorf("tmpx_macro_mapping[%q] holds %d entries; adcp v1 caps slots at %d", providerID, len(slotMap), tmpxMaxSlots)
		}
		registered := providerSlots[providerID]
		for slotID, macro := range slotMap {
			if !tmpxSlotIDRE.MatchString(slotID) {
				return nil, fmt.Errorf("tmpx_macro_mapping[%q]: slot_id %q must match adcp slot_id charset %s", providerID, slotID, tmpxSlotIDRE)
			}
			if len(slotID) > tmpxMaxSlotIDLen {
				return nil, fmt.Errorf("tmpx_macro_mapping[%q]: slot_id %q exceeds %d chars", providerID, slotID, tmpxMaxSlotIDLen)
			}
			if macro == "" {
				return nil, fmt.Errorf("tmpx_macro_mapping[%q][%q]: destination macro must be non-empty", providerID, slotID)
			}
			if len(macro) > tmpxMaxMacroLen {
				return nil, fmt.Errorf("tmpx_macro_mapping[%q][%q]: destination macro exceeds %d chars", providerID, slotID, tmpxMaxMacroLen)
			}
			if len(registered) > 0 && !registered[slotID] {
				return nil, fmt.Errorf("tmpx_macro_mapping[%q][%q] references a slot_id not in provider %q tmpx_slots %v", providerID, slotID, providerID, c.providerSlotList(providerID))
			}
		}
		if len(registered) > 0 {
			for slotID := range registered {
				if _, ok := slotMap[slotID]; !ok {
					logger.Warnf("adcontextprotocol.tmp: tmpx_macro_mapping[%q] has no entry for registered slot_id %q; that slot will fail closed at serve time", providerID, slotID)
				}
			}
		}
	}
	return priv, nil
}

// providerSlotList returns the ordered tmpx_slots list for a provider by
// name, or nil when the provider is not configured. Used only for error
// messages so operators see the registered list next to the offending
// mapping entry.
func (c *Config) providerSlotList(name string) []string {
	for i := range c.Providers {
		if c.Providers[i].Name == name {
			return c.Providers[i].TmpxSlots
		}
	}
	return nil
}
