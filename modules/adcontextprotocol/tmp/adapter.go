package tmp

import (
	"math"
	"net/url"
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
			// Strip the query component before emitting as an artifact
			// ref: gclid, click IDs and sometimes emails ride the query
			// string, and the context path is supposed to be
			// identity-free. Fragment is dropped too — same reasoning.
			out.ArtifactRefs = append(out.ArtifactRefs, tmproto.ArtifactRef{
				Type:  tmproto.ArtifactRefTypeURL,
				Value: stripURLQueryAndFragment(req.Site.Page),
			})
		}
	}
	if req.App != nil {
		out.Bundle = req.App.Bundle
	}
	// Prefer OpenRTB auto-detect; fall back to operator default only when the
	// request carries neither Site nor App. The registry response takes final
	// priority (applied by the router) — this value is only a fallback.
	switch {
	case req.App != nil:
		out.PropertyType = tmproto.PropertyTypeMobileApp
	case req.Site != nil:
		out.PropertyType = tmproto.PropertyTypeWebsite
	case cfg.DefaultPropertyType != "":
		out.PropertyType = tmproto.PropertyType(cfg.DefaultPropertyType)
	default:
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
		out.Geo = coarseGeo(cfg, req.Device.Geo)
		out.Country = req.Device.Geo.Country
	} else if req.User != nil && req.User.Geo != nil {
		out.Geo = coarseGeo(cfg, req.User.Geo)
		out.Country = req.User.Geo.Country
	}

	if req.User != nil {
		out.Identities = extractIdentities(cfg, req.User)
	}
	out.Consent = extractConsent(req)

	return out
}

// stripURLQueryAndFragment returns the URL with the query and fragment
// components removed, keeping scheme + host + path. If the input is not
// parseable as a URL, it is returned unchanged (the wire schema
// accepts opaque strings on artifact refs).
func stripURLQueryAndFragment(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// coarseGeo emits the geo fields the TMP context payload carries. When
// masking is enabled, per-field flags gate the finer-grained categories
// (city / zip / lat-lon); with masking disabled the default is the same
// strict-mode fields (country / region / metro) that the TMP wire spec
// treats as coarse enough to not identify a user. Operators who
// explicitly opt into zip / city / lat-lon through the masking config
// take responsibility for that being acceptable at their own provider.
func coarseGeo(cfg *Config, geo *openrtb2.Geo) map[string]any {
	if geo == nil {
		return nil
	}
	m := cfg.Masking
	preserveMetro := true
	preserveZip := false
	preserveCity := false
	latLongPrecision := 0
	if m.Enabled {
		preserveMetro = m.Geo.PreserveMetro
		preserveZip = m.Geo.PreserveZip
		preserveCity = m.Geo.PreserveCity
		latLongPrecision = m.Geo.LatLongPrecision
	}

	out := map[string]any{}
	if geo.Country != "" {
		out["country"] = geo.Country
	}
	if geo.Region != "" {
		out["region"] = geo.Region
	}
	if preserveMetro && geo.Metro != "" {
		out["metro"] = geo.Metro
	}
	if preserveZip && geo.ZIP != "" {
		out["zip"] = geo.ZIP
	}
	if preserveCity && geo.City != "" {
		out["city"] = geo.City
	}
	if latLongPrecision > 0 && geo.Lat != nil && geo.Lon != nil {
		out["lat"] = truncateCoord(*geo.Lat, latLongPrecision)
		out["lon"] = truncateCoord(*geo.Lon, latLongPrecision)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// truncateCoord truncates a coordinate to n decimal places using
// math.Trunc so negative coordinates truncate toward zero (matching
// what most operators expect for a "reduce precision" knob).
func truncateCoord(v float64, precision int) float64 {
	mult := math.Pow(10, float64(precision))
	return math.Trunc(v*mult) / mult
}

// extractIdentities maps openrtb2 user.eids → tmproto.IdentityToken, honoring
// the TMP cap of three tokens. Priority order: rampid, uid2, id5, then whatever
// remains. Publishers that need a different priority should tell us — right now
// this is the most common set.
//
// When Masking is enabled and PreserveMobileIds is false, maid-typed
// tokens (mobile advertising IDs) are dropped from the identity set so
// they never reach a TMP provider.
func extractIdentities(cfg *Config, user *openrtb2.User) []tmproto.IdentityToken {
	if user == nil {
		return nil
	}
	priority := map[string]int{
		"liveramp.com": 0,
		"uidapi.com":   1,
		"id5-sync.com": 2,
		"euid.eu":      3,
		"adserver.org": 4,
		"adid.google":  5,
		"idfa.apple":   5,
	}
	dropMaid := cfg.Masking.Enabled && !cfg.Masking.Device.PreserveMobileIds
	// When PreserveEids is set (masking enabled + operator populated it,
	// or the default filled in by validated()) it is authoritative: only
	// EID sources on the allowlist survive. Publishers who want the
	// default hardcoded whitelist just leave masking off; publishers who
	// want a narrower allowlist enable masking + populate the field.
	var eidAllowlist map[string]bool
	if cfg.Masking.Enabled && len(cfg.Masking.User.PreserveEids) > 0 {
		eidAllowlist = make(map[string]bool, len(cfg.Masking.User.PreserveEids))
		for _, s := range cfg.Masking.User.PreserveEids {
			eidAllowlist[strings.ToLower(s)] = true
		}
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
		if eidAllowlist != nil && !eidAllowlist[strings.ToLower(eid.Source)] {
			continue
		}
		uidType := mapEIDToUIDType(eid.Source)
		if uidType == "" {
			continue
		}
		if dropMaid && uidType == tmproto.UIDTypeMAID {
			continue
		}
		// Every source that survives mapEIDToUIDType has a priority entry;
		// the map fallback is unreachable.
		candidates = append(candidates, scored{
			tok: tmproto.IdentityToken{
				UIDType:   uidType,
				UserToken: eid.UIDs[0].ID,
			},
			score: priority[eid.Source],
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
	case "adid.google", "idfa.apple":
		return tmproto.UIDTypeMAID
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
// Callers MUST propagate the error rather than reusing a fallback ID: TMP
// requires that context and identity request_ids never correlate, and reusing
// a deterministic id would silently break that invariant.
func newRequestID() (string, error) {
	u, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
