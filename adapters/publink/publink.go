// Package publink is the Prebid Server (Go) bidder adapter for Publink.
//
// Server-to-server counterpart of the Prebid.js adapter: the host's Prebid
// Server hands this adapter the bidder params under imp.ext.bidder, the
// adapter copies placementId onto imp.tagid and POSTs the oRTB request to the
// Publink endpoint, one request per impression (the endpoint limits
// concurrency per tag, so batching would queue placements behind each other).
// The endpoint already resolves imp.ext.bidder.placementId itself, so a host
// that forwards the raw request also works; tagid is set for clarity.
package publink

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Publink adapter for the given bidder
// with the given config. The endpoint comes from static/bidder-info/publink.yaml.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

// ExtImpPublink contains optional placementId, bidFloor and currency params.
// Without placementId, the adapter uses the standard imp.tagid.
type ExtImpPublink struct {
	PlacementID string  `json:"placementId"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
	Currency    string  `json:"currency,omitempty"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	for i := range request.Imp {
		imp := request.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("imp[%d].ext: %v", i, err)})
			continue
		}
		var params ExtImpPublink
		if err := json.Unmarshal(bidderExt.Bidder, &params); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("imp[%d].ext.bidder: %v", i, err)})
			continue
		}
		if params.PlacementID == "" {
			params.PlacementID = imp.TagID
		}
		if strings.TrimSpace(params.PlacementID) == "" {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: placementId is required", i)})
			continue
		}

		if params.BidFloor < 0 || math.IsNaN(params.BidFloor) || math.IsInf(params.BidFloor, 0) || (params.Currency != "" && params.Currency != "USD") {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: bidFloor must be nonnegative and currency must be USD", i)})
			continue
		}
		// The join to the Publink portal: placementId overrides standard tagid.
		imp.TagID = params.PlacementID
		if params.BidFloor > 0 && imp.BidFloor <= 0 {
			imp.BidFloor = params.BidFloor
			imp.BidFloorCur = "USD"
		}

		// One request per impression.
		reqCopy := *request
		reqCopy.Imp = []openrtb2.Imp{imp}

		body, err := json.Marshal(reqCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")
		headers.Add("x-openrtb-version", "2.6")

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    body,
			Headers: headers,
			ImpIDs:  []string{imp.ID},
		})
	}
	return requests, errs
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = "USD"
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getMediaType(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, errs
}

// getMediaType uses the oRTB 2.6 bid.mtype (1 banner, 2 video, 3 audio) that
// the Publink endpoint always sets on every bid. Prebid Server expects the
// media type to be stated on the response, so an unset/unknown mtype is an
// explicit error rather than a guess from the impression.
func getMediaType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	default:
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("unsupported mtype %d for bid %s (imp %s)", bid.MType, bid.ID, bid.ImpID)}
	}
}
