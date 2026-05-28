package tmp

import (
	"encoding/json"
	"errors"

	"github.com/tidwall/gjson"
)

// AuctionIdentifiers groups the resolved identifiers shared across all imps.
// PropertyType and SellerAgentURL are intentionally absent: the TMP router
// resolves them server-side from the publisher's adagents.json.
type AuctionIdentifiers struct {
	PropertyRID string
	RouterURL   string
}

// accountResolver pulls TMP identifiers from the bid request ext and module
// config. AccountConfig is no longer used for TMP identifiers; everything comes
// from the bid request.
type accountResolver struct {
	requestExt json.RawMessage // request.Ext
	moduleCfg  Config
}

// resolveAuction returns the identifiers that are stable across all imps.
// Errors if property_rid is missing from request ext.
func (r accountResolver) resolveAuction() (AuctionIdentifiers, error) {
	ids := AuctionIdentifiers{
		RouterURL: r.moduleCfg.RouterURL,
	}

	if v := gjson.GetBytes(r.requestExt, "prebid.modules.scope3.tmp.property_rid"); v.Exists() {
		ids.PropertyRID = v.String()
	}

	if ids.PropertyRID == "" {
		return ids, errors.New("property_rid is required in request ext")
	}
	return ids, nil
}

// resolvePlacement returns the placement_id for one imp by reading only from
// imp.ext. Returns ("", false) if placement_id is absent from imp ext.
func (r accountResolver) resolvePlacement(impExt json.RawMessage) (string, bool) {
	if v := gjson.GetBytes(impExt, "prebid.modules.scope3.tmp.placement_id"); v.Exists() {
		return v.String(), true
	}
	return "", false
}
