# Scope3 TMP Prebid Server Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Prebid Server hook module at `modules/scope3/tmp/` that calls a Scope3-operated AdCP Trusted Match Protocol (TMP) router and enriches the bid response with eligible packages, TMPX HPKE tokens, and buyer-defined targeting key-value pairs.

**Architecture:** Three-stage hook module (Entrypoint → ProcessedAuctionRequest → AuctionResponse) that fans out `N+1` parallel HTTP calls (N Context Match + 1 Identity Match) via `errgroup`, performs publisher-side intersection of context and identity eligibility per imp, and mutates the bid response. Mirrors the existing `modules/scope3/rtd/` lifecycle pattern but with structurally-separated TMP calls.

**Tech Stack:** Go 1.23, Prebid Server hooks framework (`hookstage`), `freecache` for caching, `errgroup` for parallel fan-out, `sjson` for JSON ext mutation, `testify` for tests, `httptest` for mock router integration tests. TMP wire types vendored from `github.com/adcontextprotocol/adcp-go/tmproto` (Go-version mismatch prevents direct import).

**Spec:** `docs/superpowers/specs/2026-05-27-scope3-tmp-prebid-module-design.md`

---

## File Structure

| File | Responsibility |
|---|---|
| `modules/scope3/tmp/proto.go` | Vendored TMP wire types (subset of `adcp-go/tmproto/types_gen.go`). Hand-stamped provenance comment. |
| `modules/scope3/tmp/module.go` | `Builder()`, `Module` struct, `Config`, three hook-handler methods. Construction-time validation. |
| `modules/scope3/tmp/account.go` | `accountResolver` — pure function resolving property_rid / property_type / placement_id / seller_agent_url / router_url from ext override + account config + module config. |
| `modules/scope3/tmp/masking.go` | Deep-copy + mask user/geo/device fields. Identity extraction capped at 3. ISO 3166-1 alpha-3 → alpha-2 country converter. |
| `modules/scope3/tmp/async_request.go` | `AsyncRequest` per-auction state. `fetchAsync` orchestrates the N+1 fan-out, intersection, and caching. |
| `modules/scope3/tmp/module_test.go` | Builder validation tests + hook-level integration tests with mock `httptest.Server` router. |
| `modules/scope3/tmp/account_test.go` | Table-driven tests for the three-source resolution precedence. |
| `modules/scope3/tmp/async_request_test.go` | Tests for `intersect`, cache key composition, and goroutine lifecycle. |
| `modules/scope3/tmp/masking_test.go` | Tests for masking, identity selection ordering, country conversion. |
| `modules/scope3/tmp/testdata/*.json` | Test fixtures (bid requests, responses, account configs). |
| `modules/scope3/tmp/README.md` | Configuration, deployment, migration notes. |
| `modules/builder.go` | Register `scope3.tmp` Builder in the `ModuleBuilders` map. |
| `modules/scope3/rtd/README.md` | Add one-paragraph pointer to the new TMP module as the preferred path. |

---

## Task 1: Bootstrap module skeleton and registration

**Files:**
- Create: `modules/scope3/tmp/module.go`
- Create: `modules/scope3/tmp/module_test.go`
- Modify: `modules/builder.go`

- [ ] **Step 1: Create the empty module package**

Create `modules/scope3/tmp/module.go`:
```go
// Package tmp implements a Prebid Server module for AdCP Trusted Match Protocol.
package tmp

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// Builder is the entry point for the module.
func Builder(config json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &Module{cfg: cfg}, nil
}

// Config holds module configuration.
type Config struct{}

// Module implements the Scope3 TMP module.
type Module struct {
	cfg Config
}
```

- [ ] **Step 2: Create the registration test**

Create `modules/scope3/tmp/module_test.go`:
```go
package tmp

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/stretchr/testify/require"
)

func TestBuilder_EmptyConfig(t *testing.T) {
	m, err := Builder(json.RawMessage(`{}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	require.NotNil(t, m)
}
```

- [ ] **Step 3: Run test — should fail (Config undefined or empty struct mismatch)**

Run: `go test ./modules/scope3/tmp/...`
Expected: PASS (this is a sanity check on the empty skeleton; if it fails, the build is broken).

- [ ] **Step 4: Register the module in the builders map**

Edit `modules/builder.go`. Add the import and entry:
```go
import (
	// ... existing imports ...
	scope3Tmp "github.com/prebid/prebid-server/v4/modules/scope3/tmp"
)

func builders() ModuleBuilders {
	return ModuleBuilders{
		// ... existing entries ...
		"scope3": {
			"rtd": scope3Rtd.Builder,
			"tmp": scope3Tmp.Builder,
		},
	}
}
```

- [ ] **Step 5: Verify module list test still passes**

Run: `go test ./modules/...`
Expected: PASS for all module tests, including the existing `modules/modules_test.go` that exercises the builders map.

- [ ] **Step 6: Commit**

```bash
git add modules/builder.go modules/scope3/tmp/module.go modules/scope3/tmp/module_test.go
git commit -m "Module: Scope3 TMP - scaffold module package and register builder"
```

---

## Task 2: Vendor TMP wire types from adcp-go

**Files:**
- Create: `modules/scope3/tmp/proto.go`

- [ ] **Step 1: Identify the upstream commit**

Fetch the current HEAD SHA of `adcp-go`'s `main` branch and note it for the provenance comment.

Run: `gh api repos/adcontextprotocol/adcp-go/commits/main --jq '.sha'`
Capture the SHA into a shell variable for use in the comment below.

- [ ] **Step 2: Create proto.go with vendored types**

Create `modules/scope3/tmp/proto.go`:
```go
package tmp

import "encoding/json"

// Types in this file are copied from
//   github.com/adcontextprotocol/adcp-go/tmproto/types_gen.go
// at upstream commit <PASTE-SHA-FROM-STEP-1>.
//
// adcp-go's go.mod declares go 1.25.0; prebid-server is go 1.23.0, so direct
// import would force a Go-version bump. Re-sync this file manually when the
// TMP wire schema changes.

// PropertyType is the kind of publisher property.
type PropertyType string

const (
	PropertyTypeWebsite        PropertyType = "website"
	PropertyTypeMobileApp      PropertyType = "mobile_app"
	PropertyTypeCTVApp         PropertyType = "ctv_app"
	PropertyTypeDesktopApp     PropertyType = "desktop_app"
	PropertyTypeDOOH           PropertyType = "dooh"
	PropertyTypePodcast        PropertyType = "podcast"
	PropertyTypeRadio          PropertyType = "radio"
	PropertyTypeLinearTV       PropertyType = "linear_tv"
	PropertyTypeStreamingAudio PropertyType = "streaming_audio"
	PropertyTypeAIAssistant    PropertyType = "ai_assistant"
)

// IdentityToken is one entry in IdentityMatchRequest.Identities.
type IdentityToken struct {
	UIDType   string `json:"uid_type"`
	UserToken string `json:"user_token"`
}

// ArtifactRef references public content adjacent to the ad opportunity.
type ArtifactRef struct {
	URL string `json:"url,omitempty"`
}

// ContextMatchRequest is sent to /tmp/context.
type ContextMatchRequest struct {
	Type            string         `json:"type"`
	ProtocolVersion string         `json:"protocol_version,omitempty"`
	RequestID       string         `json:"request_id"`
	PropertyRID     string         `json:"property_rid"`
	PropertyID      string         `json:"property_id,omitempty"`
	PropertyType    PropertyType   `json:"property_type"`
	PlacementID     string         `json:"placement_id"`
	ArtifactRefs    []ArtifactRef  `json:"artifact_refs,omitempty"`
}

// Offer is one returned activated package.
type Offer struct {
	PackageID string `json:"package_id"`
}

// KeyValuePair is one entry in Signals.TargetingKVs.
type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Signals is the response-level targeting payload.
type Signals struct {
	Segments     []string       `json:"segments,omitempty"`
	TargetingKVs []KeyValuePair `json:"targeting_kvs,omitempty"`
}

// ContextMatchResponse is returned by /tmp/context.
type ContextMatchResponse struct {
	Type      string  `json:"type"`
	RequestID string  `json:"request_id"`
	Offers    []Offer `json:"offers"`
	CacheTTL  int     `json:"cache_ttl,omitempty"`
	Signals   Signals `json:"signals,omitempty"`
}

// IdentityMatchRequest is sent to /tmp/identity.
type IdentityMatchRequest struct {
	Type            string          `json:"type"`
	ProtocolVersion string          `json:"protocol_version,omitempty"`
	RequestID       string          `json:"request_id"`
	SellerAgentURL  string          `json:"seller_agent_url"`
	Identities      []IdentityToken `json:"identities"`
	Country         string          `json:"country,omitempty"`
}

// IdentityMatchResponse is returned by /tmp/identity.
type IdentityMatchResponse struct {
	Type               string   `json:"type"`
	RequestID          string   `json:"request_id"`
	EligiblePackageIDs []string `json:"eligible_package_ids"`
	TTLSec             int      `json:"ttl_sec"`
	Tmpx               string   `json:"tmpx,omitempty"`
}

// ErrorResponse is returned for protocol errors.
type ErrorResponse struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id"`
	Code      string          `json:"code"`
	Message   string          `json:"message,omitempty"`
	Extra     json.RawMessage `json:"extra,omitempty"`
}

