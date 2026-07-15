package tmp

import (
	"strings"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/gofrs/uuid"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/util/iterutil"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// tmpInputs is the intermediate shape produced by the OpenRTB→TMP adapter.
// The router turns this into per-provider ContextMatchRequest / IdentityMatchRequest.
type tmpInputs struct {
	Domain       string
	Bundle       string
	PlacementID  string
	PropertyType tmproto.PropertyType
	Geo          map[string]any
	Country      string
	ArtifactRefs []tmproto.ArtifactRef
	Identities   []tmproto.IdentityToken
	Consent      map[string]any
}

// deriveInputs pulls the fields the TMP wire needs out of an OpenRTB bid
// request. Missing fields are omitted rather than defaulted — the wire schemas
// tolerate them.
func deriveInputs(cfg *Config, req *openrtb2.BidRequest) tmpInputs {
	if req == nil {
		return tmpInputs{}
	}
	out := tmpInputs{}

	if req.Site != nil {
		out.Domain = req.Site.Domain
		if req.Site.Page != "" {
			out.ArtifactRefs = append(out.ArtifactRefs, tmproto.ArtifactRef{
				Type:  tmproto.ArtifactRefTypeURL,
				Value: req.Site.Page,
			})
		}
	}
	if req.App != nil {
		out.Bundle = req.App.Bundle
	}
	if def := cfg.DefaultPropertyType; def != "" {
		out.PropertyType = tmproto.PropertyType(def)
	} else if req.App != nil {
		out.PropertyType = tmproto.PropertyTypeMobileApp
	} else {
		out.PropertyType = tmproto.PropertyTypeWebsite
	}

	// Placement — take the first imp.tagid so the wire has something stable.
	// Publishers with multiple placements per auction need one TMP request per
	// placement; that is out of scope for this initial adapter.
	for imp := range iterutil.SlicePointerValues(req.Imp) {
		if imp.TagID != "" {
			out.PlacementID = imp.TagID
			break
		}
	}

	if req.Device != nil && req.Device.Geo != nil {
		out.Geo = coarseGeo(req.Device.Geo)
		out.Country = req.Device.Geo.Country
	} else if req.User != nil && req.User.Geo != nil {
		out.Geo = coarseGeo(req.User.Geo)
		out.Country = req.User.Geo.Country
	}

	if req.User != nil {
		out.Identities = extractIdentities(req.User)
	}
	out.Consent = extractConsent(req)

	return out
}

// coarseGeo drops fields the TMP context schema forbids (postcode, lat/long,
// accuracy) even after masking. Country / region / metro are the wire allowlist.
func coarseGeo(geo *openrtb2.Geo) map[string]any {
	if geo == nil {
		return nil
	}
	out := map[string]any{}
	if geo.Country != "" {
		out["country"] = geo.Country
	}
	if geo.Region != "" {
		out["region"] = geo.Region
	}
	if geo.Metro != "" {
		out["metro"] = geo.Metro
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// extractIdentities maps openrtb2 user.eids → tmproto.IdentityToken, honoring
// the TMP cap of three tokens. Priority order: rampid, uid2, id5, then whatever
// remains. Publishers that need a different priority should tell us — right now
// this is the most common set.
func extractIdentities(user *openrtb2.User) []tmproto.IdentityToken {
	if user == nil {
		return nil
	}
	priority := map[string]int{
		"liveramp.com": 0,
		"uidapi.com":   1,
		"id5-sync.com": 2,
		"euid.eu":      3,
		"adserver.org": 4,
	}
	type scored struct {
		tok   tmproto.IdentityToken
		score int
	}
	var candidates []scored

	for eid := range iterutil.SlicePointerValues(user.EIDs) {
		if len(eid.UIDs) == 0 || eid.UIDs[0].ID == "" {
			continue
		}
		uidType := mapEIDToUIDType(eid.Source)
		if uidType == "" {
			continue
		}
		p, ok := priority[eid.Source]
		if !ok {
			p = 100
		}
		candidates = append(candidates, scored{
			tok: tmproto.IdentityToken{
				UIDType:   uidType,
				UserToken: eid.UIDs[0].ID,
			},
			score: p,
		})
	}

	// Sort ascending by score; stable so equal-source entries keep encounter order.
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].score < candidates[j-1].score; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

	const maxTokens = 3
	if len(candidates) > maxTokens {
		candidates = candidates[:maxTokens]
	}
	out := make([]tmproto.IdentityToken, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.tok)
	}
	return out
}

// mapEIDToUIDType translates the OpenRTB EID source to the TMP uid_type enum.
// Unrecognized sources return "" so the caller drops them; the wire schema
// caps at three tokens and unknown types would just consume budget.
func mapEIDToUIDType(source string) tmproto.UIDType {
	switch strings.ToLower(source) {
	case "liveramp.com":
		return tmproto.UIDTypeRampID
	case "uidapi.com":
		return tmproto.UIDTypeUID2
	case "id5-sync.com":
		return tmproto.UIDTypeID5
	case "euid.eu":
		return tmproto.UIDTypeEUID
	case "adserver.org":
		return tmproto.UIDTypePairID
	}
	return ""
}

// extractConsent surfaces the standard consent fields the identity wire
// tolerates. Buyers in regulated jurisdictions require this.
func extractConsent(req *openrtb2.BidRequest) map[string]any {
	out := map[string]any{}
	if req.Regs != nil && req.Regs.GPP != "" {
		out["gpp"] = req.Regs.GPP
	}
	if req.User != nil && len(req.User.Ext) > 0 {
		var ext struct {
			Consent string `json:"consent"`
		}
		if err := jsonutil.Unmarshal(req.User.Ext, &ext); err == nil && ext.Consent != "" {
			out["gdpr_tcf"] = ext.Consent
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// newRequestID returns a UUID v4 formatted for the TMP wire.
func newRequestID() string {
	u, err := uuid.NewV4()
	if err != nil {
		// Extremely unlikely (would need a broken RNG). Fall back to a
		// deterministic identifier — signature verification is unaffected but
		// dedup at the provider may be weaker on repeats.
		return "adcp-tmp-request"
	}
	return u.String()
}
