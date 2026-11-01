package melvad

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Melvad adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	return &adapter{endpoint: endpoint}, nil
}

// MakeRequests forwards the OpenRTB request as is. The impressions are grouped by sspId, because the sspId
// is a part of the endpoint URL: one request per SSP.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	impsBySsp := make(map[string][]openrtb2.Imp)
	var sspIDs []string

	for _, imp := range request.Imp {
		sspID, err := impSspID(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if _, exists := impsBySsp[sspID]; !exists {
			sspIDs = append(sspIDs, sspID)
		}

		impsBySsp[sspID] = append(impsBySsp[sspID], imp)
	}

	requests := make([]*adapters.RequestData, 0, len(sspIDs))

	for _, sspID := range sspIDs {
		requestData, err := a.makeRequest(request, sspID, impsBySsp[sspID])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, requestData)
	}

	return requests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, sspID string, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	uri, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{SspID: sspID})
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Unable to build the endpoint url: %v", err),
		}
	}

	requestCopy := *request
	requestCopy.Imp = imps

	body, err := jsonutil.Marshal(&requestCopy)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps),
	}, nil
}

func impSspID(imp *openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Impression %s: ext not valid or can't be unmarshalled: %v", imp.ID, err),
		}
	}

	var melvadExt openrtb_ext.ExtImpMelvad
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &melvadExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Impression %s: ext.bidder not valid or can't be unmarshalled: %v", imp.ID, err),
		}
	}

	if melvadExt.SspID == "" {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Impression %s: ext.bidder.sspId is required", imp.ID),
		}
	}

	return melvadExt.SspID, nil
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
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %v", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(bid, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			resolveMacros(bid)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
				BidMeta: &openrtb_ext.ExtBidPrebidMeta{AdvertiserDomains: bid.ADomain},
			})
		}
	}

	return bidResponse, errs
}

// getMediaTypeForBid takes bid.mtype when the bidder sends it, else the media type of the impression the bid answers.
func getMediaTypeForBid(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
		}

		switch {
		case imp.Banner != nil:
			return openrtb_ext.BidTypeBanner, nil
		case imp.Video != nil:
			return openrtb_ext.BidTypeVideo, nil
		case imp.Native != nil:
			return openrtb_ext.BidTypeNative, nil
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Unable to determine the media type of the bid %s for the impression %s", bid.ID, bid.ImpID),
	}
}

// resolveMacros fills ${AUCTION_PRICE} in the markup and the billing url with the bid price.
func resolveMacros(bid *openrtb2.Bid) {
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.ReplaceAll(bid.AdM, "${AUCTION_PRICE}", price)
	bid.BURL = strings.ReplaceAll(bid.BURL, "${AUCTION_PRICE}", price)
}
