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
	var validImps []openrtb2.Imp

	// endpointExt holds the endpoint macro values taken from the first valid
	// impression; the outgoing request has a single URI for the whole request.
	var endpointExt *openrtb_ext.ExtImpScalibur

	// Process each impression
	for _, imp := range request.Imp {
		scaliburExt, err := parseScaliburExt(imp.Ext)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if endpointExt == nil {
			endpointExt = scaliburExt
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
		// ext.bidder, then overlay the server-computed values. The placement is
		// carried as the ORTB imp.tagid (above), so placementId is dropped from
		// the outgoing ext to keep the request ORTB-standard.
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

		// Server-computed floor fields always win over any passed-through value.
		impExtData["bidfloorcur"] = impCopy.BidFloorCur
		if impCopy.BidFloor > 0 {
			impExtData["bidfloor"] = impCopy.BidFloor
		} else {
			delete(impExtData, "bidfloor")
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

		validImps = append(validImps, impCopy)
	}

	// If no valid impressions, return errors
	if len(validImps) == 0 {
		return nil, errs
	}

	// Create the outgoing request
	requestCopy := *request
	requestCopy.Imp = validImps
	requestCopy.Cur = nil

	isDebug := request.Test == 1
	if !isDebug && len(request.Ext) > 0 {
		var extRequest openrtb_ext.ExtRequest
		if err := jsonutil.Unmarshal(request.Ext, &extRequest); err == nil {
			isDebug = extRequest.Prebid.Debug
		}
	}

	if isDebug {
		reqExt := openrtb_ext.ExtRequestScalibur{IsDebug: 1}
		if reqExtJSON, err := jsonutil.Marshal(reqExt); err == nil {
			requestCopy.Ext = reqExtJSON
		}
	} else {
		requestCopy.Ext = nil
	}

	reqJSON, err := jsonutil.Marshal(requestCopy)
	if err != nil {
		return nil, append(errs, err)
	}

	uri, err := a.buildEndpointURL(endpointExt)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
	}

	return []*adapters.RequestData{requestData}, errs
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

// buildEndpointURL resolves the operator-controlled endpoint template using the
// caller-supplied macro values. Host is SSRF-validated and defaults to the
// standard Scalibur host, so omitting all params yields the default endpoint.
func (a *adapter) buildEndpointURL(ext *openrtb_ext.ExtImpScalibur) (string, error) {
	host := defaultHost
	if ext != nil && ext.Host != "" {
		host = ext.Host
	}

	if !urlutil.IsSafeHost(host) {
		return "", &errortypes.BadInput{Message: "Invalid host"}
	}

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
