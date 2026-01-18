package scalibur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

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

	// Process each impression
	for _, imp := range request.Imp {
		scaliburExt, err := parseScaliburExt(imp.Ext)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		impCopy := imp

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

		// Prepare imp.ext with placementId and params
		impExtData := make(map[string]interface{})
		impExtData["placementId"] = scaliburExt.PlacementID

		if impCopy.BidFloor > 0 {
			impExtData["bidfloor"] = impCopy.BidFloor
		}
		impExtData["bidfloorcur"] = impCopy.BidFloorCur

		// Preserve GPID if present
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

	var endpointBuffer bytes.Buffer
	if err := a.endpoint.Execute(&endpointBuffer, nil); err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     endpointBuffer.String(),
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

	if scaliburExt.PlacementID == "" {
		return nil, &errortypes.BadInput{
			Message: "placementId is required",
		}
	}

	return &scaliburExt, nil
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