// Wire type discriminators (the "type" field on each message).
const (
	TypeContextMatchRequest   = "context_match_request"
	TypeContextMatchResponse  = "context_match_response"
	TypeIdentityMatchRequest  = "identity_match_request"
	TypeIdentityMatchResponse = "identity_match_response"
	TypeErrorResponse         = "error_response"
)
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./modules/scope3/tmp/...`
Expected: builds cleanly, no errors.

- [ ] **Step 4: Commit**

```bash
git add modules/scope3/tmp/proto.go
git commit -m "Module: Scope3 TMP - vendor TMP wire types from adcp-go"
```

---

## Task 3: Config struct and Builder validation

**Files:**
- Modify: `modules/scope3/tmp/module.go`
- Modify: `modules/scope3/tmp/module_test.go`

- [ ] **Step 1: Write Builder validation tests**

Append to `modules/scope3/tmp/module_test.go`:
```go
func TestBuilder_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError string
	}{
		{
			name:      "missing router_url",
			config:    `{"seller_agent_url":"https://example.com"}`,
			wantError: "router_url is required",
		},
		{
			name:      "missing seller_agent_url",
			config:    `{"router_url":"https://tmp.interchange.io"}`,
			wantError: "seller_agent_url is required",
		},
		{
			name:      "too many preserve_eids",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"user":{"preserve_eids":["a","b","c","d"]}}}`,
			wantError: "preserve_eids exceeds spec limit of 3 entries",
		},
		{
			name:      "negative lat_long_precision",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"geo":{"lat_long_precision":-1}}}`,
			wantError: "lat_long_precision cannot be negative",
		},
		{
			name:      "lat_long_precision over 4",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"geo":{"lat_long_precision":5}}}`,
			wantError: "lat_long_precision cannot exceed 4 decimal places for privacy protection",
		},
		{
			name:      "negative timeout_ms",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","timeout_ms":-1}`,
			wantError: "timeout_ms must be positive",
		},
		{
			name:   "valid minimal config",
			config: `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := moduledeps.ModuleDeps{HTTPClient: &http.Client{}}
			m, err := Builder(json.RawMessage(tc.config), deps)
			if tc.wantError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantError)
				require.Nil(t, m)
			} else {
				require.NoError(t, err)
				require.NotNil(t, m)
			}
		})
	}
}
```

Add the `net/http` import to the test file.

- [ ] **Step 2: Run tests — should fail (Config struct lacks fields, no validation logic)**

Run: `go test ./modules/scope3/tmp/... -run TestBuilder_Validation -v`
Expected: FAIL on every subcase.

- [ ] **Step 3: Implement the full Config + Builder**

Replace the contents of `modules/scope3/tmp/module.go`:
```go
// Package tmp implements a Prebid Server module for AdCP Trusted Match Protocol.
package tmp

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

const (
	defaultRouterURL     = "https://tmp.interchange.io"
	defaultTimeoutMs     = 200
	defaultCacheTTLSecs  = 60
	defaultCacheSize     = 10 * 1024 * 1024 // 10 MB
	maxIdentitiesPerSpec = 3
)

// Config holds module configuration.
type Config struct {
	RouterURL       string        `json:"router_url"`
	SellerAgentURL  string        `json:"seller_agent_url"`
	AuthKey         string        `json:"auth_key"`
	TimeoutMs       int           `json:"timeout_ms"`
	CacheTTLSeconds int           `json:"cache_ttl_seconds"`
	CacheSize       int           `json:"cache_size"`
	AddToTargeting  bool          `json:"add_to_targeting"`
	Masking         MaskingConfig `json:"masking"`
}

// MaskingConfig controls masking of user data before forwarding to the router.
type MaskingConfig struct {
	Enabled bool                `json:"enabled"`
	Geo     GeoMaskingConfig    `json:"geo"`
	User    UserMaskingConfig   `json:"user"`
	Device  DeviceMaskingConfig `json:"device"`
}

// GeoMaskingConfig controls geographic masking.
type GeoMaskingConfig struct {
	PreserveMetro    bool `json:"preserve_metro"`
	PreserveZip      bool `json:"preserve_zip"`
	PreserveCity     bool `json:"preserve_city"`
	LatLongPrecision int  `json:"lat_long_precision"`
}

// UserMaskingConfig controls user data masking.
type UserMaskingConfig struct {
	PreserveEids []string `json:"preserve_eids"`
}

// DeviceMaskingConfig controls device-identifier masking.
type DeviceMaskingConfig struct {
	PreserveMobileIds bool `json:"preserve_mobile_ids"`
}

// Module implements the Scope3 TMP module.
type Module struct {
	cfg        Config
	httpClient *http.Client
	cache      *freecache.Cache
	sha256Pool *sync.Pool
}

// Builder is the entry point for the module.
func Builder(rawCfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := json.Unmarshal(rawCfg, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	defaults(&cfg)

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.TimeoutMs) * time.Millisecond,
	}
	if deps.HTTPClient != nil && deps.HTTPClient.Transport != nil {
		httpClient.Transport = deps.HTTPClient.Transport
	}

	return &Module{
		cfg:        cfg,
		httpClient: httpClient,
		cache:      freecache.NewCache(cfg.CacheSize),
		sha256Pool: &sync.Pool{New: func() any { return sha256.New() }},
	}, nil
}

func validate(cfg *Config) error {
	if cfg.RouterURL == "" {
		return errors.New("router_url is required")
	}
	if cfg.SellerAgentURL == "" {
		return errors.New("seller_agent_url is required")
	}
	if cfg.TimeoutMs < 0 {
		return errors.New("timeout_ms must be positive")
	}
	if cfg.CacheSize < 0 {
		return errors.New("cache_size must be non-negative")
	}
	if cfg.Masking.Enabled {
		if cfg.Masking.Geo.LatLongPrecision < 0 {
			return errors.New("lat_long_precision cannot be negative")
		}
		if cfg.Masking.Geo.LatLongPrecision > 4 {
			return errors.New("lat_long_precision cannot exceed 4 decimal places for privacy protection")
		}
		if len(cfg.Masking.User.PreserveEids) > maxIdentitiesPerSpec {
			return fmt.Errorf("preserve_eids exceeds spec limit of %d entries", maxIdentitiesPerSpec)
		}
	}
	return nil
}

func defaults(cfg *Config) {
	if cfg.RouterURL == "" {
		cfg.RouterURL = defaultRouterURL
	}
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = defaultTimeoutMs
	}
	if cfg.CacheTTLSeconds == 0 {
		cfg.CacheTTLSeconds = defaultCacheTTLSecs
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = defaultCacheSize
	}
	if cfg.Masking.Enabled {
		if cfg.Masking.Geo.LatLongPrecision == 0 {
			cfg.Masking.Geo.LatLongPrecision = 2
		}
		if len(cfg.Masking.User.PreserveEids) == 0 {
			cfg.Masking.User.PreserveEids = []string{"liveramp.com", "uidapi.com", "id5-sync.com"}
		}
	}
}
```

- [ ] **Step 4: Run tests — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestBuilder -v`
Expected: PASS for all Builder subtests.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/module.go modules/scope3/tmp/module_test.go
git commit -m "Module: Scope3 TMP - Config struct and Builder validation"
```

---

## Task 4: accountResolver — identifier resolution from three sources

**Files:**
- Create: `modules/scope3/tmp/account.go`
- Create: `modules/scope3/tmp/account_test.go`

- [ ] **Step 1: Write the table-driven resolver test**

Create `modules/scope3/tmp/account_test.go`:
```go
package tmp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveAuctionIdentifiers(t *testing.T) {
	moduleCfg := Config{RouterURL: "https://router", SellerAgentURL: "https://us"}

	tests := []struct {
		name        string
		accountJSON string
		extJSON     string
		impTagID    string
		wantRID     string
		wantPType   PropertyType
		wantPlace   string
		wantSeller  string
		wantErr     string
	}{
		{
			name:        "all from account",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"01916f3a","property_type":"website","placements":{"header":"header_728x90"}}}}`,
			extJSON:     `{}`,
			impTagID:    "header",
			wantRID:     "01916f3a",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "header_728x90",
			wantSeller:  "https://us",
		},
		{
			name:        "ext overrides property_rid",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"acct","property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{"prebid":{"modules":{"scope3":{"tmp":{"property_rid":"override"}}}}}`,
			impTagID:    "h",
			wantRID:     "override",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "h1",
			wantSeller:  "https://us",
		},
		{
			name:        "ext placement_id overrides per-imp lookup",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{"prebid":{"modules":{"scope3":{"tmp":{"placement_id":"test_slot"}}}}}`,
			impTagID:    "h",
			wantRID:     "r",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "test_slot",
			wantSeller:  "https://us",
		},
		{
			name:        "account overrides seller_agent_url",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"h1"},"seller_agent_url":"https://alt"}}}`,
			extJSON:     `{}`,
			impTagID:    "h",
			wantRID:     "r",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "h1",
			wantSeller:  "https://alt",
		},
		{
			name:        "missing property_rid is error",
			accountJSON: `{"scope3":{"tmp":{"property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{}`,
			impTagID:    "h",
			wantErr:     "property_rid is required",
		},
		{
			name:        "missing property_type is error",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","placements":{"h":"h1"}}}}`,
			extJSON:     `{}`,
			impTagID:    "h",
			wantErr:     "property_type is required",
		},
		{
			name:        "unknown tagid yields empty placement_id (caller decides to skip)",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{}`,
			impTagID:    "unknown_tagid",
			wantRID:     "r",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "",
			wantSeller:  "https://us",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := accountResolver{
				accountConfig: json.RawMessage(tc.accountJSON),
				requestExt:    json.RawMessage(tc.extJSON),
				moduleCfg:     moduleCfg,
			}
			ids, err := r.resolveAuction()
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantRID, ids.PropertyRID)
			require.Equal(t, tc.wantPType, ids.PropertyType)
			require.Equal(t, tc.wantSeller, ids.SellerAgentURL)
			place, _ := r.resolvePlacement(tc.impTagID)
			require.Equal(t, tc.wantPlace, place)
		})
	}
}
```

- [ ] **Step 2: Run test — should fail (accountResolver not yet defined)**

Run: `go test ./modules/scope3/tmp/... -run TestResolveAuctionIdentifiers -v`
Expected: FAIL with "undefined: accountResolver".

- [ ] **Step 3: Implement accountResolver**

Create `modules/scope3/tmp/account.go`:
```go
package tmp

import (
	"encoding/json"
	"errors"

	"github.com/tidwall/gjson"
)

// AuctionIdentifiers groups the resolved identifiers shared across all imps.
type AuctionIdentifiers struct {
	PropertyRID    string
	PropertyType   PropertyType
	SellerAgentURL string
	RouterURL      string
	ExtPlacementID string // single value from ext override; applies to every imp if non-empty
}

// accountResolver pulls TMP identifiers from per-request ext, account config, and module config.
// Precedence: ext > account > module-level default (only for router_url and seller_agent_url).
// property_rid, property_type, and per-imp placement_id have NO module-level default.
type accountResolver struct {
	accountConfig json.RawMessage
	requestExt    json.RawMessage // request.Ext
	moduleCfg     Config
}

// resolveAuction returns the identifiers that are stable across all imps.
func (r accountResolver) resolveAuction() (AuctionIdentifiers, error) {
	ids := AuctionIdentifiers{
		RouterURL:      r.moduleCfg.RouterURL,
		SellerAgentURL: r.moduleCfg.SellerAgentURL,
	}

	if v := gjson.GetBytes(r.accountConfig, "scope3.tmp.property_rid"); v.Exists() {
		ids.PropertyRID = v.String()
	}
	if v := gjson.GetBytes(r.accountConfig, "scope3.tmp.property_type"); v.Exists() {
		ids.PropertyType = PropertyType(v.String())
	}
	if v := gjson.GetBytes(r.accountConfig, "scope3.tmp.seller_agent_url"); v.Exists() {
		ids.SellerAgentURL = v.String()
	}
	if v := gjson.GetBytes(r.accountConfig, "scope3.tmp.router_url"); v.Exists() {
		ids.RouterURL = v.String()
	}

	if v := gjson.GetBytes(r.requestExt, "prebid.modules.scope3.tmp.property_rid"); v.Exists() {
		ids.PropertyRID = v.String()
	}
	if v := gjson.GetBytes(r.requestExt, "prebid.modules.scope3.tmp.placement_id"); v.Exists() {
		ids.ExtPlacementID = v.String()
	}

	if ids.PropertyRID == "" {
		return ids, errors.New("property_rid is required")
	}
	if ids.PropertyType == "" {
		return ids, errors.New("property_type is required")
	}
	if ids.SellerAgentURL == "" {
		return ids, errors.New("seller_agent_url is required")
	}
	if ids.RouterURL == "" {
		return ids, errors.New("router_url is required")
	}
	return ids, nil
}

// resolvePlacement returns the placement_id for one imp.
// Returns ("", false) if the imp's tagid has no mapping and no ext override applies.
func (r accountResolver) resolvePlacement(impTagID string) (string, bool) {
	if v := gjson.GetBytes(r.requestExt, "prebid.modules.scope3.tmp.placement_id"); v.Exists() {
		return v.String(), true
	}
	if v := gjson.GetBytes(r.accountConfig, "scope3.tmp.placements."+impTagID); v.Exists() {
		return v.String(), true
	}
	return "", false
}
```

- [ ] **Step 4: Run test — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestResolveAuctionIdentifiers -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/account.go modules/scope3/tmp/account_test.go
git commit -m "Module: Scope3 TMP - accountResolver for property_rid, placement_id, seller_agent_url"
```

---

## Task 5: Country code conversion (alpha-3 → alpha-2)

**Files:**
- Create: `modules/scope3/tmp/masking.go`
- Create: `modules/scope3/tmp/masking_test.go`

- [ ] **Step 1: Write the conversion test**

Create `modules/scope3/tmp/masking_test.go`:
```go
package tmp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCountryAlpha3ToAlpha2(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"USA", "US"},
		{"GBR", "GB"},
		{"DEU", "DE"},
		{"FRA", "FR"},
		{"JPN", "JP"},
		{"CAN", "CA"},
		{"AUS", "AU"},
		{"BRA", "BR"},
		{"IND", "IN"},
		{"CHN", "CN"},
		{"unknown", ""},
		{"", ""},
		{"US", ""},  // already alpha-2 — function only accepts alpha-3
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, countryAlpha3ToAlpha2(tc.in))
		})
	}
}
```

- [ ] **Step 2: Run test — should fail (function not defined)**

Run: `go test ./modules/scope3/tmp/... -run TestCountryAlpha3ToAlpha2 -v`
Expected: FAIL "undefined: countryAlpha3ToAlpha2".

- [ ] **Step 3: Implement the conversion**

Create `modules/scope3/tmp/masking.go`:
```go
package tmp

