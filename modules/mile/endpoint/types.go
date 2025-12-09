package endpoint

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// ErrSiteNotFound is returned when no site configuration is found for a given site ID.
var ErrSiteNotFound = errors.New("site not found")

// MileRequest is the incoming payload from MilePrebidAdapter.
type MileRequest struct {
	SiteID      string          `json:"siteId"`
	PublisherID string          `json:"publisherId"`
	PlacementID string          `json:"placementId"`
	CustomData  []CustomData    `json:"customData,omitempty"`
	Raw         json.RawMessage `json:"-"`
}

// CustomData captures optional passthrough targeting/settings blocks.
type CustomData struct {
	Settings  map[string]any `json:"settings,omitempty"`
	Targeting map[string]any `json:"targeting,omitempty"`
}

// SiteConfig represents the Redis-stored schema for a Mile site.
type SiteConfig struct {
	SiteID       string                     `json:"siteId"`
	PublisherID  string                     `json:"publisherId"`
	Bidders      []string                   `json:"bidders,omitempty"`
	Placements   map[string]PlacementConfig `json:"placements"`
	SiteMetadata map[string]any             `json:"siteConfig,omitempty"`
	Ext          map[string]json.RawMessage `json:"ext,omitempty"`
}

// PlacementConfig binds a placement to bidders and bidder params.
type PlacementConfig struct {
	PlacementID   string                     `json:"placementId"`
	AdUnit        string                     `json:"ad_unit,omitempty"`
	Sizes         [][]int                    `json:"sizes,omitempty"`
	Floor         float64                    `json:"floor,omitempty"`
	Bidders       []string                   `json:"bidders,omitempty"`
	BidderParams  map[string]json.RawMessage `json:"bidder_params,omitempty"`
	Ext           map[string]json.RawMessage `json:"ext,omitempty"`
	MediaTypes    map[string]json.RawMessage `json:"media_types,omitempty"`
	Passthrough   map[string]json.RawMessage `json:"passthrough,omitempty"`
	StoredRequest string                     `json:"stored_request,omitempty"`
}

// SiteStore fetches per-site configuration (backed by Redis in production).
type SiteStore interface {
	Get(ctx context.Context, siteID string) (*SiteConfig, error)
	Close() error
}

// Hooks exposes optional lifecycle hooks.
type Hooks struct {
	Validate    func(ctx context.Context, req MileRequest) error
	Before      func(ctx context.Context, req MileRequest, site *SiteConfig, ortb *openrtb2.BidRequest) (*openrtb2.BidRequest, error)
	After       func(ctx context.Context, req MileRequest, site *SiteConfig, auctionStatus int, auctionBody []byte) ([]byte, int, error)
	OnException func(ctx context.Context, req MileRequest, err error)
}

// applyValidate executes the validation hook when present.
func (h Hooks) applyValidate(ctx context.Context, req MileRequest) error {
	if h.Validate == nil {
		return nil
	}
	return h.Validate(ctx, req)
}

// applyBefore executes the before hook when present.
func (h Hooks) applyBefore(ctx context.Context, req MileRequest, site *SiteConfig, in *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
	if h.Before == nil {
		return in, nil
	}
	return h.Before(ctx, req, site, in)
}

// applyAfter executes the after hook when present.
func (h Hooks) applyAfter(ctx context.Context, req MileRequest, site *SiteConfig, status int, body []byte) ([]byte, int, error) {
	if h.After == nil {
		return body, status, nil
	}
	return h.After(ctx, req, site, status, body)
}
