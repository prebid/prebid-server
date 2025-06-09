package blis

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iterutil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var impExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &impExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}
	var impExtBidder openrtb_ext.ImpExtBlis
	if err := jsonutil.Unmarshal(impExt.Bidder, &impExtBidder); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}

	headers := http.Header{}
	headers.Add("X-Supply-Partner-Id", impExtBidder.SupplyPartnerID)

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Headers: headers,
		Body:    requestJSON,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
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
	bidResponse.Currency = response.Cur
	var errors []error
	for seatBid := range iterators.SlicePointerValues(response.SeatBid) {
		for bid := range iterators.SlicePointerValues(seatBid.Bid) {
			resolveMacros(bid)
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse media type of impression ID \"%s\"", bid.ImpID),
	}
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid != nil {
		price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
		bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
		bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
		bid.BURL = strings.Replace(bid.BURL, "${AUCTION_PRICE}", price, -1)
	}
}
