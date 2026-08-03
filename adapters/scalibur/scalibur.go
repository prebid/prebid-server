package scalibur

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/util/urlutil"
)

// defaultHost is used to resolve the {{.Host}} macro when the caller does not
// supply a host param, preserving the standard Scalibur endpoint.
const defaultHost = "srv.scalibur.io"

type adapter struct {
	endpoint *template.Template
}

// hostGroup collects the impressions that resolve to a single endpoint host.
// Impressions may each carry their own host param, so one incoming bid request
// can fan out into one outgoing request per distinct host.
type hostGroup struct {
	host string
	imps []openrtb2.Imp
}

// Builder builds a new instance of the Scalibur adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	temp, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	return &adapter{
		endpoint: temp,
	}, nil
}

// MakeRequests creates the HTTP requests which should be made to fetch bids from Scalibur.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	// Impressions are grouped by the endpoint host they resolve to, in
	// first-seen order, so every host a publisher asked for is honored instead
	// of only the first one.
	var groups []*hostGroup
	groupByHost := make(map[string]*hostGroup)

	// Process each impression
	for _, imp := range request.Imp {
		scaliburExt, err := parseScaliburExt(imp.Ext)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		host, err := resolveEndpointHost(imp.ID, scaliburExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		impCopy := imp

		// Resolve the placement as the ORTB imp.tagid. An ad-unit-level tagid
		// (already on the imp) takes precedence; otherwise the placementId bidder
		// param is applied. An imp that resolves to no tagid is not served.
		if impCopy.TagID == "" {
			impCopy.TagID = scaliburExt.PlacementID
		}
		if impCopy.TagID == "" {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("imp %s: missing placement; set imp.tagid or the placementId param", imp.ID),
			})
			continue
		}

		// Apply bid floor and currency
		if scaliburExt.BidFloor != nil && *scaliburExt.BidFloor > 0 {
			impCopy.BidFloor = *scaliburExt.BidFloor
			if scaliburExt.BidFloorCur != "" {
				impCopy.BidFloorCur = scaliburExt.BidFloorCur
			}
		}

		if impCopy.BidFloor > 0 && impCopy.BidFloorCur != "" && impCopy.BidFloorCur != "USD" {
			convertedValue, err := reqInfo.ConvertCurrency(impCopy.BidFloor, impCopy.BidFloorCur, "USD")
			if err != nil {
				errs = append(errs, err)
				continue
			}
			impCopy.BidFloor = convertedValue
			impCopy.BidFloorCur = "USD"
		}

		if impCopy.BidFloorCur == "" {
			impCopy.BidFloorCur = "USD"
		}

		// Prepare imp.ext: pass through every field the publisher sent under
		// ext.bidder, then overlay the server-computed values. The two adapter
		// config params are dropped to keep the request ORTB-standard:
		// placementId because the placement is carried as the ORTB imp.tagid
		// (above), and host because it has already been consumed to build the
		// endpoint this request is being sent to.
		impExtData := make(map[string]interface{})

		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err == nil {
			var passthrough map[string]json.RawMessage
			if err := jsonutil.Unmarshal(bidderExt.Bidder, &passthrough); err == nil {
				for k, v := range passthrough {
					impExtData[k] = json.RawMessage(v)
				}
			}
		}
		delete(impExtData, "placementId")
		delete(impExtData, "host")

		// Server-computed floor fields always win over any passed-through value.
		// With no floor there is no floor currency either, so neither is sent.
		if impCopy.BidFloor > 0 {
			impExtData["bidfloor"] = impCopy.BidFloor
			impExtData["bidfloorcur"] = impCopy.BidFloorCur
		} else {
			delete(impExtData, "bidfloor")
			delete(impExtData, "bidfloorcur")
		}

		// Preserve GPID if present (lives outside ext.bidder)
		var rawExt map[string]json.RawMessage
		if err := jsonutil.Unmarshal(imp.Ext, &rawExt); err == nil {
			if gpid, ok := rawExt["gpid"]; ok {
				impExtData["gpid"] = json.RawMessage(gpid)
			}
		}

		impExt, err := jsonutil.Marshal(impExtData)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		impCopy.Ext = impExt

		// Apply video defaults (matching JS defaults)
		if impCopy.Video != nil {
			videoCopy := *impCopy.Video

			// Note: In openrtb v20, field names are capitalized (MIMEs not Mimes)
			if len(videoCopy.MIMEs) == 0 {
				videoCopy.MIMEs = []string{"video/mp4"}
			}
			if videoCopy.MinDuration == 0 {
				videoCopy.MinDuration = 1
			}
			if videoCopy.MaxDuration == 0 {
				videoCopy.MaxDuration = 180
			}
			if videoCopy.MaxBitRate == 0 {
				videoCopy.MaxBitRate = 30000
			}
			if len(videoCopy.Protocols) == 0 {
				// Use adcom1.MediaCreativeSubtype for protocols in v20
				videoCopy.Protocols = []adcom1.MediaCreativeSubtype{2, 3, 5, 6}
			}
			// Note: In openrtb v20, W and H are pointers
			if videoCopy.W == nil || *videoCopy.W == 0 {
				w := int64(640)
				videoCopy.W = &w
			}
			if videoCopy.H == nil || *videoCopy.H == 0 {
				h := int64(480)
				videoCopy.H = &h
			}
			if videoCopy.Placement == 0 {
				videoCopy.Placement = 1
			}
			if videoCopy.Linearity == 0 {
				videoCopy.Linearity = 1
			}

			impCopy.Video = &videoCopy
		}

		group, ok := groupByHost[host]
		if !ok {
			group = &hostGroup{host: host}
			groupByHost[host] = group
			groups = append(groups, group)
		}
		group.imps = append(group.imps, impCopy)
	}

	// If no valid impressions, return errors
	if len(groups) == 0 {
		return nil, errs
	}

	isDebug := request.Test == 1
	if !isDebug && len(request.Ext) > 0 {
		var extRequest openrtb_ext.ExtRequest
		if err := jsonutil.Unmarshal(request.Ext, &extRequest); err == nil {
			isDebug = extRequest.Prebid.Debug
		}
	}

	var reqExtJSON json.RawMessage
	if isDebug {
		reqExt := openrtb_ext.ExtRequestScalibur{IsDebug: 1}
		if marshalled, err := jsonutil.Marshal(reqExt); err == nil {
			reqExtJSON = marshalled
		}
	}

	// One outgoing request per distinct host, each carrying only the
	// impressions that asked for that host.
	requests := make([]*adapters.RequestData, 0, len(groups))
	for _, group := range groups {
		requestCopy := *request
		requestCopy.Imp = group.imps
		requestCopy.Cur = nil
		requestCopy.Ext = reqExtJSON

		reqJSON, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri, err := a.buildEndpointURL(group.host)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		})
	}

	return requests, errs
}

