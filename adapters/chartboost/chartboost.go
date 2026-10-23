package chartboost

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// Chartboost bids exclusively in USD and interprets imp.bidfloor as a USD amount.
const bidderCurrency = "USD"

type adapter struct {
	endpoint string
}

// MakeRequests forwards the bid request to the Chartboost endpoint verbatim, except that
// non-USD bid floors are converted to USD. Request shaping is otherwise left to Chartboost.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	imps := make([]openrtb2.Imp, 0, len(request.Imp))
	for _, imp := range request.Imp {
		if err := convertBidFloorToUSD(&imp, reqInfo); err != nil {
			errs = append(errs, err)
			continue
		}
		imps = append(imps, imp)
	}
	if len(imps) == 0 {
		return nil, errs
	}

	// Shallow copy so the imp slice can be replaced without touching the shared request.
	requestCopy := *request
	requestCopy.Imp = imps

	body, err := jsonutil.Marshal(&requestCopy)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-OpenRTB-Version", "2.6")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps),
	}}, errs
}

// convertBidFloorToUSD rewrites a non-USD bid floor into USD using the conversion rates
// known to Prebid Server. The imp is a shallow copy, so modifying its scalar fields is safe.
func convertBidFloorToUSD(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor <= 0 || imp.BidFloorCur == "" || strings.EqualFold(imp.BidFloorCur, bidderCurrency) {
		return nil
	}

	converted, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, bidderCurrency)
	if err != nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unable to convert bid floor of imp %s from %s to %s: %s", imp.ID, imp.BidFloorCur, bidderCurrency, err),
		}
	}

	imp.BidFloor = converted
	imp.BidFloorCur = bidderCurrency
	return nil
}

// MakeBids does no bid shaping beyond what prebid-server core requires, which is a
// media type on every bid. Everything else is done on Chartboost side.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(respData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(respData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(respData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	var errs []error
	resp := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getMediaType(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			resp.Bids = append(resp.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	if len(resp.Bids) == 0 {
		return nil, errs
	}
	if bidResp.Cur != "" {
		resp.Currency = bidResp.Cur
	}
	return resp, errs
}

func getMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d for bid %s", bid.MType, bid.ID),
		}
	}
}

// Builder builds a new instance of the Chartboost adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, serverCfg config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}
