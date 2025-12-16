package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

var (
	errNoBidders = errors.New("no bidders configured for placement")
	errNoSizes   = errors.New("no sizes configured for placement")
)

var knownBidders = openrtb_ext.BuildBidderMap()

func buildOpenRTBRequest(req MileRequest, placementID string, site *SiteConfig) (*openrtb2.BidRequest, error) {
	if site == nil {
		return nil, ErrSiteNotFound
	}
	placement := site.Placement

	if len(placement.Bidders) == 0 {
		return nil, errNoBidders
	}

	if len(placement.Sizes) == 0 && placement.StoredRequest == "" {
		return nil, errNoSizes
	}

	pubID := site.PublisherID.String()
	if pubID == "" {
		pubID = req.PublisherID
	}

	// Generate request ID safely
	requestID, err := uuid.NewV4()
	if err != nil {
		requestID = uuid.Must(uuid.NewV4()) // fallback
	}

	ortb := &openrtb2.BidRequest{
		ID: requestID.String(),
		Site: &openrtb2.Site{
			ID:        site.SiteID,
			Name:      readString(site.SiteMetadata, "name"),
			Page:      readString(site.SiteMetadata, "page"),
			Publisher: &openrtb2.Publisher{ID: pubID},
		},
		Imp: []openrtb2.Imp{
			{
				ID:    placementID,
				TagID: placement.AdUnit,
			},
		},
	}

	var aliases map[string]string
	if placement.StoredRequest != "" {
		ortb.Imp[0].Ext, aliases = buildImpExt(placement, req.CustomData, placement.StoredRequest)
	} else {
		ortb.Imp[0].Ext, aliases = buildImpExt(placement, req.CustomData, "")
		ortb.Imp[0].Banner = buildBanner(placement.Sizes)
		ortb.Imp[0].BidFloor = placement.Floor
	}

	reqExt := buildRequestExt(site.Ext, aliases)
	if len(reqExt) > 0 {
		ortb.Ext = reqExt
	}

	return ortb, nil
}

func buildBanner(sizes [][]int) *openrtb2.Banner {
	if len(sizes) == 0 {
		return nil
	}
	formats := make([]openrtb2.Format, 0, len(sizes))
	for _, s := range sizes {
		if len(s) != 2 {
			continue
		}
		formats = append(formats, openrtb2.Format{W: int64(s[0]), H: int64(s[1])})
	}
	if len(formats) == 0 {
		return nil
	}
	return &openrtb2.Banner{Format: formats}
}

func buildImpExt(placement PlacementConfig, customData []CustomData, storedRequest string) (json.RawMessage, map[string]string) {
	prebid := openrtb_ext.ExtImpPrebid{
		Bidder: make(map[string]json.RawMessage, len(placement.Bidders)),
	}

	aliases := make(map[string]string)

	for _, b := range placement.Bidders {
		bidderName := b.Bidder
		baseName := bidderName
		if strings.HasSuffix(bidderName, "_server") {
			baseName = strings.TrimSuffix(bidderName, "_server")
			aliases[bidderName] = baseName
		}
		// Skip bidders that aren't known to PBS core. This prevents schema validation errors.
		if _, ok := knownBidders[baseName]; !ok {
			continue
		}
		if len(b.Params) > 0 {
			prebid.Bidder[bidderName] = normalizeBidderParams(bidderName, b.Params)
		} else {
			prebid.Bidder[bidderName] = json.RawMessage(`{}`)
		}
	}

	if len(customData) > 0 {
		if raw, err := json.Marshal(customData); err == nil {
			prebid.Passthrough = raw
		}
	}

	if storedRequest != "" {
		prebid.StoredRequest = &openrtb_ext.ExtStoredRequest{ID: storedRequest}
	}

	extMap := map[string]any{
		"prebid": prebid,
	}

	for k, v := range placement.Ext {
		extMap[k] = v
	}

	if len(placement.Passthrough) > 0 {
		extMap["passthrough"] = placement.Passthrough
	}

	if len(placement.MediaTypes) > 0 {
		for mediaType, raw := range placement.MediaTypes {
			extMap[mediaType] = raw
		}
	}

	rawExt, err := json.Marshal(extMap)
	if err != nil {
		return nil, aliases
	}
	return rawExt, aliases
}

func buildRequestExt(siteExt map[string]json.RawMessage, aliases map[string]string) json.RawMessage {
	priceGranularity := openrtb_ext.NewPriceGranularityDefault()
	prebid := &openrtb_ext.ExtRequestPrebid{
		Targeting: &openrtb_ext.ExtRequestTargeting{
			PriceGranularity:  &priceGranularity,
			IncludeBidderKeys: boolPtr(true),
			IncludeWinners:    boolPtr(true),
		},
	}
	if len(aliases) > 0 {
		prebid.Aliases = aliases
	}

	ext := map[string]any{
		"prebid": prebid,
	}

	for k, v := range siteExt {
		ext[k] = v
	}

	raw, err := json.Marshal(ext)
	if err != nil {
		return nil
	}
	return raw
}

func readString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func boolPtr(v bool) *bool {
	return &v
}

// normalizeBidderParams fixes known type mismatches to satisfy bidder schemas.
// Example: medianet requires string values for cid/crid; Redis may store them as numbers.
func normalizeBidderParams(bidder string, params json.RawMessage) json.RawMessage {
	if len(params) == 0 {
		return params
	}
	if !strings.HasPrefix(bidder, "medianet") {
		return params
	}

	var m map[string]any
	if err := json.Unmarshal(params, &m); err != nil {
		return params
	}

	coerceString := func(key string) {
		if v, ok := m[key]; ok {
			switch t := v.(type) {
			case float64:
				m[key] = fmt.Sprintf("%.0f", t)
			case json.Number:
				m[key] = t.String()
			}
		}
	}

	coerceString("cid")
	coerceString("crid")

	raw, err := json.Marshal(m)
	if err != nil {
		return params
	}
	return raw
}