// countryAlpha3ToAlpha2 converts an ISO 3166-1 alpha-3 code to alpha-2.
// Returns "" for unknown or empty input. Input is case-sensitive uppercase.
func countryAlpha3ToAlpha2(alpha3 string) string {
	return iso3166Alpha3ToAlpha2[alpha3]
}

// iso3166Alpha3ToAlpha2 is the static mapping of ISO 3166-1 alpha-3 codes
// (used by OpenRTB device.geo.country) to alpha-2 codes (required by TMP).
var iso3166Alpha3ToAlpha2 = map[string]string{
	"AFG": "AF", "ALA": "AX", "ALB": "AL", "DZA": "DZ", "ASM": "AS",
	"AND": "AD", "AGO": "AO", "AIA": "AI", "ATA": "AQ", "ATG": "AG",
	"ARG": "AR", "ARM": "AM", "ABW": "AW", "AUS": "AU", "AUT": "AT",
	"AZE": "AZ", "BHS": "BS", "BHR": "BH", "BGD": "BD", "BRB": "BB",
	"BLR": "BY", "BEL": "BE", "BLZ": "BZ", "BEN": "BJ", "BMU": "BM",
	"BTN": "BT", "BOL": "BO", "BES": "BQ", "BIH": "BA", "BWA": "BW",
	"BVT": "BV", "BRA": "BR", "IOT": "IO", "BRN": "BN", "BGR": "BG",
	"BFA": "BF", "BDI": "BI", "CPV": "CV", "KHM": "KH", "CMR": "CM",
	"CAN": "CA", "CYM": "KY", "CAF": "CF", "TCD": "TD", "CHL": "CL",
	"CHN": "CN", "CXR": "CX", "CCK": "CC", "COL": "CO", "COM": "KM",
	"COD": "CD", "COG": "CG", "COK": "CK", "CRI": "CR", "CIV": "CI",
	"HRV": "HR", "CUB": "CU", "CUW": "CW", "CYP": "CY", "CZE": "CZ",
	"DNK": "DK", "DJI": "DJ", "DMA": "DM", "DOM": "DO", "ECU": "EC",
	"EGY": "EG", "SLV": "SV", "GNQ": "GQ", "ERI": "ER", "EST": "EE",
	"SWZ": "SZ", "ETH": "ET", "FLK": "FK", "FRO": "FO", "FJI": "FJ",
	"FIN": "FI", "FRA": "FR", "GUF": "GF", "PYF": "PF", "ATF": "TF",
	"GAB": "GA", "GMB": "GM", "GEO": "GE", "DEU": "DE", "GHA": "GH",
	"GIB": "GI", "GRC": "GR", "GRL": "GL", "GRD": "GD", "GLP": "GP",
	"GUM": "GU", "GTM": "GT", "GGY": "GG", "GIN": "GN", "GNB": "GW",
	"GUY": "GY", "HTI": "HT", "HMD": "HM", "VAT": "VA", "HND": "HN",
	"HKG": "HK", "HUN": "HU", "ISL": "IS", "IND": "IN", "IDN": "ID",
	"IRN": "IR", "IRQ": "IQ", "IRL": "IE", "IMN": "IM", "ISR": "IL",
	"ITA": "IT", "JAM": "JM", "JPN": "JP", "JEY": "JE", "JOR": "JO",
	"KAZ": "KZ", "KEN": "KE", "KIR": "KI", "PRK": "KP", "KOR": "KR",
	"KWT": "KW", "KGZ": "KG", "LAO": "LA", "LVA": "LV", "LBN": "LB",
	"LSO": "LS", "LBR": "LR", "LBY": "LY", "LIE": "LI", "LTU": "LT",
	"LUX": "LU", "MAC": "MO", "MKD": "MK", "MDG": "MG", "MWI": "MW",
	"MYS": "MY", "MDV": "MV", "MLI": "ML", "MLT": "MT", "MHL": "MH",
	"MTQ": "MQ", "MRT": "MR", "MUS": "MU", "MYT": "YT", "MEX": "MX",
	"FSM": "FM", "MDA": "MD", "MCO": "MC", "MNG": "MN", "MNE": "ME",
	"MSR": "MS", "MAR": "MA", "MOZ": "MZ", "MMR": "MM", "NAM": "NA",
	"NRU": "NR", "NPL": "NP", "NLD": "NL", "NCL": "NC", "NZL": "NZ",
	"NIC": "NI", "NER": "NE", "NGA": "NG", "NIU": "NU", "NFK": "NF",
	"MNP": "MP", "NOR": "NO", "OMN": "OM", "PAK": "PK", "PLW": "PW",
	"PSE": "PS", "PAN": "PA", "PNG": "PG", "PRY": "PY", "PER": "PE",
	"PHL": "PH", "PCN": "PN", "POL": "PL", "PRT": "PT", "PRI": "PR",
	"QAT": "QA", "REU": "RE", "ROU": "RO", "RUS": "RU", "RWA": "RW",
	"BLM": "BL", "SHN": "SH", "KNA": "KN", "LCA": "LC", "MAF": "MF",
	"SPM": "PM", "VCT": "VC", "WSM": "WS", "SMR": "SM", "STP": "ST",
	"SAU": "SA", "SEN": "SN", "SRB": "RS", "SYC": "SC", "SLE": "SL",
	"SGP": "SG", "SXM": "SX", "SVK": "SK", "SVN": "SI", "SLB": "SB",
	"SOM": "SO", "ZAF": "ZA", "SGS": "GS", "SSD": "SS", "ESP": "ES",
	"LKA": "LK", "SDN": "SD", "SUR": "SR", "SJM": "SJ", "SWE": "SE",
	"CHE": "CH", "SYR": "SY", "TWN": "TW", "TJK": "TJ", "TZA": "TZ",
	"THA": "TH", "TLS": "TL", "TGO": "TG", "TKL": "TK", "TON": "TO",
	"TTO": "TT", "TUN": "TN", "TUR": "TR", "TKM": "TM", "TCA": "TC",
	"TUV": "TV", "UGA": "UG", "UKR": "UA", "ARE": "AE", "GBR": "GB",
	"USA": "US", "UMI": "UM", "URY": "UY", "UZB": "UZ", "VUT": "VU",
	"VEN": "VE", "VNM": "VN", "VGB": "VG", "VIR": "VI", "WLF": "WF",
	"ESH": "EH", "YEM": "YE", "ZMB": "ZM", "ZWE": "ZW",
}
```

- [ ] **Step 4: Run test — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestCountryAlpha3ToAlpha2 -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/masking.go modules/scope3/tmp/masking_test.go
git commit -m "Module: Scope3 TMP - ISO 3166-1 alpha-3 to alpha-2 country conversion"
```

---

## Task 6: Identity extraction capped at 3

**Files:**
- Modify: `modules/scope3/tmp/masking.go`
- Modify: `modules/scope3/tmp/masking_test.go`

- [ ] **Step 1: Write identity-extraction test**

Append to `modules/scope3/tmp/masking_test.go`:
```go
import (
	"encoding/json"
	// keep existing imports
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestExtractIdentities_RespectsOrderAndCap(t *testing.T) {
	tests := []struct {
		name         string
		preserveEids []string
		userExtJSON  string
		userID       string
		want         []IdentityToken
	}{
		{
			name:         "no user — empty",
			preserveEids: []string{"liveramp.com", "uidapi.com", "id5-sync.com"},
			userExtJSON:  ``,
			want:         nil,
		},
		{
			name:         "single liveramp eid",
			preserveEids: []string{"liveramp.com", "uidapi.com", "id5-sync.com"},
			userExtJSON:  `{"eids":[{"source":"liveramp.com","uids":[{"id":"RID-123"}]}]}`,
			want:         []IdentityToken{{UIDType: "liveramp.com", UserToken: "RID-123"}},
		},
		{
			name:         "all three preferred sources in order",
			preserveEids: []string{"liveramp.com", "uidapi.com", "id5-sync.com"},
			userExtJSON: `{"eids":[
				{"source":"id5-sync.com","uids":[{"id":"ID5-X"}]},
				{"source":"liveramp.com","uids":[{"id":"RID-1"}]},
				{"source":"uidapi.com","uids":[{"id":"UID-2"}]}
			]}`,
			want: []IdentityToken{
				{UIDType: "liveramp.com", UserToken: "RID-1"},
				{UIDType: "uidapi.com", UserToken: "UID-2"},
				{UIDType: "id5-sync.com", UserToken: "ID5-X"},
			},
		},
		{
			name:         "non-preferred source ignored",
			preserveEids: []string{"liveramp.com"},
			userExtJSON:  `{"eids":[{"source":"criteo.com","uids":[{"id":"X"}]}]}`,
			want:         nil,
		},
		{
			name:         "fallback to user.id when no eids and ext doesn't carry one",
			preserveEids: []string{"liveramp.com"},
			userExtJSON:  ``,
			userID:       "pub-uid-9",
			want:         []IdentityToken{{UIDType: "publisher_user_id", UserToken: "pub-uid-9"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var user *openrtb2.User
			if tc.userExtJSON != "" || tc.userID != "" {
				user = &openrtb2.User{ID: tc.userID}
				if tc.userExtJSON != "" {
					user.Ext = json.RawMessage(tc.userExtJSON)
				}
			}
			got := extractIdentities(user, tc.preserveEids)
			require.Equal(t, tc.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test — should fail (function not defined)**

Run: `go test ./modules/scope3/tmp/... -run TestExtractIdentities -v`
Expected: FAIL "undefined: extractIdentities".

- [ ] **Step 3: Implement identity extraction**

Append to `modules/scope3/tmp/masking.go`:
```go
import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// extractIdentities picks up to 3 identity tokens from the user object, in the
// order specified by preserveEids. Falls back to publisher user.id only when
// no eids match and user.id is non-empty.
//
// The spec hard-caps Identities at 3 entries (maxItems: 3) because of the TMPX
// HPKE plaintext byte budget. Builder validation already rejects preserveEids
// longer than 3, so this function trusts that bound.
func extractIdentities(user *openrtb2.User, preserveEids []string) []IdentityToken {
	if user == nil {
		return nil
	}

	var ext struct {
		EIDs []openrtb2.EID `json:"eids"`
	}
	if len(user.Ext) > 0 {
		_ = json.Unmarshal(user.Ext, &ext) // best effort; treat parse failure as no EIDs
	}

	bySource := make(map[string]string, len(ext.EIDs))
	for _, eid := range ext.EIDs {
		if len(eid.UIDs) == 0 {
			continue
		}
		if _, dup := bySource[eid.Source]; dup {
			continue
		}
		bySource[eid.Source] = eid.UIDs[0].ID
	}

	out := make([]IdentityToken, 0, len(preserveEids))
	for _, source := range preserveEids {
		if id, ok := bySource[source]; ok && id != "" {
			out = append(out, IdentityToken{UIDType: source, UserToken: id})
		}
	}

	if len(out) == 0 && user.ID != "" {
		out = append(out, IdentityToken{UIDType: "publisher_user_id", UserToken: user.ID})
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
```

- [ ] **Step 4: Run test — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestExtractIdentities -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/masking.go modules/scope3/tmp/masking_test.go
git commit -m "Module: Scope3 TMP - identity token extraction with preference order"
```

---

## Task 7: Cache key composition

**Files:**
- Create: `modules/scope3/tmp/async_request.go`
- Create: `modules/scope3/tmp/async_request_test.go`

- [ ] **Step 1: Write the cache key test**

Create `modules/scope3/tmp/async_request_test.go`:
```go
package tmp

import (
	"crypto/sha256"
	"sync"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/require"
)

func newPool() *sync.Pool {
	return &sync.Pool{New: func() any { return sha256.New() }}
}

func TestContextCacheKey_StableAndDistinct(t *testing.T) {
	pool := newPool()
	br := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "example.com", Page: "https://example.com/x"},
		User: &openrtb2.User{Ext: []byte(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R1"}]}]}`)},
	}
	a := contextCacheKey(pool, "rid_A", "place_1", br)
	b := contextCacheKey(pool, "rid_A", "place_1", br)
	require.Equal(t, a, b, "same inputs → same key")

	c := contextCacheKey(pool, "rid_B", "place_1", br)
	require.NotEqual(t, a, c, "different property_rid → different key")

	d := contextCacheKey(pool, "rid_A", "place_2", br)
	require.NotEqual(t, a, d, "different placement_id → different key")

	br2 := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "example.com", Page: "https://example.com/x"},
		User: &openrtb2.User{Ext: []byte(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R2"}]}]}`)},
	}
	e := contextCacheKey(pool, "rid_A", "place_1", br2)
	require.NotEqual(t, a, e, "different user identifier → different key")
}

