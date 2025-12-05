package sparteo

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

type extBidWrapper struct {
	Prebid openrtb_ext.ExtBidPrebid `json:"prebid"`
}

const unknownValue = "unknown"

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: cfg.Endpoint,
	}
	return bidder, nil
}

func parseExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpSparteo, error) {
	var bidderExt adapters.ExtImpBidder

	bidderExtErr := jsonutil.Unmarshal(imp.Ext, &bidderExt)
	if bidderExtErr != nil {
		return nil, fmt.Errorf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, bidderExtErr)
	}

	impExt := openrtb_ext.ExtImpSparteo{}
	sparteoExtErr := jsonutil.Unmarshal(bidderExt.Bidder, &impExt)
	if sparteoExtErr != nil {
		return nil, fmt.Errorf("ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, sparteoExtErr)
	}

	return &impExt, nil
}

func (a *adapter) MakeRequests(req *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	request := *req
	var errs []error

	request.Imp = make([]openrtb2.Imp, len(req.Imp))
	copy(request.Imp, req.Imp)

	var networkID string
	for i, imp := range request.Imp {
		extImpSparteo, err := parseExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if networkID == "" && extImpSparteo.NetworkId != "" {
			networkID = extImpSparteo.NetworkId
		}

		var extMap map[string]interface{}
		if err := jsonutil.Unmarshal(imp.Ext, &extMap); err != nil {
			errs = append(errs, fmt.Errorf("ignoring imp id=%s, error while unmarshaling ext, err: %s", imp.ID, err))
			continue
		}

		sparteoMap, ok := extMap["sparteo"].(map[string]interface{})
		if !ok {
			sparteoMap = make(map[string]interface{})
			extMap["sparteo"] = sparteoMap
		}
		paramsMap, ok := sparteoMap["params"].(map[string]interface{})
		if !ok {
			paramsMap = make(map[string]interface{})
			sparteoMap["params"] = paramsMap
		}
		if bidderObj, ok := extMap["bidder"].(map[string]interface{}); ok {
			delete(extMap, "bidder")
			for k, v := range bidderObj {
				paramsMap[k] = v
			}
		}
		updatedExt, err := jsonutil.Marshal(extMap)
		if err != nil {
			errs = append(errs, fmt.Errorf("ignoring imp id=%s, error while marshaling updated ext, err: %s", imp.ID, err))
			continue
		}
		request.Imp[i].Ext = updatedExt
	}

	var sb strings.Builder
	sb.WriteString("network_id=")
	sb.WriteString(url.QueryEscape(networkID))

	var pubToUpdate *openrtb2.Publisher
	var pubExtPath string

	if req.Site != nil {
		siteCopy := *req.Site
		request.Site = &siteCopy
		if req.Site.Publisher != nil {
			pubCopy := *req.Site.Publisher
			request.Site.Publisher = &pubCopy
		}

		pubToUpdate = ensurePublisher(request.Site.Publisher)
		request.Site.Publisher = pubToUpdate
		pubExtPath = "site.publisher.ext"

		domain := resolveSiteDomain(request.Site)
		if domain == "" {
			domain = unknownValue
			errs = append(errs, &errortypes.BadInput{
				Message: "Domain not found. Missing the site.domain or the site.page field",
			})
		}
		sb.WriteString("&site_domain=")
		sb.WriteString(url.QueryEscape(domain))
	} else if req.App != nil {
		appCopy := *req.App
		request.App = &appCopy
		if req.App.Publisher != nil {
			pubCopy := *req.App.Publisher
			request.App.Publisher = &pubCopy
		}

		pubToUpdate = ensurePublisher(request.App.Publisher)
		request.App.Publisher = pubToUpdate
		pubExtPath = "app.publisher.ext"

		appDomain := resolveAppDomain(request.App)
		if appDomain == "" {
			appDomain = unknownValue
		}
		sb.WriteString("&app_domain=")
		sb.WriteString(url.QueryEscape(appDomain))

		bundle := resolveBundle(request.App)
		if bundle == "" {
			bundle = unknownValue
			errs = append(errs, &errortypes.BadInput{
				Message: "Bundle not found. Missing the app.bundle field.",
			})
		}
		sb.WriteString("&bundle=")
		sb.WriteString(url.QueryEscape(bundle))
	} else {
		// NO CONTEXT (Fallback)
		request.Site = &openrtb2.Site{}
		pubToUpdate = ensurePublisher(request.Site.Publisher)
		request.Site.Publisher = pubToUpdate
		pubExtPath = "site.publisher.ext"
	}

	ext, err := updatePublisherExtension(&pubToUpdate.Ext, networkID, pubExtPath)
	if err != nil {
		errs = append(errs, err)
	} else {
		pubToUpdate.Ext = ext
	}

	uri, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, []error{fmt.Errorf("invalid endpoint URL %q: %w", a.endpoint, err)}
	}
	uri.RawQuery = sb.String()

	body, err := jsonutil.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	requestData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    uri.String(),
		Body:   body,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	return []*adapters.RequestData{requestData}, errs
}

