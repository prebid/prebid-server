package exco

import (
	"fmt"
	"net/http"

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

func Builder(
	bidderName openrtb_ext.BidderName,
	config config.Adapter,
	server config.Server,
) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(
	request *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) ([]*adapters.RequestData, []error) {
	var errs []error

	adjustedReq, err := adjustRequest(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	// Create the request to the Exco endpoint
	reqjsonutil, err := jsonutil.Marshal(adjustedReq)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/jsonutil;charset=utf-8")

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqjsonutil,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{reqData}, errs
}

func (a *adapter) MakeBids(
	internalRequest *openrtb2.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponse()
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case 1:
		return openrtb_ext.BidTypeBanner, nil
	case 2:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unrecognized bid_ad_type in response from exco: %d", bid.MType)
	}
}

func adjustRequest(
	request *openrtb2.BidRequest,
) (*openrtb2.BidRequest, error) {
	var publisherId string

	for i := 0; i < len(request.Imp); i++ {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, &bidderExt); err != nil {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Information: %s", i, err.Error()),
			}
		}

		var impExt openrtb_ext.ImpExtExco
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Information: %s", i, err.Error()),
			}
		}

		if impExt.PublisherId == "" {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Information: %s", i, "Missing publisherId"),
			}
		}

		if impExt.TagId == "" {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Information: %s", i, "Missing tagId"),
			}
		}

		if impExt.AccountId == "" {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Information: %s", i, "Missing accountId"),
			}
		}

		publisherId = impExt.PublisherId
		request.Imp[i].TagID = impExt.TagId
	}

	if request.App != nil {
		appCopy := *request.App
		request.App = &appCopy

		if request.App.Publisher == nil {
			request.App.Publisher = &openrtb2.Publisher{}
		} else {
			publisherCopy := *request.App.Publisher
			request.App.Publisher = &publisherCopy
		}

		request.App.Publisher.ID = publisherId
	}

	if request.Site != nil {
		siteCopy := *request.Site
		request.Site = &siteCopy

		if request.Site.Publisher == nil {
			request.Site.Publisher = &openrtb2.Publisher{}
		} else {
			publisherCopy := *request.Site.Publisher
			request.Site.Publisher = &publisherCopy
		}

		request.Site.Publisher.ID = publisherId
	}

	return request, nil
}
