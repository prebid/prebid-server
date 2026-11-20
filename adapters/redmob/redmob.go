package redmob

import (
	"encoding/json"
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
		if request.Device != nil {
			if request.Device.UA != "" {
				headers.Add("User-Agent", request.Device.UA)
			}
			if request.Device.IP != "" {
				headers.Add("X-Forwarded-For", request.Device.IP)
			} else if request.Device.IPv6 != "" {
				headers.Add("X-Forwarded-For", request.Device.IPv6)
			}
		}

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
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
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
			if bidType == openrtb_ext.BidTypeNative {
				bid.AdM = unwrapNativeAdm(bid.AdM)
			}
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

// unwrapNativeAdm returns the native response from an adm of the form {"native":{...}}, which is how the exchange
// sends it. Prebid Server core reads adm as a bare native response when it fills in the asset types. Anything else
// is returned unchanged.
func unwrapNativeAdm(adm string) string {
	var wrapper struct {
		Native json.RawMessage `json:"native"`
	}
	if err := jsonutil.Unmarshal([]byte(adm), &wrapper); err != nil || len(wrapper.Native) == 0 {
		return adm
	}
	return string(wrapper.Native)
}

// getMediaTypeForBid uses bid.mtype, which the exchange sets on every bid. Inferring the type from the
// impression is only a safety net for a bid without it, and only works for single-format impressions.
// A bid for an impression the request does not have is refused, with or without mtype.
func getMediaTypeForBid(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	imp := findImp(bid.ImpID, imps)
	if imp == nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid references unknown imp %s", bid.ImpID),
		}
	}

	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case 0:
		// no mtype: fall through to the impression
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported bid.mtype %d for imp %s", bid.MType, bid.ImpID),
		}
	}

	if imp.Banner != nil && imp.Video == nil && imp.Native == nil {
		return openrtb_ext.BidTypeBanner, nil
	}
	if imp.Video != nil && imp.Banner == nil && imp.Native == nil {
		return openrtb_ext.BidTypeVideo, nil
	}
	if imp.Native != nil && imp.Banner == nil && imp.Video == nil {
		return openrtb_ext.BidTypeNative, nil
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("bid.mtype is required to resolve the media type of imp %s", bid.ImpID),
	}
}

func findImp(impID string, imps []openrtb2.Imp) *openrtb2.Imp {
	for i := range imps {
		if imps[i].ID == impID {
			return &imps[i]
		}
	}
	return nil
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
