package adman

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// AdmanAdapter struct
type AdmanAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name return actual adapter name
func (a *AdmanAdapter) Name() string {
	return "adman"
}

func (a *AdmanAdapter) FamilyName() string {
	return "adman"
}

// SkipNoCookies will not skip bids without synced users
func (a *AdmanAdapter) SkipNoCookies() bool {
	return false
}

// NewAdmanAdapter create a new SovrnSonobiAdapter instance
func NewAdmanAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *AdmanAdapter {
	return NewAdmanBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

// NewAdmanBidder Initializes the Bidder
func NewAdmanBidder(client *http.Client, endpoint string) *AdmanAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &AdmanAdapter{
		http: a,
		URI:  endpoint,
	}
}

type admanParams struct {
	TagID string `json:"TagID"`
}

// MakeRequests create bid request for adman demand
func (a *AdmanAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var admanExt openrtb_ext.ExtImpAdman
	var err error

	var adapterRequests []*adapters.RequestData

	for _, imp := range request.Imp {
		reqCopy := *request
		reqCopy.Imp = append(make([]openrtb.Imp, 0, 1), imp)

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(reqCopy.Imp[0].Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		if err = json.Unmarshal(bidderExt.Bidder, &admanExt); err != nil {
			errs = append(errs, err)
			continue
		}

		reqCopy.Imp[0].TagID = admanExt.TagID

		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}
	return adapterRequests, errs
}

func (a *AdmanAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {

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
	}, errs
}

// MakeBids makes the bids
func (a *AdmanAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
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
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
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
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\" ", impID),
	}
}
