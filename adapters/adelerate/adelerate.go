package adelerate

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

const defaultCurrency = "USD"

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	// PBS resolves stored impressions within one publisher account per request.
	type publisherRequest struct {
		publisherID string
		imps        []openrtb2.Imp
	}
	var groups []publisherRequest
	publishers := make(map[string]int)

	for i := range request.Imp {
		currImp := request.Imp[i]

		var extImpBidder adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(currImp.Ext, &extImpBidder); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext for impression %s: %s", currImp.ID, err.Error()),
			})
			continue
		}

		var bidderExt openrtb_ext.ImpExtAdelerate
		if err := jsonutil.Unmarshal(extImpBidder.Bidder, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext for impression %s: %s", currImp.ID, err.Error()),
			})
			continue
		}
		if bidderExt.PlacementID == "" || bidderExt.PublisherID == "" {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("missing placementId or publisherId for impression %s", currImp.ID)})
			continue
		}

		if bidderExt.Floor > 0 && currImp.BidFloor == 0 {
			floorCurrency := bidderExt.FloorCurrency
			if floorCurrency == "" {
				floorCurrency = defaultCurrency
			}

			floor := bidderExt.Floor
			if floorCurrency != defaultCurrency {
				convertedFloor, err := reqInfo.ConvertCurrency(floor, floorCurrency, defaultCurrency)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				floor = convertedFloor
			}

			currImp.BidFloor = floor
			currImp.BidFloorCur = defaultCurrency
		}

		if currImp.Secure == nil {
			currImp.Secure = openrtb2.Int8Ptr(1)
		}

		var impExt map[string]json.RawMessage
		if err := jsonutil.Unmarshal(currImp.Ext, &impExt); err != nil {
			errs = append(errs, err)
			continue
		}
		delete(impExt, "bidder")
		// The caller's PBS configuration belongs to the outer auction. Demand
		// bidder parameters must come from the Adelerate stored impression.
		impExt["prebid"], _ = jsonutil.Marshal(struct {
			StoredRequest openrtb_ext.ExtStoredRequest `json:"storedrequest"`
		}{StoredRequest: openrtb_ext.ExtStoredRequest{ID: bidderExt.PlacementID}})
		currImp.Ext, _ = jsonutil.Marshal(impExt)

		group, exists := publishers[bidderExt.PublisherID]
		if !exists {
			group = len(groups)
			publishers[bidderExt.PublisherID] = group
			groups = append(groups, publisherRequest{publisherID: bidderExt.PublisherID})
		}
		groups[group].imps = append(groups[group].imps, currImp)
	}

	var requests []*adapters.RequestData
	for _, group := range groups {
		reqCopy := *request
		reqCopy.Imp = group.imps
		var err error
		reqCopy.Ext, err = withoutPrebid(request.Ext)
		if err != nil {
			return nil, append(errs, &errortypes.BadInput{Message: fmt.Sprintf("invalid request.ext: %s", err)})
		}
		publisher := &openrtb2.Publisher{ID: group.publisherID}
		if request.App != nil {
			app := *request.App
			app.Publisher = publisher
			reqCopy.App = &app
		} else if request.Site != nil {
			site := *request.Site
			site.Publisher = publisher
			reqCopy.Site = &site
		} else {
			return nil, append(errs, &errortypes.BadInput{Message: "request must contain site or app"})
		}
		adapterReq, err := a.buildRequest(&reqCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, adapterReq)
	}
	return requests, errs
}

func withoutPrebid(ext json.RawMessage) (json.RawMessage, error) {
	if len(ext) == 0 {
		return nil, nil
	}
	var fields map[string]json.RawMessage
	if err := jsonutil.Unmarshal(ext, &fields); err != nil {
		return nil, err
	}
	delete(fields, "prebid")
	if len(fields) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(fields)
}

func (a *adapter) buildRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	reqJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  http.MethodPost,
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

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case 0:
		var ext openrtb_ext.ExtBid
		if len(bid.Ext) > 0 && jsonutil.Unmarshal(bid.Ext, &ext) == nil && ext.Prebid != nil {
			switch ext.Prebid.Type {
			case openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative:
				return ext.Prebid.Type, nil
			}
		}
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s missing supported mtype or ext.prebid.type", bid.ID)}
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %s", bid.MType, bid.ID),
		}
	}
}