func TestIdentityCacheKey_StableAndDistinct(t *testing.T) {
	pool := newPool()
	idents := []IdentityToken{{UIDType: "liveramp.com", UserToken: "R1"}}
	a := identityCacheKey(pool, "https://us", "US", idents)
	b := identityCacheKey(pool, "https://us", "US", idents)
	require.Equal(t, a, b)

	c := identityCacheKey(pool, "https://other", "US", idents)
	require.NotEqual(t, a, c)
}
```

- [ ] **Step 2: Run test — should fail (functions not defined)**

Run: `go test ./modules/scope3/tmp/... -run TestContextCacheKey -v`
Expected: FAIL "undefined: contextCacheKey".

- [ ] **Step 3: Implement cache key composition**

Create `modules/scope3/tmp/async_request.go`:
```go
package tmp

import (
	"encoding/hex"
	"encoding/json"
	"hash"
	"sync"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// contextCacheKey derives a stable hex string from inputs that scope a Context
// Match result. Same (property_rid, placement_id, page/app, privacy-safe ids)
// returns the same key.
func contextCacheKey(pool *sync.Pool, propertyRID, placementID string, br *openrtb2.BidRequest) string {
	h := pool.Get().(hash.Hash)
	defer pool.Put(h)
	h.Reset()

	_, _ = h.Write([]byte("p:" + propertyRID))
	_, _ = h.Write([]byte("|pl:" + placementID))
	writeSiteOrApp(h, br)
	writePrivacySafeUserIDs(h, br.User)
	return hex.EncodeToString(h.Sum(nil))
}

// identityCacheKey derives a stable hex string from inputs that scope an
// Identity Match result. Identity match results are page-context-free, so the
// key intentionally excludes site/app/placement.
func identityCacheKey(pool *sync.Pool, sellerAgentURL, country string, idents []IdentityToken) string {
	h := pool.Get().(hash.Hash)
	defer pool.Put(h)
	h.Reset()

	_, _ = h.Write([]byte("s:" + sellerAgentURL))
	_, _ = h.Write([]byte("|c:" + country))
	for _, t := range idents {
		_, _ = h.Write([]byte("|id:" + t.UIDType + "=" + t.UserToken))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeSiteOrApp(h hash.Hash, br *openrtb2.BidRequest) {
	if br.Site != nil {
		_, _ = h.Write([]byte("|d:" + br.Site.Domain))
		if br.Site.Page != "" {
			_, _ = h.Write([]byte("|pg:" + br.Site.Page))
		}
	}
	if br.App != nil {
		_, _ = h.Write([]byte("|a:" + br.App.Bundle))
	}
}

func writePrivacySafeUserIDs(h hash.Hash, user *openrtb2.User) {
	if user == nil {
		return
	}
	var ext struct {
		EIDs []openrtb2.EID `json:"eids"`
	}
	if len(user.Ext) > 0 {
		_ = json.Unmarshal(user.Ext, &ext)
	}
	for _, eid := range ext.EIDs {
		if len(eid.UIDs) > 0 {
			_, _ = h.Write([]byte("|e:" + eid.Source + "=" + eid.UIDs[0].ID))
		}
	}
}
```

- [ ] **Step 4: Run test — should pass**

Run: `go test ./modules/scope3/tmp/... -run "TestContextCacheKey|TestIdentityCacheKey" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/async_request.go modules/scope3/tmp/async_request_test.go
git commit -m "Module: Scope3 TMP - cache key composition for context and identity"
```

---

## Task 8: `intersect` pure function

**Files:**
- Modify: `modules/scope3/tmp/async_request.go`
- Modify: `modules/scope3/tmp/async_request_test.go`

- [ ] **Step 1: Write the intersect test**

Append to `modules/scope3/tmp/async_request_test.go`:
```go
func TestIntersect(t *testing.T) {
	tests := []struct {
		name           string
		contextOffers  []Offer
		identityElig   []string
		want           []string
	}{
		{
			name:          "both empty",
			contextOffers: nil,
			identityElig:  nil,
			want:          []string{},
		},
		{
			name:          "context empty",
			contextOffers: nil,
			identityElig:  []string{"pkg1"},
			want:          []string{},
		},
		{
			name:          "identity empty",
			contextOffers: []Offer{{PackageID: "pkg1"}},
			identityElig:  nil,
			want:          []string{},
		},
		{
			name:          "full overlap",
			contextOffers: []Offer{{PackageID: "pkg1"}, {PackageID: "pkg2"}},
			identityElig:  []string{"pkg2", "pkg1"},
			want:          []string{"pkg1", "pkg2"}, // order follows contextOffers
		},
		{
			name:          "partial overlap",
			contextOffers: []Offer{{PackageID: "pkg1"}, {PackageID: "pkg2"}, {PackageID: "pkg3"}},
			identityElig:  []string{"pkg2"},
			want:          []string{"pkg2"},
		},
		{
			name:          "no overlap",
			contextOffers: []Offer{{PackageID: "pkg1"}},
			identityElig:  []string{"pkg2"},
			want:          []string{},
		},
		{
			name:          "dedupe within context offers",
			contextOffers: []Offer{{PackageID: "pkg1"}, {PackageID: "pkg1"}, {PackageID: "pkg2"}},
			identityElig:  []string{"pkg1", "pkg2"},
			want:          []string{"pkg1", "pkg2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := intersect(tc.contextOffers, tc.identityElig)
			require.Equal(t, tc.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test — should fail**

Run: `go test ./modules/scope3/tmp/... -run TestIntersect -v`
Expected: FAIL "undefined: intersect".

- [ ] **Step 3: Implement intersect**

Append to `modules/scope3/tmp/async_request.go`:
```go
// intersect returns the package IDs that appear in both the context offers
// list and the identity-eligible list. Order follows the contextOffers; output
// is deduplicated. Returns an empty (non-nil) slice when either input is empty.
func intersect(contextOffers []Offer, identityEligible []string) []string {
	out := []string{}
	if len(contextOffers) == 0 || len(identityEligible) == 0 {
		return out
	}
	eligible := make(map[string]struct{}, len(identityEligible))
	for _, id := range identityEligible {
		eligible[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(contextOffers))
	for _, offer := range contextOffers {
		if _, alreadyEmitted := seen[offer.PackageID]; alreadyEmitted {
			continue
		}
		if _, ok := eligible[offer.PackageID]; ok {
			out = append(out, offer.PackageID)
			seen[offer.PackageID] = struct{}{}
		}
	}
	return out
}
```

- [ ] **Step 4: Run test — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestIntersect -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/async_request.go modules/scope3/tmp/async_request_test.go
git commit -m "Module: Scope3 TMP - intersect for publisher-side package eligibility join"
```

---

## Task 9: Bid-request masking adapted from RTD

**Files:**
- Modify: `modules/scope3/tmp/masking.go`
- Modify: `modules/scope3/tmp/masking_test.go`

- [ ] **Step 1: Write masking test**

Append to `modules/scope3/tmp/masking_test.go`:
```go
func TestMaskBidRequest_StripsSensitiveFields(t *testing.T) {
	cfg := MaskingConfig{
		Enabled: true,
		Geo:     GeoMaskingConfig{PreserveMetro: true, PreserveZip: true, PreserveCity: false, LatLongPrecision: 2},
		User:    UserMaskingConfig{PreserveEids: []string{"liveramp.com"}},
		Device:  DeviceMaskingConfig{PreserveMobileIds: false},
	}
	br := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IP:    "73.158.22.41",
			IPv6:  "2001:db8::1",
			UA:    "ua",
			OS:    "macOS",
			IFA:   "A1B2-C3D4",
			Geo:   &openrtb2.Geo{Country: "USA", Region: "NY", City: "NYC", Metro: "501", ZIP: "10001", Lat: 40.7128, Lon: -74.0059, Accuracy: 5},
		},
		User: &openrtb2.User{
			ID:       "uid",
			BuyerUID: "bid",
			Yob:      1985,
			Gender:   "M",
			Keywords: "kw",
			Ext: []byte(`{"eids":[
				{"source":"liveramp.com","uids":[{"id":"keep"}]},
				{"source":"criteo.com","uids":[{"id":"drop"}]}
			]}`),
		},
	}

	masked := maskBidRequest(br, cfg)
	require.NotNil(t, masked)

	require.Empty(t, masked.Device.IP)
	require.Empty(t, masked.Device.IPv6)
	require.Empty(t, masked.Device.IFA)
	require.NotEmpty(t, masked.Device.UA)
	require.NotEmpty(t, masked.Device.OS)

	require.Equal(t, "USA", masked.Device.Geo.Country)
	require.Equal(t, "NY", masked.Device.Geo.Region)
	require.Empty(t, masked.Device.Geo.City)
	require.Equal(t, "501", masked.Device.Geo.Metro)
	require.Equal(t, "10001", masked.Device.Geo.ZIP)
	require.InDelta(t, 40.71, masked.Device.Geo.Lat, 0.001)
	require.InDelta(t, -74.01, masked.Device.Geo.Lon, 0.001)
	require.Zero(t, masked.Device.Geo.Accuracy)

	require.Empty(t, masked.User.ID)
	require.Empty(t, masked.User.BuyerUID)
	require.Zero(t, masked.User.Yob)
	require.Empty(t, masked.User.Gender)
	require.Empty(t, masked.User.Keywords)
}

func TestMaskBidRequest_DisabledIsPassthrough(t *testing.T) {
	cfg := MaskingConfig{Enabled: false}
	br := &openrtb2.BidRequest{Device: &openrtb2.Device{IP: "73.158.22.41"}}
	masked := maskBidRequest(br, cfg)
	require.Equal(t, "73.158.22.41", masked.Device.IP)
}
```

- [ ] **Step 2: Run test — should fail**

Run: `go test ./modules/scope3/tmp/... -run TestMaskBidRequest -v`
Expected: FAIL "undefined: maskBidRequest".

- [ ] **Step 3: Implement masking**

Append to `modules/scope3/tmp/masking.go`:
```go
import "math"

// maskBidRequest returns a deep copy with sensitive fields removed per config.
// Returns the original (unmodified) if masking is disabled.
func maskBidRequest(orig *openrtb2.BidRequest, cfg MaskingConfig) *openrtb2.BidRequest {
	if !cfg.Enabled {
		return orig
	}
	raw, err := json.Marshal(orig)
	if err != nil {
		return nil
	}
	var copy openrtb2.BidRequest
	if err := json.Unmarshal(raw, &copy); err != nil {
		return nil
	}
	maskUser(&copy, cfg.User)
	maskDevice(&copy, cfg.Device)
	maskGeo(&copy, cfg.Geo)
	return &copy
}

func maskUser(br *openrtb2.BidRequest, cfg UserMaskingConfig) {
	if br.User == nil {
		return
	}
	br.User.ID = ""
	br.User.BuyerUID = ""
	br.User.Yob = 0
	br.User.Gender = ""
	br.User.Data = nil
	br.User.Keywords = ""
	br.User.EIDs = filterEIDs(br.User.EIDs, cfg.PreserveEids)
}

func filterEIDs(eids []openrtb2.EID, allow []string) []openrtb2.EID {
	if len(allow) == 0 {
		return nil
	}
	allowSet := make(map[string]struct{}, len(allow))
	for _, a := range allow {
		allowSet[a] = struct{}{}
	}
	out := make([]openrtb2.EID, 0, len(eids))
	for _, e := range eids {
		if _, ok := allowSet[e.Source]; ok {
			out = append(out, e)
		}
	}
	return out
}

func maskDevice(br *openrtb2.BidRequest, cfg DeviceMaskingConfig) {
	if br.Device == nil {
		return
	}
	br.Device.IP = ""
	br.Device.IPv6 = ""
	if !cfg.PreserveMobileIds {
		br.Device.IFA = ""
		br.Device.DPIDMD5 = ""
		br.Device.DPIDSHA1 = ""
		br.Device.MACMD5 = ""
		br.Device.MACSHA1 = ""
		br.Device.DIDMD5 = ""
		br.Device.DIDSHA1 = ""
	}
}

func maskGeo(br *openrtb2.BidRequest, cfg GeoMaskingConfig) {
	if br.Device == nil || br.Device.Geo == nil {
		return
	}
	g := br.Device.Geo
	if !cfg.PreserveMetro {
		g.Metro = ""
	}
	if !cfg.PreserveZip {
		g.ZIP = ""
	}
	if !cfg.PreserveCity {
		g.City = ""
	}
	g.Accuracy = 0
	g.Lat = truncateLatLong(g.Lat, cfg.LatLongPrecision)
	g.Lon = truncateLatLong(g.Lon, cfg.LatLongPrecision)
}

func truncateLatLong(v float64, precision int) float64 {
	if precision <= 0 {
		return 0
	}
	mult := math.Pow(10, float64(precision))
	return math.Round(v*mult) / mult
}
```

- [ ] **Step 4: Run test — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestMaskBidRequest -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/masking.go modules/scope3/tmp/masking_test.go
git commit -m "Module: Scope3 TMP - bid request masking adapted from RTD"
```

---

## Task 10: AsyncRequest scaffolding

**Files:**
- Modify: `modules/scope3/tmp/async_request.go`
- Modify: `modules/scope3/tmp/async_request_test.go`

- [ ] **Step 1: Write a lifecycle test**

Append to `modules/scope3/tmp/async_request_test.go`:
```go
import (
	"context"
)

func TestAsyncRequest_LifecycleNoFetch(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar := newAsyncRequest(parent)
	require.NotNil(t, ar)
	require.NotNil(t, ar.ctx)
	require.NotNil(t, ar.cancel)

	// No fetch was called; Done channel should be nil.
	require.Nil(t, ar.done)

	ar.cancel()
}
```

- [ ] **Step 2: Run — should fail**

Run: `go test ./modules/scope3/tmp/... -run TestAsyncRequest_LifecycleNoFetch -v`
Expected: FAIL "undefined: newAsyncRequest".

- [ ] **Step 3: Implement AsyncRequest scaffolding**

Append to `modules/scope3/tmp/async_request.go`:
```go
import (
	"context"
)

// AsyncResult is the data the auction response hook reads after fan-out.
type AsyncResult struct {
	PerPlacement   map[string]PlacementResult // placement_id → result
	ImpToPlacement map[string]string          // imp.id → placement_id
	TMPX           string
}

// PlacementResult holds the per-placement enrichment that ends up on
// each bid whose impid maps to this placement.
type PlacementResult struct {
	EligiblePackages []string
	TargetingKVs     []KeyValuePair
	Segments         []string
}

// AsyncRequest is per-auction state created in HandleEntrypointHook and
// drained in HandleAuctionResponseHook.
type AsyncRequest struct {
	module *Module
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	result *AsyncResult
	err    error
}

// newAsyncRequest creates per-auction state. Done is nil until fetchAsync
// runs — the auction-response hook must check for nil before reading.
func newAsyncRequest(parent context.Context) *AsyncRequest {
	ctx, cancel := context.WithCancel(parent)
	return &AsyncRequest{ctx: ctx, cancel: cancel}
}
```

- [ ] **Step 4: Run — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestAsyncRequest_LifecycleNoFetch -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/async_request.go modules/scope3/tmp/async_request_test.go
git commit -m "Module: Scope3 TMP - AsyncRequest per-auction state scaffolding"
```

---

## Task 11: HTTP call helpers — fetchContext and fetchIdentity

**Files:**
- Modify: `modules/scope3/tmp/async_request.go`
- Modify: `modules/scope3/tmp/async_request_test.go`

- [ ] **Step 1: Write HTTP-level tests**

Append to `modules/scope3/tmp/async_request_test.go`:
```go
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func TestFetchContext_HappyPath(t *testing.T) {
	want := ContextMatchResponse{
		Type:      TypeContextMatchResponse,
		RequestID: "req-x",
		Offers:    []Offer{{PackageID: "pkg_abc"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/tmp/context", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	req := ContextMatchRequest{
		Type:        TypeContextMatchRequest,
		RequestID:   "req-x",
		PropertyRID: "rid",
		PlacementID: "pl",
	}
	got, err := fetchContext(context.Background(), &http.Client{}, srv.URL, "", &req)
	require.NoError(t, err)
	require.Equal(t, want.RequestID, got.RequestID)
	require.Equal(t, "pkg_abc", got.Offers[0].PackageID)
}

func TestFetchContext_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := fetchContext(context.Background(), &http.Client{}, srv.URL, "", &ContextMatchRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "400")
}

func TestFetchIdentity_HappyPath(t *testing.T) {
	want := IdentityMatchResponse{
		Type:               TypeIdentityMatchResponse,
		RequestID:          "id-y",
		EligiblePackageIDs: []string{"pkg_abc"},
		Tmpx:               "k1.xyz",
		TTLSec:             60,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/tmp/identity", r.URL.Path)

		var body IdentityMatchRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Empty(t, body.RequestID == "")
		require.Equal(t, "auth-token", r.Header.Get("x-scope3-auth"))
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	req := IdentityMatchRequest{Type: TypeIdentityMatchRequest, RequestID: "id-y", SellerAgentURL: "https://us"}
	got, err := fetchIdentity(context.Background(), &http.Client{}, srv.URL, "auth-token", &req)
	require.NoError(t, err)
	require.Equal(t, want.RequestID, got.RequestID)
	require.Equal(t, "k1.xyz", got.Tmpx)
}
```

- [ ] **Step 2: Run — should fail**

Run: `go test ./modules/scope3/tmp/... -run "TestFetchContext|TestFetchIdentity" -v`
Expected: FAIL "undefined: fetchContext".

- [ ] **Step 3: Implement the two HTTP helpers**

Append to `modules/scope3/tmp/async_request.go`:
```go
import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func fetchContext(ctx context.Context, client *http.Client, routerURL, authKey string, req *ContextMatchRequest) (*ContextMatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode context request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, routerURL+"/tmp/context", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authKey != "" {
		httpReq.Header.Set("x-scope3-auth", authKey)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("context match returned status %d: %s", resp.StatusCode, string(body))
	}

	var out ContextMatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode context response: %w", err)
	}
	return &out, nil
}

func fetchIdentity(ctx context.Context, client *http.Client, routerURL, authKey string, req *IdentityMatchRequest) (*IdentityMatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode identity request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, routerURL+"/tmp/identity", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authKey != "" {
		httpReq.Header.Set("x-scope3-auth", authKey)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("identity match returned status %d: %s", resp.StatusCode, string(body))
	}

	var out IdentityMatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode identity response: %w", err)
	}
	return &out, nil
}
```

- [ ] **Step 4: Run — should pass**

Run: `go test ./modules/scope3/tmp/... -run "TestFetchContext|TestFetchIdentity" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/async_request.go modules/scope3/tmp/async_request_test.go
git commit -m "Module: Scope3 TMP - fetchContext and fetchIdentity HTTP helpers"
```

---

## Task 12: Asymmetric N+1 fan-out, intersection, and result assembly

**Files:**
- Modify: `modules/scope3/tmp/async_request.go`
- Modify: `modules/scope3/tmp/async_request_test.go`

- [ ] **Step 1: Write an integration test for fetchAsync**

Append to `modules/scope3/tmp/async_request_test.go`:
```go
import (
	"sync/atomic"
)

func TestFetchAsync_MultiImpThreePlacements_HappyPath(t *testing.T) {
	var ctxCalls, idCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tmp/context":
			ctxCalls.Add(1)
			var req ContextMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(ContextMatchResponse{
				Type:      TypeContextMatchResponse,
				RequestID: req.RequestID,
				Offers:    []Offer{{PackageID: "pkg_" + req.PlacementID}},
				Signals:   Signals{TargetingKVs: []KeyValuePair{{Key: "k_" + req.PlacementID, Value: "v"}}},
			})
		case "/tmp/identity":
			idCalls.Add(1)
			var req IdentityMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(IdentityMatchResponse{
				Type:               TypeIdentityMatchResponse,
				RequestID:          req.RequestID,
				EligiblePackageIDs: []string{"pkg_header_728x90", "pkg_preroll_video"},
				Tmpx:               "k1.token",
			})
		}
	}))
	defer srv.Close()

	mod, err := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	require.NoError(t, err)
	m := mod.(*Module)

	br := &openrtb2.BidRequest{
		ID: "auction-1",
		Imp: []openrtb2.Imp{
			{ID: "imp1", TagID: "header"},
			{ID: "imp2", TagID: "sidebar"},
			{ID: "imp3", TagID: "video"},
		},
		Site: &openrtb2.Site{Domain: "example.com", Page: "https://example.com/x"},
		User: &openrtb2.User{Ext: json.RawMessage(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R1"}]}]}`)},
		Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}},
	}
	accountCfg := json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"header":"header_728x90","sidebar":"sidebar_300x250","video":"preroll_video"}}}}`)

	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(br, accountCfg, json.RawMessage(`{}`))
	<-ar.done

	require.NoError(t, ar.err)
	require.NotNil(t, ar.result)
	require.Equal(t, int32(3), ctxCalls.Load(), "one context call per unique placement")
	require.Equal(t, int32(1), idCalls.Load(), "exactly one identity call regardless of imp count")

	require.Equal(t, "k1.token", ar.result.TMPX)
	require.Equal(t, []string{"pkg_header_728x90"}, ar.result.PerPlacement["header_728x90"].EligiblePackages)
	require.Equal(t, []string{"pkg_preroll_video"}, ar.result.PerPlacement["preroll_video"].EligiblePackages)
	require.Empty(t, ar.result.PerPlacement["sidebar_300x250"].EligiblePackages, "sidebar pkg not in identity eligible set")

	require.Equal(t, "header_728x90", ar.result.ImpToPlacement["imp1"])
}

func TestFetchAsync_SharedPlacementDeduped(t *testing.T) {
	var ctxCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tmp/context":
			ctxCalls.Add(1)
			var req ContextMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(ContextMatchResponse{Type: TypeContextMatchResponse, RequestID: req.RequestID, Offers: []Offer{{PackageID: "pkg_shared"}}})
		case "/tmp/identity":
			var req IdentityMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(IdentityMatchResponse{Type: TypeIdentityMatchResponse, RequestID: req.RequestID, EligiblePackageIDs: []string{"pkg_shared"}})
		}
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)
	br := &openrtb2.BidRequest{
		ID: "a",
		Imp: []openrtb2.Imp{{ID: "i1", TagID: "h"}, {ID: "i2", TagID: "h"}},
		Site: &openrtb2.Site{Domain: "x.com"},
	}
	cfg := json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"shared"}}}}`)
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(br, cfg, nil)
	<-ar.done
	require.Equal(t, int32(1), ctxCalls.Load(), "shared placement dedupes to one context call")
}

