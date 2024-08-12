package insticator

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
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

// Builder builds a new insticatorance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// getMediaTypeForImp figures out which media type this bid is for
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

	for i := 0; i < len(request.Imp); i++ {
		impCopy, err := makeImps(request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		var impExt ext

		// Populate site.publisher.id from imp extension
		if request.Site != nil || request.App != nil {
			populatePublisherId(&impCopy, request)
		}

		// Group together the imps having Insticator adUnitId. However, let's not block request creation.
		if err := json.Unmarshal(impCopy.Ext, &impExt); err != nil {
			errs = append(errs, err)
			continue
		}

		impKey := impExt.Insticator.AdUnitId

		resolvedBidFloor, errFloor := resolveBidFloor(impCopy.BidFloor, impCopy.BidFloorCur, requestInfo)
		if errFloor != nil {
			errs = append(errs, errFloor)
		} else {
			if resolvedBidFloor > 0 {
				impCopy.BidFloor = resolvedBidFloor
				impCopy.BidFloorCur = "USD"
			}
		}

		groupedImps[impKey] = append(groupedImps[impKey], impCopy)
	}

	for _, impList := range groupedImps {
		if adapterReq, err := a.makeRequest(*request, impList); err == nil {
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

func makeImps(imp openrtb2.Imp) (openrtb2.Imp, error) {

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var insticatorExt openrtb_ext.ExtImpInsticator
	if err := json.Unmarshal(bidderExt.Bidder, &insticatorExt); err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var impExt ext
	impExt.Insticator.AdUnitId = insticatorExt.AdUnitId
	impExt.Insticator.PublisherId = insticatorExt.PublisherId

	impExtJSON, err := json.Marshal(impExt)
	if err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	imp.Ext = impExtJSON
	// Validate Video if it exists
	if imp.Video != nil {
		videoCopy, err := validateVideoParams(imp.Video)

		imp.Video = videoCopy

		if err != nil {
			return openrtb2.Imp{}, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	return imp, nil
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

func validateVideoParams(video *openrtb2.Video) (*openrtb2.Video, error) {
	videoCopy := *video
	if (videoCopy.W == nil || *videoCopy.W == 0) ||
		(videoCopy.H == nil || *videoCopy.H == 0) ||
		videoCopy.MIMEs == nil {

		return nil, &errortypes.BadInput{
			Message: "One or more invalid or missing video field(s) w, h, mimes",
		}
	}

	return &videoCopy, nil
}

// populate publisherId to site/app object from imp extension
func populatePublisherId(imp *openrtb2.Imp, request *openrtb2.BidRequest) {
	var ext ext

	if request.Site != nil && request.Site.Publisher == nil {
		request.Site.Publisher = &openrtb2.Publisher{}
		if err := json.Unmarshal(imp.Ext, &ext); err == nil {
			request.Site.Publisher.ID = ext.Insticator.PublisherId
		} else {
			log.Printf("Error unmarshalling imp extension: %v", err)
		}
	}

	if request.App != nil && request.App.Publisher == nil {
		request.App.Publisher = &openrtb2.Publisher{}
		if err := json.Unmarshal(imp.Ext, &ext); err == nil {
			request.App.Publisher.ID = ext.Insticator.PublisherId
		} else {
			log.Printf("Error unmarshalling imp extension: %v", err)
		}
	}
}
