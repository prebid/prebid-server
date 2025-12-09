package endpoint

import (
	"encoding/json"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

var (
	errMissingPlacement = errors.New("placement not found for site")
	errNoBidders        = errors.New("no bidders configured for placement")
	errNoSizes          = errors.New("no sizes configured for placement")
)

func buildOpenRTBRequest(req MileRequest, site *SiteConfig) (*openrtb2.BidRequest, error) {
	if site == nil {
		return nil, ErrSiteNotFound
	}
	placement, ok := site.Placements[req.PlacementID]
	if !ok {
		return nil, errMissingPlacement
	}

	bidders := placement.Bidders
	if len(bidders) == 0 {
		bidders = site.Bidders
	}
	if len(bidders) == 0 {
		return nil, errNoBidders
	}

	if len(placement.Sizes) == 0 && placement.StoredRequest == "" {
		return nil, errNoSizes
	}

	pubID := site.PublisherID
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
				ID:    placement.PlacementID,
				TagID: placement.AdUnit,
			},
		},
	}

	if placement.StoredRequest != "" {
		ortb.Imp[0].Ext = buildImpExt(bidders, placement, req.CustomData, placement.StoredRequest)
	} else {
		ortb.Imp[0].Ext = buildImpExt(bidders, placement, req.CustomData, "")
		ortb.Imp[0].Banner = buildBanner(placement.Sizes)
		ortb.Imp[0].BidFloor = placement.Floor
	}

	reqExt := buildRequestExt(site.Ext)
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

func buildImpExt(bidders []string, placement PlacementConfig, customData []CustomData, storedRequest string) json.RawMessage {
	prebid := openrtb_ext.ExtImpPrebid{
		Bidder: make(map[string]json.RawMessage, len(bidders)),
	}

	for _, bidder := range bidders {
		if params, ok := placement.BidderParams[bidder]; ok && len(params) > 0 {
			prebid.Bidder[bidder] = params
			continue
		}
		prebid.Bidder[bidder] = json.RawMessage(`{}`)
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
		return nil
	}
	return rawExt
}

func buildRequestExt(siteExt map[string]json.RawMessage) json.RawMessage {
	priceGranularity := openrtb_ext.NewPriceGranularityDefault()
	prebid := &openrtb_ext.ExtRequestPrebid{
		Targeting: &openrtb_ext.ExtRequestTargeting{
			PriceGranularity:  &priceGranularity,
			IncludeBidderKeys: boolPtr(true),
			IncludeWinners:    boolPtr(true),
		},
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