func TestFetchAsync_PartialFailure_P1Strict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tmp/identity" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		var req ContextMatchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(ContextMatchResponse{Type: TypeContextMatchResponse, RequestID: req.RequestID, Offers: []Offer{{PackageID: "pkg_a"}}})
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)
	br := &openrtb2.BidRequest{
		ID: "a",
		Imp: []openrtb2.Imp{{ID: "i1", TagID: "h"}},
		Site: &openrtb2.Site{Domain: "x.com"},
	}
	cfg := json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"p"}}}}`)
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(br, cfg, nil)
	<-ar.done

	require.Error(t, ar.err, "P1 strict: identity failure means whole fetch is errored")
	require.Nil(t, ar.result)
}
```

- [ ] **Step 2: Run — should fail**

Run: `go test ./modules/scope3/tmp/... -run TestFetchAsync -v`
Expected: FAIL "undefined: fetchAsync".

- [ ] **Step 3: Add the errgroup dependency check and implement fetchAsync**

First confirm the `errgroup` package is already in go.mod:
```bash
grep "golang.org/x/sync" go.mod
```
Expected: a line like `golang.org/x/sync v0.X.X` is present (it is — used elsewhere in prebid-server).

Append to `modules/scope3/tmp/async_request.go`:
```go
import (
	"github.com/gofrs/uuid"
	"golang.org/x/sync/errgroup"
)

// fetchAsync runs the full N+1 fan-out in a goroutine. The Done channel is
// closed when the result (or error) is ready. Callers should:
//   - wait on <-ar.done (or <-ar.ctx.Done() for graceful timeout)
//   - read ar.result OR ar.err
//   - call ar.cancel() to release the context.
func (ar *AsyncRequest) fetchAsync(br *openrtb2.BidRequest, accountCfg json.RawMessage, requestExt json.RawMessage) {
	ar.done = make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ar.err = fmt.Errorf("panic in fetchAsync: %v", r)
			}
			close(ar.done)
		}()
		ar.run(br, accountCfg, requestExt)
	}()
}

func (ar *AsyncRequest) run(br *openrtb2.BidRequest, accountCfg, requestExt json.RawMessage) {
	resolver := accountResolver{accountConfig: accountCfg, requestExt: requestExt, moduleCfg: ar.module.cfg}
	ids, err := resolver.resolveAuction()
	if err != nil {
		ar.err = err
		return
	}

	// Resolve per-imp placements; dedupe.
	impToPlacement := make(map[string]string, len(br.Imp))
	uniquePlacements := []string{}
	seenPlacement := map[string]struct{}{}
	for _, imp := range br.Imp {
		place, ok := resolver.resolvePlacement(imp.TagID)
		if !ok || place == "" {
			continue
		}
		impToPlacement[imp.ID] = place
		if _, dup := seenPlacement[place]; !dup {
			seenPlacement[place] = struct{}{}
			uniquePlacements = append(uniquePlacements, place)
		}
	}
	if len(uniquePlacements) == 0 {
		ar.err = errors.New("no placements resolved for any imp")
		return
	}

	masked := br
	if ar.module.cfg.Masking.Enabled {
		masked = maskBidRequest(br, ar.module.cfg.Masking)
		if masked == nil {
			ar.err = errors.New("masking failed; refusing to send unmasked request")
			return
		}
	}

	identities := extractIdentities(masked.User, ar.module.cfg.Masking.User.PreserveEids)
	country := ""
	if masked.Device != nil && masked.Device.Geo != nil {
		country = countryAlpha3ToAlpha2(masked.Device.Geo.Country)
	}

	contextResults := make(map[string]*ContextMatchResponse, len(uniquePlacements))
	var contextMu sync.Mutex
	var identityResp *IdentityMatchResponse

	g, gctx := errgroup.WithContext(ar.ctx)

	for _, placement := range uniquePlacements {
		placement := placement
		g.Go(func() error {
			req := &ContextMatchRequest{
				Type:         TypeContextMatchRequest,
				RequestID:    mustUUID(),
				PropertyRID:  ids.PropertyRID,
				PropertyType: ids.PropertyType,
				PlacementID:  placement,
			}
			if masked.Site != nil && masked.Site.Page != "" {
				req.ArtifactRefs = []ArtifactRef{{URL: masked.Site.Page}}
			}
			resp, err := fetchContext(gctx, ar.module.httpClient, ids.RouterURL, ar.module.cfg.AuthKey, req)
			if err != nil {
				return fmt.Errorf("context placement=%s: %w", placement, err)
			}
			contextMu.Lock()
			contextResults[placement] = resp
			contextMu.Unlock()
			return nil
		})
	}

	g.Go(func() error {
		req := &IdentityMatchRequest{
			Type:           TypeIdentityMatchRequest,
			RequestID:      mustUUID(),
			SellerAgentURL: ids.SellerAgentURL,
			Identities:     identities,
			Country:        country,
		}
		resp, err := fetchIdentity(gctx, ar.module.httpClient, ids.RouterURL, ar.module.cfg.AuthKey, req)
		if err != nil {
			return fmt.Errorf("identity: %w", err)
		}
		identityResp = resp
		return nil
	})

	if err := g.Wait(); err != nil {
		ar.err = err
		return
	}

	perPlacement := make(map[string]PlacementResult, len(contextResults))
	identityElig := []string{}
	if identityResp != nil {
		identityElig = identityResp.EligiblePackageIDs
	}
	for placement, ctxResp := range contextResults {
		perPlacement[placement] = PlacementResult{
			EligiblePackages: intersect(ctxResp.Offers, identityElig),
			TargetingKVs:     ctxResp.Signals.TargetingKVs,
			Segments:         ctxResp.Signals.Segments,
		}
	}

	tmpx := ""
	if identityResp != nil {
		tmpx = identityResp.Tmpx
	}
	ar.result = &AsyncResult{
		PerPlacement:   perPlacement,
		ImpToPlacement: impToPlacement,
		TMPX:           tmpx,
	}
}

func mustUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return "" // extremely rare; downstream handles empty
	}
	return u.String()
}
```

