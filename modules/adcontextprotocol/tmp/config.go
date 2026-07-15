package tmp

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/adcontextprotocol/adcp-go/tmproto"
)

// Config is the JSON configuration for the module. See README.md.
type Config struct {
	// SellerAgentURL identifies this Prebid Server deployment as a seller agent.
	// MUST match one of the property's adagents.json authorized_agents[].url
	// entries (compared with AdCP URL canonicalization). Same value for every
	// user on a given placement — carries no user identity.
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
	// merged TMP signals. Defaults to "adcp".
	TargetingKey string `json:"targeting_key"`

	// AddToTargeting mirrors the response signals into prebid.targeting so
	// downstream ad servers (e.g. GAM) can consume them.
	AddToTargeting bool `json:"add_to_targeting"`
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
}

// PropertyRegistryConfig configures the domain → property_rid resolver.
type PropertyRegistryConfig struct {
	// Endpoint is the resolve endpoint of the property registry, e.g.
	// https://agenticadvertising.org/api/properties/resolve. Domain is
	// appended as ?domain=… on GET.
	Endpoint string `json:"endpoint"`
	// AuthBearer is the optional bearer token sent as Authorization: Bearer …
	// on registry calls. May be substituted from env in deployment YAML.
	AuthBearer string `json:"auth_bearer"`
	// CacheTTLSeconds is how long a successful lookup is memoized. Default 3600.
	CacheTTLSeconds int `json:"cache_ttl_seconds"`
	// NegativeCacheTTLSeconds is how long a "not found" answer is memoized. Default 300.
	NegativeCacheTTLSeconds int `json:"negative_cache_ttl_seconds"`
	// CacheSize is the max number of entries kept in memory. Default 4096.
	CacheSize int `json:"cache_size"`
	// TimeoutMs bounds a single registry HTTP call. Default 500.
	TimeoutMs int `json:"timeout_ms"`
}

// ProviderConfig describes a single downstream TMP provider (identity agent,
// context agent, or both).
type ProviderConfig struct {
	Name string `json:"name"`
	// IdentityURL, if set, receives IdentityMatch requests.
	IdentityURL string `json:"identity_url"`
	// ContextURL, if set, receives ContextMatch requests.
	ContextURL string `json:"context_url"`
	// TimeoutMs overrides the module-level timeout for this provider. Optional.
	TimeoutMs int `json:"timeout_ms"`
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

// validated returns a Config with defaults filled in, along with the parsed
// Ed25519 private key. Invalid configuration is rejected here rather than at
// call sites.
func (c *Config) validated() (ed25519.PrivateKey, error) {
	if c.SellerAgentURL == "" {
		return nil, errors.New("seller_agent_url is required")
	}
	if c.Signing.KeyID == "" {
		return nil, errors.New("signing.key_id is required")
	}
	if c.Signing.PrivateKeyPEM == "" {
		return nil, errors.New("signing.private_key_pem is required")
	}
	priv, err := tmproto.LoadEd25519PrivateKeyPEM([]byte(c.Signing.PrivateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("signing.private_key_pem: %w", err)
	}
	if len(c.Providers) == 0 {
		return nil, errors.New("at least one provider is required")
	}
	for i := range c.Providers {
		p := &c.Providers[i]
		if p.Name == "" {
			return nil, fmt.Errorf("providers[%d].name is required", i)
		}
		if p.IdentityURL == "" && p.ContextURL == "" {
			return nil, fmt.Errorf("providers[%d] (%s): at least one of identity_url or context_url is required", i, p.Name)
		}
	}
	if c.PropertyRegistry.Endpoint == "" {
		return nil, errors.New("property_registry.endpoint is required")
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
	return priv, nil
}
