package targetVideo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type TargetVideoAdapter struct {
	endpoint string
}

type impExt struct {
	TargetVideo openrtb_ext.ExtImpTargetVideo `json:"targetVideo"`
}

func (a *TargetVideoAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	totalImps := len(request.Imp)
	errors := make([]error, 0)
	adapterRequests := make([]*adapters.RequestData, 0, totalImps)

	// Split multi-imp request into multiple ad server requests. SRA is currently not recommended.
	for i := 0; i < totalImps; i++ {
		if adapterReq, err := a.makeRequest(*request, request.Imp[i]); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errors = append(errors, err)
		}
	}

	return adapterRequests, errors
}

func (a *TargetVideoAdapter) makeRequest(request openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {

	// For now, this adapter sends one imp per request, but we still
	// iterate over all imps in the request to perform the required
	// imp.ext transformation.
	request.Imp = []openrtb2.Imp{imp}

	_, errImp := validateImpAndSetExt(&imp)
	if errImp != nil {
		return nil, errImp
	}

	for i := range request.Imp {
		if len(request.Imp[i].Ext) == 0 {
			continue
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(request.Imp[i].Ext, &root); err != nil {
			// If ext cannot be parsed, skip transformation for this imp
			continue
		}

		// Try to extract placementId from ext.bidder.targetVideo (or targetvideo)
		placementId := ""
		if bRaw, ok := root["bidder"]; ok && len(bRaw) > 0 {
			var bidder map[string]json.RawMessage
			if err := json.Unmarshal(bRaw, &bidder); err == nil {
				if placementIdRaw, ok := bidder["placementId"]; ok && len(placementIdRaw) > 0 {

					var asStr string
					var asInt int64
					if err := json.Unmarshal(placementIdRaw, &asStr); err == nil && asStr != "" {
						placementId = asStr
					} else if err := json.Unmarshal(placementIdRaw, &asInt); err == nil {
						placementId = fmt.Sprintf("%d", asInt)
					}

				}
			}
			// Remove bidder node as required
			delete(root, "bidder")
		}

		// If we obtained a placementId, set ext.prebid.storedrequest.id = placementId
		if placementId != "" {
			// Build prebid.storedrequest structure, preserving existing prebid if any
			var prebid map[string]json.RawMessage
			if pr, ok := root["prebid"]; ok && len(pr) > 0 {
				_ = json.Unmarshal(pr, &prebid)
			}
			if prebid == nil {
				prebid = make(map[string]json.RawMessage)
			}
			stored := map[string]string{"id": placementId}
			storedRaw, _ := json.Marshal(stored)
			prebid["storedrequest"] = storedRaw
			prebidRaw, _ := json.Marshal(prebid)
			root["prebid"] = prebidRaw
		}

		// Marshal back the transformed ext
		if newExt, err := json.Marshal(root); err == nil {
			request.Imp[i].Ext = newExt
		}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	//fmt.Println("TARGET VIDEO reqJson: ", string(reqJSON))

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *TargetVideoAdapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if httpRes.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", httpRes.StatusCode)}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(httpRes.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("error while decoding response, err: %s", err)}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	if len(bidResp.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	br := adapters.NewBidderResponse()
	errs := []error{}

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]
			// Ensure imp exists and is video
			mediaType := openrtb_ext.BidTypeVideo
			for _, imp := range bidReq.Imp {
				if imp.ID == bid.ImpID {
					if imp.Video == nil {
						// Not a video impression; skip
						errs = append(errs, &errortypes.BadServerResponse{Message: fmt.Sprintf("ignoring bid id=%s for non-video imp id=%s", bid.ID, bid.ImpID)})
						mediaType = ""
					}
					break
				}
			}

			br.Bids = append(br.Bids, &adapters.TypedBid{Bid: &bid, BidType: mediaType})
		}
	}
	return br, errs
}

func validateImpAndSetExt(imp *openrtb2.Imp) (int, error) {
	if imp.Video == nil {
		return 0, &errortypes.BadInput{Message: fmt.Sprintf("Only video impressions are supported by targetvideo. ImpID=%s", imp.ID)}
	}
	if len(imp.Ext) == 0 {
		return 0, &errortypes.BadInput{Message: fmt.Sprintf("imp.ext is required and must contain bidder params for targetvideo. ImpID=%s", imp.ID)}
	}
	var ext impExt
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return 0, &errortypes.BadInput{Message: fmt.Sprintf("error parsing imp.ext for targetvideo, err: %s", err)}
	}

	return 0, nil
}

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {

	bidder := &TargetVideoAdapter{
		endpoint: cfg.Endpoint,
	}
	return bidder, nil
}