func normalizeHostname(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	u, err := url.Parse(host)
	if err != nil || u.Hostname() == "" {
		if i := strings.Index(host, ":"); i >= 0 {
			host = host[:i]
		} else if i := strings.Index(host, "/"); i >= 0 {
			host = host[:i]
		}
	} else {
		host = u.Hostname()
	}

	host = strings.ToLower(host)
	host = strings.TrimSuffix(host, ".")
	host = strings.TrimPrefix(host, "www.")

	if host == "null" {
		return ""
	}
	return host
}

func resolveSiteDomain(site *openrtb2.Site) string {
	if site != nil {
		if d := normalizeHostname(site.Domain); d != "" {
			return d
		}
		if fromPage := normalizeHostname(site.Page); fromPage != "" {
			return fromPage
		}
	}
	return ""
}

func resolveAppDomain(app *openrtb2.App) string {
	if app != nil {
		if d := normalizeHostname(app.Domain); d != "" {
			return d
		}
	}
	return ""
}

func resolveBundle(app *openrtb2.App) string {
	if app == nil {
		return ""
	}
	raw := strings.TrimSpace(app.Bundle)
	if raw == "" || strings.EqualFold(raw, "null") {
		return ""
	}
	return raw
}

func ensurePublisher(p *openrtb2.Publisher) *openrtb2.Publisher {
	if p == nil {
		p = &openrtb2.Publisher{}
	}
	if p.Ext == nil {
		p.Ext = jsonutil.RawMessage("{}")
	}
	return p
}

func updatePublisherExtension(targetExt *jsonutil.RawMessage, networkID, fieldPath string) ([]byte, error) {
	var pubExt map[string]interface{}
	if err := jsonutil.Unmarshal(*targetExt, &pubExt); err != nil {
		pubExt = make(map[string]interface{})
	}

	params, ok := pubExt["params"].(map[string]interface{})
	if !ok {
		params = make(map[string]interface{})
		pubExt["params"] = params
	}
	params["networkId"] = networkID

	updated, err := jsonutil.Marshal(pubExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error marshaling %s: %s", fieldPath, err),
		}
	}
	return updated, nil
}

func (a *adapter) MakeBids(req *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(respData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(respData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(respData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = bidResp.Cur

	for _, seatBid := range bidResp.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := a.getMediaType(&bid)
			if err != nil {
				continue
			}

			switch bidType {
			case openrtb_ext.BidTypeBanner:
				seatBid.Bid[i].MType = openrtb2.MarkupBanner
			case openrtb_ext.BidTypeVideo:
				seatBid.Bid[i].MType = openrtb2.MarkupVideo
			case openrtb_ext.BidTypeNative:
				seatBid.Bid[i].MType = openrtb2.MarkupNative
			default:
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidderResponse, nil
}

func (a *adapter) getMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	var wrapper extBidWrapper
	if err := jsonutil.Unmarshal(bid.Ext, &wrapper); err != nil {
		return "", fmt.Errorf("error unmarshaling bid ext for bid id=%s: %v", bid.ID, err)
	}
	bidExt := wrapper.Prebid

	bidType, err := openrtb_ext.ParseBidType(string(bidExt.Type))
	if err != nil {
		return "", fmt.Errorf("error parsing bid type for bid id=%s: %v", bid.ID, err)
	}

	if bidType == openrtb_ext.BidTypeAudio {
		return "", fmt.Errorf("bid type %q is not supported for bid id=%s", bidExt.Type, bid.ID)
	}

	return bidType, nil
}