- [ ] **Step 4: Run — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestFetchAsync -v`
Expected: PASS for all three subtests (happy path, dedupe, P1 strict).

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/async_request.go modules/scope3/tmp/async_request_test.go
git commit -m "Module: Scope3 TMP - asymmetric N+1 fan-out with errgroup and P1 strict"
```

---

## Task 13: HandleEntrypointHook

**Files:**
- Modify: `modules/scope3/tmp/module.go`
- Modify: `modules/scope3/tmp/module_test.go`

- [ ] **Step 1: Write the entrypoint hook test**

Append to `modules/scope3/tmp/module_test.go`:
```go
import (
	"context"
	"net/http/httptest"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

const asyncRequestKey = "scope3.tmp.AsyncRequest"

func TestHandleEntrypointHook_StoresAsyncRequest(t *testing.T) {
	mod, err := Builder(json.RawMessage(`{"router_url":"https://r","seller_agent_url":"https://us"}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	require.NoError(t, err)
	m := mod.(*Module)

	miCtx := hookstage.ModuleInvocationContext{}
	payload := hookstage.EntrypointPayload{Request: httptest.NewRequest("POST", "/openrtb2/auction", nil)}
	result, err := m.HandleEntrypointHook(context.Background(), miCtx, payload)
	require.NoError(t, err)
	require.NotNil(t, result.ModuleContext)

	stored, ok := result.ModuleContext.Get(asyncRequestKey)
	require.True(t, ok)
	_, isAR := stored.(*AsyncRequest)
	require.True(t, isAR)
}
```

- [ ] **Step 2: Run — should fail**

Run: `go test ./modules/scope3/tmp/... -run TestHandleEntrypointHook -v`
Expected: FAIL — `HandleEntrypointHook` undefined.

- [ ] **Step 3: Implement HandleEntrypointHook and the asyncRequestKey constant**

Append to `modules/scope3/tmp/module.go`:
```go
import (
	"context"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

const moduleContextAsyncKey = "scope3.tmp.AsyncRequest"

// Interface assertions.
var (
	_ hookstage.Entrypeint              = (*Module)(nil) // intentional typo will fail compile if interface name drifts — see step 4
)

// HandleEntrypointHook initializes per-auction state.
func (m *Module) HandleEntrypointHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(payload.Request.Context())
	ar.module = m
	mc.Set(moduleContextAsyncKey, ar)
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: mc}, nil
}
```

- [ ] **Step 4: Fix the interface assertion (verify the real interface name from prebid-server)**

Replace the deliberately-bad assertion in the previous step with the correct one. Look up the real name:
```bash
grep -n "type Entrypoint " hooks/hookstage/*.go
```
Expected: a line showing `type Entrypoint interface { ... }`.

Replace in `module.go`:
```go
var (
	_ hookstage.Entrypoint = (*Module)(nil)
)
```

Also update the test constant to match the module's exported constant — replace `const asyncRequestKey = ...` in the test file with importing the module's constant (already used internally). Since the test is in the same package, replace:
```go
const asyncRequestKey = "scope3.tmp.AsyncRequest"
```
with:
```go
// asyncRequestKey is just an alias for the internal constant; the test reads
// the same key from ModuleContext.
const asyncRequestKey = moduleContextAsyncKey
```

- [ ] **Step 5: Run — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestHandleEntrypointHook -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add modules/scope3/tmp/module.go modules/scope3/tmp/module_test.go
git commit -m "Module: Scope3 TMP - HandleEntrypointHook initializes per-auction state"
```

---

## Task 14: HandleProcessedAuctionHook

**Files:**
- Modify: `modules/scope3/tmp/module.go`
- Modify: `modules/scope3/tmp/module_test.go`

- [ ] **Step 1: Write the processed-auction hook test**

Append to `modules/scope3/tmp/module_test.go`:
```go
import (
	"sync/atomic"
)

func TestHandleProcessedAuctionHook_KicksOffGoroutine(t *testing.T) {
	var ctxHit atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tmp/context" {
			ctxHit.Store(true)
		}
		var rid string
		if r.URL.Path == "/tmp/context" {
			var req ContextMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			rid = req.RequestID
		} else {
			var req IdentityMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			rid = req.RequestID
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "x", "request_id": rid, "offers": []any{}, "eligible_package_ids": []any{}})
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	mc.Set(moduleContextAsyncKey, ar)

	br := &openrtb2.BidRequest{
		ID:   "a",
		Imp:  []openrtb2.Imp{{ID: "i", TagID: "h"}},
		Site: &openrtb2.Site{Domain: "x.com"},
	}
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: mc,
		AccountConfig: json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"p"}}}}`),
	}
	payload := hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: br}}
	_, err := m.HandleProcessedAuctionHook(context.Background(), miCtx, payload)
	require.NoError(t, err)

	<-ar.done
	require.True(t, ctxHit.Load(), "context endpoint was called from the goroutine")
}
```

Add the necessary import: `"github.com/prebid/prebid-server/v4/openrtb_ext"`.

- [ ] **Step 2: Run — should fail**

Run: `go test ./modules/scope3/tmp/... -run TestHandleProcessedAuctionHook -v`
Expected: FAIL — `HandleProcessedAuctionHook` undefined.

- [ ] **Step 3: Implement HandleProcessedAuctionHook**

Append to `modules/scope3/tmp/module.go`:
```go
var (
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
)

// HandleProcessedAuctionHook starts the asynchronous TMP fan-out. Returns
// immediately while the goroutine runs in parallel with the bidder auction.
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	var ret hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]

	stored, ok := miCtx.ModuleContext.Get(moduleContextAsyncKey)
	if !ok {
		return ret, nil
	}
	ar, ok := stored.(*AsyncRequest)
	if !ok {
		return ret, nil
	}

	requestExt := json.RawMessage(nil)
	if payload.Request != nil && payload.Request.BidRequest != nil {
		requestExt = payload.Request.BidRequest.Ext
	}

	ar.fetchAsync(payload.Request.BidRequest, miCtx.AccountConfig, requestExt)
	return ret, nil
}
```

- [ ] **Step 4: Run — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestHandleProcessedAuctionHook -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/module.go modules/scope3/tmp/module_test.go
git commit -m "Module: Scope3 TMP - HandleProcessedAuctionHook starts async fan-out"
```

---

## Task 15: HandleAuctionResponseHook with per-imp mutation

**Files:**
- Modify: `modules/scope3/tmp/module.go`
- Modify: `modules/scope3/tmp/module_test.go`

- [ ] **Step 1: Write the auction-response hook test**

Append to `modules/scope3/tmp/module_test.go`:
```go
import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/tidwall/gjson"
)

func TestHandleAuctionResponseHook_WritesPerBidExt(t *testing.T) {
	mod, _ := Builder(json.RawMessage(`{"router_url":"https://r","seller_agent_url":"https://us","add_to_targeting":true}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.done = make(chan struct{})
	close(ar.done)
	ar.result = &AsyncResult{
		PerPlacement: map[string]PlacementResult{
			"header_728x90": {EligiblePackages: []string{"pkg_abc"}, TargetingKVs: []KeyValuePair{{Key: "buyer_kv", Value: "v1"}}, Segments: []string{"seg_a"}},
		},
		ImpToPlacement: map[string]string{"imp1": "header_728x90"},
		TMPX:           "k1.token",
	}
	mc.Set(moduleContextAsyncKey, ar)

	resp := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{ID: "b1", ImpID: "imp1", Ext: json.RawMessage(`{}`)}},
		}},
		Ext: json.RawMessage(`{}`),
	}
	payload := hookstage.AuctionResponsePayload{BidResponse: resp}
	miCtx := hookstage.ModuleInvocationContext{ModuleContext: mc}

	result, err := m.HandleAuctionResponseHook(context.Background(), miCtx, payload)
	require.NoError(t, err)

	// Apply the mutations from the ChangeSet, like Prebid does in production.
	for _, mut := range result.ChangeSet.Mutations() {
		payload, _ = mut.Apply(payload)
	}

	respExt := gjson.GetBytes(payload.BidResponse.Ext, "scope3.tmp.tmpx")
	require.Equal(t, "k1.token", respExt.String())

	bidExt := payload.BidResponse.SeatBid[0].Bid[0].Ext
	require.Equal(t, "header_728x90", gjson.GetBytes(bidExt, "scope3.tmp.placement_id").String())
	require.Equal(t, "pkg_abc", gjson.GetBytes(bidExt, "scope3.tmp.eligible_packages.0").String())
	require.Equal(t, "k1.token", gjson.GetBytes(bidExt, "prebid.targeting.TMPX").String())
	require.Equal(t, "v1", gjson.GetBytes(bidExt, "prebid.targeting.buyer_kv").String())
}

func TestHandleAuctionResponseHook_PartialFailureNoMutation(t *testing.T) {
	mod, _ := Builder(json.RawMessage(`{"router_url":"https://r","seller_agent_url":"https://us"}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.done = make(chan struct{})
	close(ar.done)
	ar.err = errors.New("identity: failed")
	mc.Set(moduleContextAsyncKey, ar)

	resp := &openrtb2.BidResponse{Ext: json.RawMessage(`{}`)}
	payload := hookstage.AuctionResponsePayload{BidResponse: resp}
	miCtx := hookstage.ModuleInvocationContext{ModuleContext: mc}

	result, _ := m.HandleAuctionResponseHook(context.Background(), miCtx, payload)
	require.Empty(t, result.ChangeSet.Mutations(), "P1 strict: no mutation on error")
}
```

- [ ] **Step 2: Run — should fail**

Run: `go test ./modules/scope3/tmp/... -run TestHandleAuctionResponseHook -v`
Expected: FAIL — `HandleAuctionResponseHook` undefined.

- [ ] **Step 3: Implement HandleAuctionResponseHook**

Append to `modules/scope3/tmp/module.go`:
```go
import (
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/util/iterutil"
	"github.com/tidwall/sjson"
)

var (
	_ hookstage.AuctionResponse = (*Module)(nil)
)

// HandleAuctionResponseHook waits for the async fan-out to complete (bounded
// by the hook's context) and writes per-imp enrichment into the response.
func (m *Module) HandleAuctionResponseHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	var ret hookstage.HookResult[hookstage.AuctionResponsePayload]

	stored, ok := miCtx.ModuleContext.Get(moduleContextAsyncKey)
	if !ok {
		return ret, nil
	}
	ar, ok := stored.(*AsyncRequest)
	if !ok {
		return ret, nil
	}
	defer ar.cancel()

	if ar.done == nil {
		// Processed-auction hook never ran (e.g., test bypass). Nothing to write.
		return ret, nil
	}

	select {
	case <-ar.done:
	case <-ctx.Done():
		ret.AnalyticsTags = analyticsErrorTag("scope3_tmp_timeout", "auction context cancelled")
		return ret, nil
	}

	if ar.err != nil {
		ret.AnalyticsTags = analyticsErrorTag("scope3_tmp_fetch", ar.err.Error())
		return ret, nil
	}
	if ar.result == nil {
		return ret, nil
	}

	result := ar.result
	addToTargeting := m.cfg.AddToTargeting

	ret.ChangeSet.AddMutation(
		func(p hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			if p.BidResponse.Ext == nil {
				p.BidResponse.Ext = []byte("{}")
			}
			if result.TMPX != "" {
				p.BidResponse.Ext, _ = sjson.SetBytes(p.BidResponse.Ext, "scope3.tmp.tmpx", result.TMPX)
			}

			for seatBid := range iterutil.SlicePointerValues(p.BidResponse.SeatBid) {
				for bid := range iterutil.SlicePointerValues(seatBid.Bid) {
					placement, ok := result.ImpToPlacement[bid.ImpID]
					if !ok {
						continue
					}
					pkg := result.PerPlacement[placement]
					if bid.Ext == nil {
						bid.Ext = []byte("{}")
					}
					bid.Ext, _ = sjson.SetBytes(bid.Ext, "scope3.tmp.placement_id", placement)
					bid.Ext, _ = sjson.SetBytes(bid.Ext, "scope3.tmp.eligible_packages", pkg.EligiblePackages)
					if len(pkg.Segments) > 0 {
						bid.Ext, _ = sjson.SetBytes(bid.Ext, "scope3.tmp.segments", pkg.Segments)
					}
					if addToTargeting {
						if result.TMPX != "" {
							bid.Ext, _ = sjson.SetBytes(bid.Ext, "prebid.targeting.TMPX", result.TMPX)
						}
						for _, kv := range pkg.TargetingKVs {
							bid.Ext, _ = sjson.SetBytes(bid.Ext, "prebid.targeting."+kv.Key, kv.Value)
						}
					}
				}
			}
			return p, nil
		},
		hookstage.MutationUpdate,
		"ext",
	)
	return ret, nil
}

func analyticsErrorTag(name, msg string) hookanalytics.Analytics {
	return hookanalytics.Analytics{
		Activities: []hookanalytics.Activity{{
			Name:   name,
			Status: hookanalytics.ActivityStatusError,
			Results: []hookanalytics.Result{{
				Status: hookanalytics.ResultStatusError,
				Values: map[string]interface{}{"error": msg},
			}},
		}},
	}
}
```

- [ ] **Step 4: Run — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestHandleAuctionResponseHook -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/scope3/tmp/module.go modules/scope3/tmp/module_test.go
git commit -m "Module: Scope3 TMP - HandleAuctionResponseHook with per-imp mutation"
```

---

## Task 16: Wire-shape assertions — privacy guarantees on outbound JSON

**Files:**
- Modify: `modules/scope3/tmp/module_test.go`

- [ ] **Step 1: Write the wire-shape test**

Append to `modules/scope3/tmp/module_test.go`:
```go
func TestOutboundWireShape_PrivacyGuarantees(t *testing.T) {
	var contextBody, identityBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/tmp/context":
			contextBody = buf
		case "/tmp/identity":
			identityBody = buf
		}
		_, _ = w.Write([]byte(`{"type":"x","request_id":"","offers":[],"eligible_package_ids":[]}`))
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{
		"router_url":"`+srv.URL+`",
		"seller_agent_url":"https://us",
		"masking":{"enabled":true,"user":{"preserve_eids":["liveramp.com"]}}
	}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	br := &openrtb2.BidRequest{
		ID: "a",
		Imp: []openrtb2.Imp{{ID: "i1", TagID: "h"}},
		Site: &openrtb2.Site{Domain: "x.com"},
		Device: &openrtb2.Device{IP: "1.2.3.4", IFA: "AAA-BBB", Geo: &openrtb2.Geo{Country: "USA"}},
		User: &openrtb2.User{
			ID:       "uid",
			BuyerUID: "buid",
			Ext:      json.RawMessage(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R1"}]},{"source":"criteo.com","uids":[{"id":"DROP"}]}]}`),
		},
	}

	ar := newAsyncRequest(context.Background())
	ar.module = m
	cfg := json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"p"}}}}`)
	ar.fetchAsync(br, cfg, nil)
	<-ar.done

	require.NotEmpty(t, contextBody)
	require.NotEmpty(t, identityBody)

	require.NotContains(t, string(contextBody), `"ip":"1.2.3.4"`)
	require.NotContains(t, string(contextBody), `"ifa":"AAA-BBB"`)
	require.NotContains(t, string(contextBody), `"id":"uid"`)
	require.NotContains(t, string(identityBody), `"ip":"1.2.3.4"`)
	require.NotContains(t, string(identityBody), `"package_ids"`)
	require.NotContains(t, string(identityBody), `"criteo.com"`)
	require.Contains(t, string(identityBody), `"country":"US"`)

	ctxID := gjson.GetBytes(contextBody, "request_id").String()
	idID := gjson.GetBytes(identityBody, "request_id").String()
	require.NotEqual(t, ctxID, idID, "context and identity request_ids MUST NOT correlate")
}
```

Add the `io` import.

- [ ] **Step 2: Run — should pass (all the privacy logic is already in place from earlier tasks)**

Run: `go test ./modules/scope3/tmp/... -run TestOutboundWireShape -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add modules/scope3/tmp/module_test.go
git commit -m "Module: Scope3 TMP - wire-shape assertions for privacy and correlation"
```

---

## Task 17: Test fixtures and additional integration scenarios

**Files:**
- Create: `modules/scope3/tmp/testdata/*.json`
- Modify: `modules/scope3/tmp/module_test.go`

- [ ] **Step 1: Create JSON fixtures**

Create `modules/scope3/tmp/testdata/bid_request_multi_imp_three_placements.json`:
```json
{
  "id": "auction-three",
  "imp": [
    {"id": "imp_header", "tagid": "header", "banner": {"format": [{"w": 728, "h": 90}]}},
    {"id": "imp_side",   "tagid": "sidebar", "banner": {"format": [{"w": 300, "h": 250}]}},
    {"id": "imp_video",  "tagid": "video",   "video":  {"mimes": ["video/mp4"]}}
  ],
  "site": {"domain": "example.com", "page": "https://example.com/news/x"},
  "device": {"ua": "ua", "geo": {"country": "USA"}},
  "user": {"ext": {"eids": [{"source": "liveramp.com", "uids": [{"id": "RID-123"}]}]}}
}
```

Create `modules/scope3/tmp/testdata/account_config_three_placements.json`:
```json
{
  "scope3": {
    "tmp": {
      "property_rid": "01916f3a-9c4e-7000-8000-000000000010",
      "property_type": "website",
      "placements": {
        "header":  "header_728x90",
        "sidebar": "sidebar_300x250",
        "video":   "preroll_video"
      }
    }
  }
}
```

Create `modules/scope3/tmp/testdata/context_response_empty.json`:
```json
{"type": "context_match_response", "request_id": "x", "offers": []}
```

Create `modules/scope3/tmp/testdata/identity_response_with_tmpx_only.json`:
```json
{"type": "identity_match_response", "request_id": "y", "eligible_package_ids": [], "tmpx": "k1.tokenABC", "ttl_sec": 60}
```

- [ ] **Step 2: Write a fixture-driven test for the TMPX-only success case**

Append to `modules/scope3/tmp/module_test.go`:
```go
import "os"

func TestEndToEnd_SuccessTMPXOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tmp/context":
			data, _ := os.ReadFile("testdata/context_response_empty.json")
			_, _ = w.Write(data)
		case "/tmp/identity":
			data, _ := os.ReadFile("testdata/identity_response_with_tmpx_only.json")
			_, _ = w.Write(data)
		}
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)

	brData, _ := os.ReadFile("testdata/bid_request_multi_imp_three_placements.json")
	var br openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(brData, &br))

	accountCfg, _ := os.ReadFile("testdata/account_config_three_placements.json")

	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(&br, accountCfg, nil)
	<-ar.done

	require.NoError(t, ar.err)
	require.NotNil(t, ar.result)
	require.Equal(t, "k1.tokenABC", ar.result.TMPX, "TMPX emitted even when intersection is empty")
	for _, pr := range ar.result.PerPlacement {
		require.Empty(t, pr.EligiblePackages, "intersection is empty")
	}
}
```

- [ ] **Step 3: Run — should pass**

Run: `go test ./modules/scope3/tmp/... -run TestEndToEnd_SuccessTMPXOnly -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add modules/scope3/tmp/testdata modules/scope3/tmp/module_test.go
git commit -m "Module: Scope3 TMP - test fixtures and TMPX-only success scenario"
```

---

## Task 18: Coverage verification

**Files:** none modified

- [ ] **Step 1: Run coverage**

Run:
```bash
go test -coverprofile=/tmp/scope3_tmp_cover.out ./modules/scope3/tmp/...
go tool cover -func=/tmp/scope3_tmp_cover.out | tail -1
```

Expected: total coverage **≥ 90%**.

- [ ] **Step 2: Identify any uncovered lines**

Run:
```bash
go tool cover -func=/tmp/scope3_tmp_cover.out | sort -k3 -n | head -20
```

Expected: only trivial functions (e.g., variant constants, getters) below the gate.

- [ ] **Step 3: If below 90%, add targeted tests**

For each uncovered branch in `intersect`, `accountResolver`, or per-imp mapping, add a unit test that triggers it. Re-run Step 1 until coverage ≥ 90% line and the three critical functions hit 100% branch.

To check branch coverage on a specific function:
```bash
go test -covermode=atomic -coverprofile=/tmp/scope3_tmp_cover.out ./modules/scope3/tmp/...
go tool cover -html=/tmp/scope3_tmp_cover.out -o /tmp/scope3_tmp_cover.html
open /tmp/scope3_tmp_cover.html  # macOS — visually inspect intersect, accountResolver, fetchAsync
```

- [ ] **Step 4: Commit any added tests**

```bash
# only if step 3 added tests
git add modules/scope3/tmp/*_test.go
git commit -m "Module: Scope3 TMP - additional tests to meet 90% coverage gate"
```

---

## Task 19: README and migration notes

**Files:**
- Create: `modules/scope3/tmp/README.md`
- Modify: `modules/scope3/rtd/README.md`

- [ ] **Step 1: Write the new module's README**

Create `modules/scope3/tmp/README.md`:
```markdown
# Scope3 TMP Module

This module integrates the AdCP **Trusted Match Protocol** (TMP) for cross-publisher
frequency capping with publisher-side privacy join. It calls a Scope3-operated TMP
router (e.g., `https://tmp.interchange.io`) and enriches the bid response with
eligible package IDs, the TMPX exposure token, and buyer-defined targeting
key-value pairs.

## Maintainer

- Email: bokelley@scope3.com
- Company: Scope3

## Spec references

- TMP overview: https://docs.adcontextprotocol.org/docs/trusted-match
- TMP specification: https://docs.adcontextprotocol.org/docs/trusted-match/specification
- Universal Macros (`{TMPX}`): https://docs.adcontextprotocol.org/docs/creative/universal-macros

## Configuration

### Module config (`pbs.yaml`)

```yaml
hooks:
  enabled: true
  modules:
    scope3:
      tmp:
        enabled: true
        router_url: https://tmp.interchange.io
        seller_agent_url: https://prebid.example.com/scope3
        auth_key: ${SCOPE3_TMP_AUTH_KEY}
        timeout_ms: 200
        cache_ttl_seconds: 60
        cache_size: 10485760
        add_to_targeting: false
        masking:
          enabled: true
          geo:
            preserve_metro: true
            preserve_zip: true
            preserve_city: false
            lat_long_precision: 2
          user:
            preserve_eids:
              - liveramp.com
              - uidapi.com
              - id5-sync.com
          device:
            preserve_mobile_ids: false

  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          entrypoint:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "scope3.tmp"
                    hook_impl_code: "HandleEntrypointHook"
          processed_auction_request:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "scope3.tmp"
                    hook_impl_code: "HandleProcessedAuctionHook"
          auction_response:
            groups:
              - timeout: 250
                hook_sequence:
                  - module_code: "scope3.tmp"
                    hook_impl_code: "HandleAuctionResponseHook"
```

### Account config (per-publisher stored config)

```json
{
  "scope3": {
    "tmp": {
      "property_rid": "01916f3a-9c4e-7000-8000-000000000010",
      "property_type": "website",
      "placements": {
        "div-gpt-ad-header":  "header_728x90",
        "div-gpt-ad-sidebar": "sidebar_300x250",
        "div-gpt-ad-video":   "preroll_video"
      }
    }
  }
}
```

`property_rid` and `property_type` are required per-account. Without them the
module silently skips the auction (no enrichment). `seller_agent_url` defaults
to the module-level config; account config can override.

### Per-request override (testing only)

```json
{
  "ext": {
    "prebid": {
      "modules": {
        "scope3": {
          "tmp": {
            "property_rid": "01916f3a-...",
            "placement_id": "test_slot"
          }
        }
      }
    }
  }
}
```

## Output shape

```json
{
  "ext": {
    "scope3": { "tmp": { "tmpx": "k1.dG1weC1leGFtcGxl..." } }
  },
  "seatbid": [{
    "bid": [{
      "impid": "imp-header",
      "ext": {
        "scope3": {
          "tmp": {
            "placement_id": "header_728x90",
            "eligible_packages": ["pkg_abc"],
            "segments": ["news_intender"]
          }
        },
        "prebid": {
          "targeting": {
            "TMPX": "k1.dG1weC1leGFtcGxl...",
            "buyer_kv_key": "buyer_kv_value"
          }
        }
      }
    }]
  }]
}
```

`ext.scope3.tmp.tmpx` is auction-level (user-scoped, identical across all bids).
Per-bid `ext.scope3.tmp` carries the placement-specific enrichment.

When `add_to_targeting: true`, `TMPX` is set as a `prebid.targeting` key (matching
the AdCP Universal Macros convention) and `signals.targeting_kvs` from the buyer
flow verbatim into the same `prebid.targeting` namespace.

## Privacy

Field masking follows the same model as `modules/scope3/rtd/`. The TMP wire
guarantees an extra structural separation:

- Context Match contains page context but no user identity.
- Identity Match contains opaque user tokens but no page context.
- `package_ids` is omitted from Identity Match per spec (correlation prevention).
- Identity tokens are capped at 3 per the TMP spec hard limit (`maxItems: 3`).
- `request_id`s on the two calls are independent UUIDs.

## Multi-imp behavior

Each imp's placement is resolved from `account.scope3.tmp.placements[imp.tagid]`.
Unique placements deduplicate to one Context Match each. Identity Match is one
call per auction (page-context-free). Per-imp results are scoped onto each
`seatbid[].bid[]` via `bid.impid` → `imp.tagid` → `placement_id` lookup.

## Coexistence with scope3.rtd

If both `scope3.rtd` and `scope3.tmp` are enabled, the TMP module overwrites
the RTD module's writes to `ext.scope3.*`. Operators should configure the
execution plan with TMP listed last in the `auction_response` stage; the
overwrite is also defensive (independent of plan order).
```

- [ ] **Step 2: Add the deprecation pointer to the RTD README**

Edit `modules/scope3/rtd/README.md`. After the title heading (line 1), add a deprecation notice block:
```markdown
> **Deprecation notice**: For new deployments, prefer `modules/scope3/tmp` —
> the AdCP Trusted Match Protocol module — which offers stronger privacy
> guarantees through structurally-separated context and identity matching.
> See [`../tmp/README.md`](../tmp/README.md). This module remains supported
> through at least one release cycle for existing integrations.
```

- [ ] **Step 3: Commit**

```bash
git add modules/scope3/tmp/README.md modules/scope3/rtd/README.md
git commit -m "Module: Scope3 TMP - README and RTD deprecation pointer"
```

---

## Task 20: Final integration check

**Files:** none modified

- [ ] **Step 1: Full module test run**

Run:
```bash
go test ./modules/scope3/...
```
Expected: all tests in both `scope3/rtd` and `scope3/tmp` pass.

- [ ] **Step 2: Module registration smoke test**

Run:
```bash
go test ./modules/... -run TestRegistration
```
Expected: PASS — confirms `scope3.tmp` Builder is reachable through the central registry.

- [ ] **Step 3: Full project compile**

Run:
```bash
go build ./...
```
Expected: compiles cleanly. No errors. No warnings.

- [ ] **Step 4: Vet pass**

Run:
```bash
go vet ./modules/scope3/tmp/...
```
Expected: no findings.

- [ ] **Step 5: Coverage final check**

Run:
```bash
go test -coverprofile=/tmp/scope3_tmp_cover.out ./modules/scope3/tmp/...
go tool cover -func=/tmp/scope3_tmp_cover.out | tail -1
```
Expected: total coverage ≥ 90%.

- [ ] **Step 6: No commit needed**

This task is verification only. If anything fails, drop back to the relevant task and fix.

---

## Self-Review

**Spec coverage:**

| Spec section | Plan task(s) |
|---|---|
| Module placement and naming | Task 1 |
| Three-stage hook lifecycle | Tasks 13, 14, 15 |
| `Config` and Builder validation | Task 3 |
| Identifier resolution (property_rid/placement_id/property_type/seller_agent_url/router_url) | Task 4 |
| `accountResolver` per-source precedence | Task 4 |
| Masking + identity cap + country conversion | Tasks 5, 6, 9 |
| Cache key composition | Task 7 |
| `intersect` pure function | Task 8 |
| AsyncRequest scaffolding | Task 10 |
| HTTP helpers (Context, Identity) | Task 11 |
| Asymmetric N+1 fan-out | Task 12 |
| Vendored proto types | Task 2 |
| Output shape (per-imp / per-bid) | Task 15 |
| Privacy wire-shape assertions | Task 16 |
| Fixtures + TMPX-only success | Task 17 |
| Coverage gate ≥ 90% | Tasks 18, 20 |
| README + RTD deprecation pointer | Task 19 |
| Final integration check | Task 20 |

No spec sections without a task.

**Placeholder scan:** no TBDs, no "TODO", no "implement later", no "fill in details". Every code step has the actual code.

**Type consistency:** spot-checked across tasks:
- `Module`, `Config`, `AsyncRequest`, `AsyncResult`, `PlacementResult` consistent across tasks 3, 10, 12, 15.
- `accountResolver`, `AuctionIdentifiers` consistent across tasks 4, 12.
- `intersect(contextOffers, identityEligible) []string` signature consistent across tasks 8, 12.
- `extractIdentities(user, preserveEids) []IdentityToken` consistent across tasks 6, 12.
- `moduleContextAsyncKey` constant defined in Task 13, referenced in Tasks 14, 15.
- `fetchContext` / `fetchIdentity` signature consistent across tasks 11, 12.
- Wire types (`ContextMatchRequest`, `IdentityMatchResponse`, etc.) defined in Task 2, used across Tasks 11, 12, 15, 16, 17.

**Sharp edge from spec self-review** about per-request `placement_id` override being a single value — captured in the resolver implementation (Task 4) and documented in the README (Task 19).
