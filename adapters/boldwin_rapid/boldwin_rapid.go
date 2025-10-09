package boldwin_rapid

import (
	"encoding/json"
	"fmt"
	"text/template"

	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint template: %v", err)
	}

	bidder := &adapter{
		endpoint: endpointTemplate,
	}

	return bidder, nil
}

func (a adapter) buildEndpointURL(boldwinExt openrtb_ext.ImpExtBoldwinRapid) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		PublisherID: boldwinExt.Pid,
		PlacementID: boldwinExt.Tid,
	}

	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	reqCopy := *request

	for _, imp := range request.Imp {
		// Create a new request with just this impression
		reqCopy.Imp = []openrtb2.Imp{imp}

		var bidderExt adapters.ExtImpBidder
		var boldwinExt openrtb_ext.ImpExtBoldwinRapid

		// Use the current impression's Ext
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{err}
		}

		if err := jsonutil.Unmarshal(bidderExt.Bidder, &boldwinExt); err != nil {
			return nil, []error{err}
		}

		endpoint, err := a.buildEndpointURL(boldwinExt)
		if err != nil {
			return nil, []error{err}
		}

		adapterReq, err := a.makeRequest(&reqCopy, endpoint)
		if err != nil {
			return nil, []error{err}
		}

		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
	}

	return adapterRequests, nil
}

func (a *adapter) getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")
	headers.Add("Host", "rtb.beardfleet.com") // required header for the request

	if request.Device != nil {
		if request.Device.UA != "" {
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

	return headers
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, endpoint string) (*adapters.RequestData, error) {
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := a.getHeaders(request)

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	err := adapters.CheckResponseStatusCodeForErrors(responseData)
	if err != nil {
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
			bidType, err := getBidMediaType(&seatBid.Bid[i])
			if err != nil {
				return nil, []error{err}
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getBidMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unable to fetch mediaType in multi-format: %s", bid.ImpID)
	}
}
