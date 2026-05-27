package tmp

import "encoding/json"

// Types in this file are copied from
//   github.com/adcontextprotocol/adcp-go/tmproto/types_gen.go
// at upstream commit 266bb8349622ed2e4c965af4161dfcfd05ac0f0e.
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
