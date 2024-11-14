package insticator

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type ext struct {
	Insticator impInsticatorExt `json:"insticator"`
}

type impInsticatorExt struct {
	AdUnitId    string `json:"adUnitId"`
	PublisherId string `json:"publisherId"`
}

type adapter struct {
	endpoint string
}

type reqExt struct {
	Insticator *reqInsticatorExt `json:"insticator,omitempty"`
}

type reqInsticatorExt struct {
	Caller []insticatorCaller `json:"caller,omitempty"`
}

type insticatorCaller struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// caller Info used to track Prebid Server
// as one of the hops in the request to exchange
var caller = insticatorCaller{"Prebid-Server", "n/a"}

type bidExt struct {
	Insticator bidInsticatorExt `json:"insticator,omitempty"`
}

type bidInsticatorExt struct {
	MediaType string `json:"mediaType,omitempty"`
}

// Builder builds a new instance of the Insticator adapter with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// getMediaTypeForBid figures out which media type this bid is for
func getMediaTypeForBid(bid *openrtb2.Bid) openrtb_ext.BidType {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo
	default:
		return openrtb_ext.BidTypeBanner
	}
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData
	var groupedImps = make(map[string][]openrtb2.Imp)

	reqExt, err := makeReqExt(request)
	if err != nil {
		errs = append(errs, err)
	}

	request.Ext = reqExt

	// Create a deep copy of the request to avoid modifying the original request
	requestCopy := *request
	requestCopy.Site = request.Site
	requestCopy.App = request.App

	for i := 0; i < len(request.Imp); i++ {
		impCopy, impKey, err := makeImps(request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Populate site.publisher.id from imp extension
		if requestCopy.Site != nil || requestCopy.App != nil {
			if err := populatePublisherId(&impCopy, &requestCopy); err != nil {
				errs = append(errs, err)
				continue
			}
		}

		resolvedBidFloor, errFloor := resolveBidFloor(impCopy.BidFloor, impCopy.BidFloorCur, requestInfo)
		if errFloor != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Error in converting the provided bid floor currency from %s to USD",
					impCopy.BidFloorCur),
			})
			continue
		} else {
			if resolvedBidFloor > 0 {
				impCopy.BidFloor = resolvedBidFloor
				impCopy.BidFloorCur = "USD"
			}
		}

		groupedImps[impKey] = append(groupedImps[impKey], impCopy)
	}

	for _, impList := range groupedImps {
		if adapterReq, err := a.makeRequest(requestCopy, impList); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errs = append(errs, err)
		}
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request openrtb2.BidRequest, impList []openrtb2.Imp) (*adapters.RequestData, error) {
	request.Imp = impList

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
			headers.Add("IP", request.Device.IP)
		}
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType := getMediaTypeForBid(bid)
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func makeImps(imp openrtb2.Imp) (openrtb2.Imp, string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb2.Imp{}, "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var insticatorExt openrtb_ext.ExtImpInsticator
	if err := json.Unmarshal(bidderExt.Bidder, &insticatorExt); err != nil {
		return openrtb2.Imp{}, "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	// check if the adUnitId is not empty
	if insticatorExt.AdUnitId == "" {
		return openrtb2.Imp{}, "", &errortypes.BadInput{
			Message: "Missing adUnitId",
		}
	}

	// check if the publisherId is not empty
	if insticatorExt.PublisherId == "" {
		return openrtb2.Imp{}, insticatorExt.AdUnitId, &errortypes.BadInput{
			Message: "Missing publisherId",
		}
	}

	// Directly construct the impExt
	impExt := ext{
		Insticator: impInsticatorExt{
			AdUnitId:    insticatorExt.AdUnitId,
			PublisherId: insticatorExt.PublisherId,
		},
	}

	impExtJSON, err := json.Marshal(impExt)
	if err != nil {
		return openrtb2.Imp{}, "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	imp.Ext = impExtJSON

	// Validate Video if it exists
	if imp.Video != nil {
		if err := validateVideoParams(imp.Video); err != nil {
			return openrtb2.Imp{}, insticatorExt.AdUnitId, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	// Return the imp, AdUnitId, and no error
	return imp, insticatorExt.AdUnitId, nil
}

func makeReqExt(request *openrtb2.BidRequest) ([]byte, error) {
	var reqExt reqExt

	if len(request.Ext) > 0 {
		if err := json.Unmarshal(request.Ext, &reqExt); err != nil {
			return nil, err
		}
	}

	if reqExt.Insticator == nil {
		reqExt.Insticator = &reqInsticatorExt{}
	}

	if reqExt.Insticator.Caller == nil {
		reqExt.Insticator.Caller = make([]insticatorCaller, 0)
	}

	reqExt.Insticator.Caller = append(reqExt.Insticator.Caller, caller)

	return json.Marshal(reqExt)
}

func resolveBidFloor(bidFloor float64, bidFloorCur string, reqInfo *adapters.ExtraRequestInfo) (float64, error) {
	if bidFloor > 0 && bidFloorCur != "" && strings.ToUpper(bidFloorCur) != "USD" {
		floor, err := reqInfo.ConvertCurrency(bidFloor, bidFloorCur, "USD")
		return roundTo4Decimals(floor), err
	}

	return bidFloor, nil
}

// roundTo4Decimals function
func roundTo4Decimals(amount float64) float64 {
	return math.Round(amount*10000) / 10000
}

func validateVideoParams(video *openrtb2.Video) error {
	videoCopy := *video
	if (videoCopy.W == nil || *videoCopy.W == 0) ||
		(videoCopy.H == nil || *videoCopy.H == 0) ||
		videoCopy.MIMEs == nil {

		return &errortypes.BadInput{
			Message: "One or more invalid or missing video field(s) w, h, mimes",
		}
	}

	return nil
}

// populatePublisherId function populates site.publisher.id or app.publisher.id
func populatePublisherId(imp *openrtb2.Imp, request *openrtb2.BidRequest) error {
	var ext ext

	// Unmarshal the imp extension to get the publisher ID
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return &errortypes.BadInput{Message: "Error unmarshalling imp extension"}
	}

	// Populate site.publisher.id if request.Site is not nil
	if request.Site != nil {
		if request.Site.Publisher == nil {
			request.Site.Publisher = &openrtb2.Publisher{}
		}
		request.Site.Publisher.ID = ext.Insticator.PublisherId
	}

	// Populate app.publisher.id if request.App is not nil
	if request.App != nil {
		if request.App.Publisher == nil {
			request.App.Publisher = &openrtb2.Publisher{}
		}
		request.App.Publisher.ID = ext.Insticator.PublisherId
	}
	return nil
}
