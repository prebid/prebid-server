package colossus

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type ColossusAdapter struct {
	URI string
}

type ColossusResponseBidExt struct {
	MediaType string `json:"mediaType"`
}

// Builder builds a new instance of the Colossus adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &ColossusAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests create bid request for colossus demand
func (a *ColossusAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	reqCopy := *request
	for _, imp := range request.Imp {
		reqCopy.Imp = []openrtb2.Imp{imp}

		if tagID, err := jsonparser.GetString(reqCopy.Imp[0].Ext, "bidder", "TagID"); err == nil {
			reqCopy.Imp[0].TagID = tagID
		} else if err != jsonparser.KeyPathNotFoundError {
			errs = append(errs, err)
			continue
		}

		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}
	return adapterRequests, errs
}

func (a *ColossusAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {

	var errs []error

	reqJSON, err := json.Marshal(request)

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, errs
}

// MakeBids makes the bids
func (a *ColossusAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImp(sb.Bid[i], internalRequest.Imp)
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

func getMediaTypeForImp(bid openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	var bidExt ColossusResponseBidExt
	err := jsonutil.Unmarshal(bid.Ext, &bidExt)
	if err == nil {
		switch bidExt.MediaType {
		case "banner":
			return openrtb_ext.BidTypeBanner, nil
		case "native":
			return openrtb_ext.BidTypeNative, nil
		case "video":
			return openrtb_ext.BidTypeVideo, nil
		}
	}

	for _, imp := range imps {
		if imp.ID == bid.ImpID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\"", bid.ImpID),
	}
}
