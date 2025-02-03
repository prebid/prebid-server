package readpeak

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Readpeak adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	requestCopy := *request
	var rpExt openrtb_ext.ImpExtReadpeak
	var imps []openrtb2.Imp
	for i := 0; i < len(requestCopy.Imp); i++ {
		var impExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(requestCopy.Imp[i].Ext, &impExt); err != nil {
			errors = append(errors, err)
			continue
		}
		if err := jsonutil.Unmarshal(impExt.Bidder, &rpExt); err != nil {
			errors = append(errors, err)
			continue
		}
		imp := requestCopy.Imp[i]
		if rpExt.TagId != "" {
			imp.TagID = rpExt.TagId
		}
		if rpExt.Bidfloor != 0 {
			imp.BidFloor = rpExt.Bidfloor
		}
		imps = append(imps, imp)
	}

	if len(imps) == 0 {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to find compatible impressions for request %s", requestCopy.ID),
		}
		return nil, []error{err}
	}
	requestCopy.Imp = imps
	publisher := &openrtb2.Publisher{
		ID: rpExt.PublisherId,
	}

	if requestCopy.Site != nil {
		siteCopy := *request.Site
		if rpExt.SiteId != "" {
			siteCopy.ID = rpExt.SiteId
		}
		siteCopy.Publisher = publisher
		requestCopy.Site = &siteCopy
	} else if requestCopy.App != nil {
		appCopy := *request.App
		if rpExt.SiteId != "" {
			appCopy.ID = rpExt.SiteId
		}
		appCopy.Publisher = publisher
		requestCopy.App = &appCopy
	}

	requestJSON, err := json.Marshal(requestCopy)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, errors
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
	if len(response.Cur) != 0 {
		bidResponse.Currency = response.Cur
	}
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
			if err != nil {
				errors = append(errors, err)
				continue
			}
			resolveMacros(&seatBid.Bid[i])
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
				BidMeta: getBidMeta(&seatBid.Bid[i]),
			})
		}
	}
	return bidResponse, errors
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid != nil {
		price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
		bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
		bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
		bid.BURL = strings.Replace(bid.BURL, "${AUCTION_PRICE}", price, -1)
	}
}
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Failed to find impression type \"%s\"", bid.ImpID)
	}
}

func getBidMeta(bid *openrtb2.Bid) *openrtb_ext.ExtBidPrebidMeta {
	return &openrtb_ext.ExtBidPrebidMeta{
		AdvertiserDomains: bid.ADomain,
	}
}
