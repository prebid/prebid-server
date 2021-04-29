package unruly

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type UnrulyAdapter struct {
	URI string
}

// Builder builds a new instance of the Unruly adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &UnrulyAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func (a *UnrulyAdapter) ReplaceImp(imp openrtb2.Imp, request *openrtb2.BidRequest) *openrtb2.BidRequest {
	reqCopy := *request
	reqCopy.Imp = append(make([]openrtb2.Imp, 0, 1), imp)
	return &reqCopy
}

func (a *UnrulyAdapter) BuildRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: AddHeadersToRequest(),
	}, nil
}

func AddHeadersToRequest() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Unruly-Origin", "Prebid-Server")
	return headers
}

func (a *UnrulyAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData
	for _, imp := range request.Imp {
		impWithUnrulyExt, err := convertBidderNameInExt(&imp)
		if err != nil {
			errs = append(errs, err)
		} else {
			newRequest := a.ReplaceImp(*impWithUnrulyExt, request)
			adapterReq, errors := a.BuildRequest(newRequest)
			if adapterReq != nil {
				adapterRequests = append(adapterRequests, adapterReq)
			}
			errs = append(errs, errors...)
		}
	}
	return adapterRequests, errs
}

func getMediaTypeForImpWithId(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			return openrtb_ext.BidTypeVideo, nil
		}
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\" ", impID),
	}
}

func CheckResponse(response *adapters.ResponseData) error {
	if response.StatusCode != http.StatusOK {
		return &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}
	}
	return nil
}

func convertToAdapterBidResponse(response *adapters.ResponseData, internalRequest *openrtb2.BidRequest) (*adapters.BidderResponse, []error) {
	var errs []error
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImpWithId(sb.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

func convertBidderNameInExt(imp *openrtb2.Imp) (*openrtb2.Imp, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}
	var unrulyExt openrtb_ext.ExtImpUnruly
	if err := json.Unmarshal(bidderExt.Bidder, &unrulyExt); err != nil {
		return nil, err
	}
	var impExtUnruly = ImpExtUnruly{Unruly: openrtb_ext.ExtImpUnruly{
		SiteID: unrulyExt.SiteID,
		UUID:   unrulyExt.UUID,
	}}
	bytes, err := json.Marshal(impExtUnruly)
	if err != nil {
		return nil, err
	}
	imp.Ext = bytes
	return imp, nil
}

func (a *UnrulyAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if err := CheckResponse(response); err != nil {
		return nil, []error{err}
	}
	return convertToAdapterBidResponse(response, internalRequest)
}

type ImpExtUnruly struct {
	Unruly openrtb_ext.ExtImpUnruly `json:"unruly"`
}
