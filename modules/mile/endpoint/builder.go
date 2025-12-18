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

	if len(placement.Sizes) == 0 && placement.StoredRequest == "" && (req.BaseORTB == nil || findImp(req.BaseORTB, placementID) == nil) {
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
		Imp: []openrtb2.Imp{
			{
				ID:    placementID,
				TagID: placement.AdUnit,
			},
		},
	}

	if req.BaseORTB != nil && req.BaseORTB.App != nil {
		ortb.App = req.BaseORTB.App
		// If publisher ID is missing in App, use the one from Redis or request
		if ortb.App.Publisher == nil {
			ortb.App.Publisher = &openrtb2.Publisher{ID: pubID}
		} else if ortb.App.Publisher.ID == "" {
			ortb.App.Publisher.ID = pubID
		}
	} else {
		ortb.Site = &openrtb2.Site{
			ID:        site.SiteID,
			Name:      readString(site.SiteMetadata, "name"),
			Page:      readString(site.SiteMetadata, "page"),
			Publisher: &openrtb2.Publisher{ID: pubID},
		}
	}

	if req.BaseORTB != nil {
		ortb.ID = req.BaseORTB.ID
		ortb.Device = req.BaseORTB.Device
		ortb.User = req.BaseORTB.User
		ortb.TMax = req.BaseORTB.TMax
		ortb.Cur = req.BaseORTB.Cur
		ortb.Source = req.BaseORTB.Source
		ortb.Regs = req.BaseORTB.Regs
		ortb.App = req.BaseORTB.App
		ortb.AT = req.BaseORTB.AT
		ortb.WSeat = req.BaseORTB.WSeat
		ortb.BSeat = req.BaseORTB.BSeat
		ortb.WLang = req.BaseORTB.WLang
		ortb.BAdv = req.BaseORTB.BAdv
		ortb.BCat = req.BaseORTB.BCat
		ortb.BApp = req.BaseORTB.BApp
		ortb.Test = req.BaseORTB.Test

		if req.BaseORTB.Site != nil {
			if ortb.Site.Page == "" {
				ortb.Site.Page = req.BaseORTB.Site.Page
			}
			ortb.Site.Domain = req.BaseORTB.Site.Domain
			ortb.Site.Ref = req.BaseORTB.Site.Ref
			ortb.Site.Search = req.BaseORTB.Site.Search
			ortb.Site.Mobile = req.BaseORTB.Site.Mobile
			ortb.Site.Keywords = req.BaseORTB.Site.Keywords
			ortb.Site.Cat = req.BaseORTB.Site.Cat
			ortb.Site.SectionCat = req.BaseORTB.Site.SectionCat
			ortb.Site.PageCat = req.BaseORTB.Site.PageCat
			ortb.Site.PrivacyPolicy = req.BaseORTB.Site.PrivacyPolicy
			ortb.Site.Ext = req.BaseORTB.Site.Ext
		}
		if baseImp := findImp(req.BaseORTB, placementID); baseImp != nil {
			ortb.Imp[0].Secure = baseImp.Secure
			ortb.Imp[0].Video = baseImp.Video
			ortb.Imp[0].Native = baseImp.Native
			ortb.Imp[0].Audio = baseImp.Audio
			ortb.Imp[0].Instl = baseImp.Instl
			ortb.Imp[0].DisplayManager = baseImp.DisplayManager
			ortb.Imp[0].DisplayManagerVer = baseImp.DisplayManagerVer
			ortb.Imp[0].ClickBrowser = baseImp.ClickBrowser
			ortb.Imp[0].Exp = baseImp.Exp
			ortb.Imp[0].PMP = baseImp.PMP
			if baseImp.Banner != nil && len(placement.Sizes) == 0 {
				ortb.Imp[0].Banner = baseImp.Banner
			}
		}
	}

	var aliases map[string]string
	if placement.StoredRequest != "" {
		ortb.Imp[0].Ext, aliases = buildImpExt(placement, req.CustomData, placement.StoredRequest)
	} else {
		ortb.Imp[0].Ext, aliases = buildImpExt(placement, req.CustomData, "")
		if ortb.Imp[0].Banner == nil {
			ortb.Imp[0].Banner = buildBanner(placement.Sizes)
		}
		ortb.Imp[0].BidFloor = placement.Floor
	}

	reqExt := buildRequestExt(site.Ext, aliases)
	if len(reqExt) > 0 {
		if len(ortb.Ext) > 0 {
			// Merge ext if already exists from BaseORTB
			var baseExt map[string]json.RawMessage
			_ = json.Unmarshal(ortb.Ext, &baseExt)
			var newExt map[string]json.RawMessage
			_ = json.Unmarshal(reqExt, &newExt)
			for k, v := range newExt {
				baseExt[k] = v
			}
			ortb.Ext, _ = json.Marshal(baseExt)
		} else {
			ortb.Ext = reqExt
		}
	}

	return ortb, nil
}

func findImp(ortb *openrtb2.BidRequest, placementID string) *openrtb2.Imp {
	if ortb == nil {
		return nil
	}
	for i := range ortb.Imp {
		var pID string
		if len(ortb.Imp[i].Ext) > 0 {
			var ext struct {
				PlacementID string `json:"placementId"`
			}
			if err := json.Unmarshal(ortb.Imp[i].Ext, &ext); err == nil {
				pID = ext.PlacementID
			}
		}
		if pID == "" {
			pID = ortb.Imp[i].TagID
		}
		if pID == placementID {
			return &ortb.Imp[i]
		}
	}
	return nil
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
