package adspiro

import (
	"fmt"
	"net/http"
	"net/url"
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

// Builder builds a new instance of the Adspiro adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %w", err)
	}
	return &adapter{endpoint: endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var publisherIDs []string
	impsByPublisherID := make(map[string][]openrtb2.Imp)

	for _, imp := range request.Imp {
		publisherID, err := getPublisherID(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := convertBidFloor(&imp, requestInfo); err != nil {
			errs = append(errs, err)
			continue
		}

		if _, ok := impsByPublisherID[publisherID]; !ok {
			publisherIDs = append(publisherIDs, publisherID)
		}
		impsByPublisherID[publisherID] = append(impsByPublisherID[publisherID], imp)
	}

	outgoingRequest := *request
	outgoingRequest.Cur = []string{"USD"}

	var requests []*adapters.RequestData
	for _, publisherID := range publisherIDs {
		outgoingRequest.Imp = impsByPublisherID[publisherID]

		requestData, err := a.makeRequestData(&outgoingRequest, publisherID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, requestData)
	}

	return requests, errs
}

func getPublisherID(imp *openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext: %v", imp.ID, err),
		}
	}

	var impExt openrtb_ext.ExtImpAdspiro
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext.bidder: %v", imp.ID, err),
		}
	}

	return impExt.PublisherID, nil
}

func convertBidFloor(imp *openrtb2.Imp, requestInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor <= 0 || imp.BidFloorCur == "" || strings.EqualFold(imp.BidFloorCur, "USD") {
		return nil
	}

	bidFloor, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
	if err != nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to convert bidfloor from %s to USD: %v", imp.ID, imp.BidFloorCur, err),
		}
	}

	imp.BidFloor = bidFloor
	imp.BidFloorCur = "USD"
	return nil
}

func (a *adapter) makeRequestData(request *openrtb2.BidRequest, publisherID string) (*adapters.RequestData, error) {
	uri, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{PublisherID: url.QueryEscape(publisherID)})
	if err != nil {
		return nil, err
	}

	body, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.6")

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getBidType(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			typedBid := &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			}
			if bidType == openrtb_ext.BidTypeVideo {
				typedBid.BidVideo = getBidVideo(bid)
			}
			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
	}

	return bidResponse, errs
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case 0:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("missing mtype for bid %s on imp %s", bid.ID, bid.ImpID),
		}
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %s on imp %s", bid.MType, bid.ID, bid.ImpID),
		}
	}
}

func getBidVideo(bid *openrtb2.Bid) *openrtb_ext.ExtBidPrebidVideo {
	bidVideo := &openrtb_ext.ExtBidPrebidVideo{Duration: int(bid.Dur)}
	if len(bid.Cat) > 0 {
		bidVideo.PrimaryCategory = bid.Cat[0]
	}
	return bidVideo
}
