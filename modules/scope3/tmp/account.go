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
