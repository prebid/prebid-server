package mobfoxpb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type MobfoxpbAdapter struct {
	URI string
}

// NewMobfoxpbBidder Initializes the Bidder
func NewMobfoxpbBidder(endpoint string) *MobfoxpbAdapter {
	endpointURL, err := url.ParseRequestURI(endpoint)
	if err != nil {
		glog.Fatalf("invalid endpoint provided for Mobfox: %s, error: %v", endpoint, err)
		return nil
	}
	return &MobfoxpbAdapter{
		URI: endpointURL.String(),
	}
}

// MakeRequests create bid request for mobfoxpb demand
func (a *MobfoxpbAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var err error
	var tagID string

	var adapterRequests []*adapters.RequestData

	reqCopy := *request
	for _, imp := range request.Imp {
		reqCopy.Imp = []openrtb.Imp{imp}

		tagID, err = jsonparser.GetString(reqCopy.Imp[0].Ext, "bidder", "TagID")
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqCopy.Imp[0].TagID = tagID

		adapterReq, err := a.makeRequest(&reqCopy)
		if err != nil {
			errs = append(errs, err)
		}
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
	}
	return adapterRequests, errs
}

func (a *MobfoxpbAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, error) {
	reqJSON, err := json.Marshal(request)

	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
	}, nil
}

// MakeBids makes the bids
func (a *MobfoxpbAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidType, err := getMediaTypeForImp(bid.ImpID, internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType, nil
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to find impression \"%s\" ", impID),
	}
}