// MakeBids unpacks the server's response into bids.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	// Parse the external request to get impression details
	var bidReq openrtb2.BidRequest
	if err := jsonutil.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	// Build impression map for lookup
	impMap := make(map[string]*openrtb2.Imp, len(bidReq.Imp))
	for i := range bidReq.Imp {
		impMap[bidReq.Imp[i].ID] = &bidReq.Imp[i]
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidReq.Imp))

	// Set currency
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	} else {
		bidResponse.Currency = "USD"
	}

	// Process each seat bid
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			// Find the corresponding imp
			imp, found := impMap[bid.ImpID]
			if !found {
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Invalid bid imp ID %s", bid.ImpID),
				}}
			}

			// Determine bid type based on imp
			bidType, err := getBidMediaType(bid, imp)
			if err != nil {
				return nil, []error{&errortypes.BadServerResponse{
					Message: err.Error(),
				}}
			}

			bidCopy := bid

			// Handle video VAST
			if bidType == openrtb_ext.BidTypeVideo {
				// Try to extract custom fields vastXml and vastUrl from bid.ext
				var bidExtData struct {
					VastXML string `json:"vastXml"`
					VastURL string `json:"vastUrl"`
				}
				if bid.Ext != nil {
					if err := jsonutil.Unmarshal(bid.Ext, &bidExtData); err == nil {
						if bidExtData.VastXML != "" {
							bidCopy.AdM = bidExtData.VastXML
						} else if bidExtData.VastURL != "" && bidCopy.AdM == "" {
							// Wrap VAST URL in VAST wrapper
							bidCopy.AdM = fmt.Sprintf(`<VAST version="3.0"><Ad><Wrapper><VASTAdTagURI><![CDATA[%s]]></VASTAdTagURI></Wrapper></Ad></VAST>`, bidExtData.VastURL)
						}
					}
				}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bidCopy,
				BidType: bidType,
			})
		}
	}

	if len(bidResponse.Bids) == 0 {
		return nil, nil
	}

	return bidResponse, nil
}

// parseScaliburExt extracts Scalibur-specific parameters from the impression extension.
func parseScaliburExt(impExt json.RawMessage) (*openrtb_ext.ExtImpScalibur, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(impExt, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to parse imp.ext: %s", err.Error()),
		}
	}

	var scaliburExt openrtb_ext.ExtImpScalibur
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &scaliburExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to parse Scalibur params: %s", err.Error()),
		}
	}

	return &scaliburExt, nil
}

// resolveEndpointHost determines the endpoint host for a single impression. The
// host param is caller-supplied, so it is SSRF-validated; when omitted it falls
// back to the standard Scalibur host, meaning an imp with no host param yields
// the default endpoint.
func resolveEndpointHost(impID string, ext *openrtb_ext.ExtImpScalibur) (string, error) {
	host := defaultHost
	if ext.Host != "" {
		host = ext.Host
	}

	if !urlutil.IsSafeHost(host) {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid host %s", impID, host),
		}
	}

	return host, nil
}

// buildEndpointURL resolves the operator-controlled endpoint template with the
// host resolved by resolveEndpointHost.
func (a *adapter) buildEndpointURL(host string) (string, error) {
	return macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{Host: host})
}

// getBidMediaType determines the media type based on the impression
func getBidMediaType(bid openrtb2.Bid, imp *openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	}

	// Fallback for bidders not supporting mtype (non-multi-format requests)
	if imp.Banner != nil && imp.Video == nil {
		return openrtb_ext.BidTypeBanner, nil
	}
	if imp.Video != nil && imp.Banner == nil {
		return openrtb_ext.BidTypeVideo, nil
	}

	return "", fmt.Errorf("unsupported or ambiguous media type for bid id=%s", bid.ID)
}
