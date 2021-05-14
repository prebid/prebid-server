package mobfoxpb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	ROUTE_NATIVE  = "o"
	ROUTE_RTB     = "rtb"
	METHOD_NATIVE = "ortb"
	METHOD_RTB    = "req"
	MACROS_ROUTE  = "__route__"
	MACROS_METHOD = "__method__"
	MACROS_KEY    = "__key__"
)

type adapter struct {
	URI string
}

// Builder builds a new instance of the Mobfox adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests create bid request for mobfoxpb demand
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var route string
	var method string
	var adapterRequests []*adapters.RequestData

	requestURI := a.URI
	reqCopy := *request
	imp := request.Imp[0]
	tagID, errTag := jsonparser.GetString(imp.Ext, "bidder", "TagID")
	key, errKey := jsonparser.GetString(imp.Ext, "bidder", "key")
	if errTag != nil && errKey != nil {
		errs = append(errs, &errortypes.BadInput{
			Message: fmt.Sprintf("Invalid or non existing key and tagId, atleast one should be present"),
		})
		return nil, errs
	}

	if key != "" {
		route = ROUTE_RTB
		method = METHOD_RTB
		requestURI = strings.Replace(requestURI, MACROS_KEY, key, 1)
	} else if tagID != "" {
		method = METHOD_NATIVE
		route = ROUTE_NATIVE
	}

	requestURI = strings.Replace(requestURI, MACROS_ROUTE, route, 1)
	requestURI = strings.Replace(requestURI, MACROS_METHOD, method, 1)

	reqCopy.Imp = []openrtb2.Imp{imp}
	adapterReq, err := a.makeRequest(&reqCopy, requestURI)
	if err != nil {
		errs = append(errs, err)
	}
	if adapterReq != nil {
		adapterRequests = append(adapterRequests, adapterReq)
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, requestURI string) (*adapters.RequestData, error) {
	reqJSON, err := json.Marshal(request)

	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     requestURI,
		Body:    reqJSON,
		Headers: headers,
	}, nil
}

// MakeBids makes the bids
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

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
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType, nil
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to find impression \"%s\"", impID),
	}
}
