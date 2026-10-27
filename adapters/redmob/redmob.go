package redmob

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/util/ptrutil"
)

const auctionPriceMacro = "${AUCTION_PRICE}"

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Redmob adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error
	for _, imp := range request.Imp {
		// supplyPartnerId stays in imp.ext.bidder, where the exchange reads it to attribute the traffic.
		if err := validateImpExt(imp); err != nil {
			errs = append(errs, err)
			continue
		}
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {

			convertedBidFloor, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				errs = append(errs, err)
				continue
			}
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedBidFloor
		}
		setBannerSize(&imp)

		requestCopy := *request
		requestCopy.Imp = []openrtb2.Imp{imp}
		requestCopy.Cur = []string{"USD"}
		requestBodyJSON, err := jsonutil.Marshal(&requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		requestData := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    requestBodyJSON,
			Headers: headers,
			ImpIDs:  []string{imp.ID},
		}
		requests = append(requests, requestData)
	}
	return requests, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if responseData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}}
	}
	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("failed to parse bid response: %s", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(requestData.ImpIDs))
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
			resolveAuctionPrice(bid)
			typedBid := &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			}
			if len(bid.ADomain) > 0 {
				typedBid.BidMeta = &openrtb_ext.ExtBidPrebidMeta{AdvertiserDomains: bid.ADomain}
			}
			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
	}
	return bidResponse, errs
}

// resolveAuctionPrice replaces the ${AUCTION_PRICE} macro with the bid price in the exchange's currency (USD).
// Left unresolved, the Prebid Universal Creative would fill it with the ad server price, which may be in another
// currency and quantized by the price granularity; the exchange's win notices record that value as the USD cost.
func resolveAuctionPrice(bid *openrtb2.Bid) {
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.ReplaceAll(bid.AdM, auctionPriceMacro, price)
	bid.NURL = strings.ReplaceAll(bid.NURL, auctionPriceMacro, price)
	bid.BURL = strings.ReplaceAll(bid.BURL, auctionPriceMacro, price)
}

// validateImpExt checks that the impression carries a readable Redmob bidder extension.
func validateImpExt(imp openrtb2.Imp) error {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext for imp %s: %s", imp.ID, err)}
	}
	var redmobExt openrtb_ext.ImpExtRedmob
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &redmobExt); err != nil {
		return &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext.bidder for imp %s: %s", imp.ID, err)}
	}
	return nil
}

// getMediaTypeForBid uses bid.mtype, which the exchange sets on every bid. Inferring the type from the
// impression is only a safety net for a bid without it, and only works for single-format impressions.
func getMediaTypeForBid(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case 0:
		// no mtype: fall through to the impression
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported bid.mtype %d for imp %s", bid.MType, bid.ImpID),
		}
	}

	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
		}
		if imp.Banner != nil && imp.Video == nil {
			return openrtb_ext.BidTypeBanner, nil
		}
		if imp.Video != nil && imp.Banner == nil {
			return openrtb_ext.BidTypeVideo, nil
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid.mtype is required to resolve the media type of imp %s", bid.ImpID),
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("bid references unknown imp %s", bid.ImpID),
	}
}

// setBannerSize fills banner.w and banner.h from the first format when they are missing,
// because the Redmob exchange requires both. The banner is copied, never modified in place.
func setBannerSize(imp *openrtb2.Imp) {
	if imp.Banner == nil || (imp.Banner.W != nil && imp.Banner.H != nil) || len(imp.Banner.Format) == 0 {
		return
	}
	banner := *imp.Banner
	banner.W = ptrutil.ToPtr(banner.Format[0].W)
	banner.H = ptrutil.ToPtr(banner.Format[0].H)
	imp.Banner = &banner
}
