package insticator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/mathutil"
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

	// Create a copy of the request to avoid modifying the original request
	requestCopy := *request
	isPublisherIdPopulated := false // Flag to track if populatePublisherId has been called

	for i := 0; i < len(request.Imp); i++ {
		impCopy, impKey, publisherId, err := makeImps(request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Populate publisher.id from imp extension
		if !isPublisherIdPopulated {
			populatePublisherId(publisherId, &requestCopy)
			isPublisherIdPopulated = true
		}

		resolvedBidFloor, errFloor := resolveBidFloor(impCopy.BidFloor, impCopy.BidFloorCur, requestInfo)
		if errFloor != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Error in converting the provided bid floor currency from %s to USD",
					impCopy.BidFloorCur),
			})
			continue
		}
		if resolvedBidFloor > 0 {
			impCopy.BidFloor = resolvedBidFloor
			impCopy.BidFloorCur = "USD"
		}

		groupedImps[impKey] = append(groupedImps[impKey], impCopy)
	}

	for _, impList := range groupedImps {
		if adapterReq, err := a.makeRequest(&requestCopy, impList); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errs = append(errs, err)
		}
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, impList []openrtb2.Imp) (*adapters.RequestData, error) {
	request.Imp = impList

	reqJSON, err := jsonutil.Marshal(request)
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
			headers.Set("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Set("X-Forwarded-For", request.Device.IP)
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
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
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

func makeImps(imp openrtb2.Imp) (openrtb2.Imp, string, string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb2.Imp{}, "", "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var insticatorExt openrtb_ext.ExtImpInsticator
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &insticatorExt); err != nil {
		return openrtb2.Imp{}, "", "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	// Directly construct the impExt
	impExt := ext{
		Insticator: impInsticatorExt{
			AdUnitId:    insticatorExt.AdUnitId,
			PublisherId: insticatorExt.PublisherId,
		},
	}

	impExtJSON, err := jsonutil.Marshal(impExt)
	if err != nil {
		return openrtb2.Imp{}, "", "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	imp.Ext = impExtJSON

	// Validate Video if it exists
	if imp.Video != nil {
		if err := validateVideoParams(imp.Video); err != nil {
			return openrtb2.Imp{}, insticatorExt.AdUnitId, insticatorExt.PublisherId, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	// Return the imp, AdUnitId, and no error
	return imp, insticatorExt.AdUnitId, insticatorExt.PublisherId, nil
}

func makeReqExt(request *openrtb2.BidRequest) ([]byte, error) {
	var reqExt reqExt

	if len(request.Ext) > 0 {
		if err := jsonutil.Unmarshal(request.Ext, &reqExt); err != nil {
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

	return jsonutil.Marshal(reqExt)
}

func resolveBidFloor(bidFloor float64, bidFloorCur string, reqInfo *adapters.ExtraRequestInfo) (float64, error) {
	if bidFloor > 0 && bidFloorCur != "" && strings.ToUpper(bidFloorCur) != "USD" {
		floor, err := reqInfo.ConvertCurrency(bidFloor, bidFloorCur, "USD")
		return mathutil.RoundTo4Decimals(floor), err
	}

	return bidFloor, nil
}

func validateVideoParams(video *openrtb2.Video) error {
	if video.W == nil || *video.W == 0 || video.H == nil || *video.H == 0 || video.MIMEs == nil {
		return &errortypes.BadInput{
			Message: "One or more invalid or missing video field(s) w, h, mimes",
		}
	}

	return nil
}

// populatePublisherId function populates site.publisher.id or app.publisher.id
func populatePublisherId(publisherId string, request *openrtb2.BidRequest) {

	// Populate site.publisher.id if request.Site is not nil
	if request.Site != nil {
		// Make a shallow copy of Site if it already exists
		siteCopy := *request.Site
		request.Site = &siteCopy

		// Make a shallow copy of Publisher if it already exists
		if request.Site.Publisher != nil {
			publisherCopy := *request.Site.Publisher
			request.Site.Publisher = &publisherCopy
		} else {
			request.Site.Publisher = &openrtb2.Publisher{}
		}

		request.Site.Publisher.ID = publisherId
	}

	// Populate app.publisher.id if request.App is not nil
	if request.App != nil {
		// Make a shallow copy of App if it already exists
		appCopy := *request.App
		request.App = &appCopy

		// Make a shallow copy of Publisher if it already exists
		if request.App.Publisher != nil {
			publisherCopy := *request.App.Publisher
			request.App.Publisher = &publisherCopy
		} else {
			request.App.Publisher = &openrtb2.Publisher{}
		}

		request.App.Publisher.ID = publisherId
	}
}
