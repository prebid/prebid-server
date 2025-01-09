package consumable

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type adapter struct {
	endpoint string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}
	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	if request.Site != nil {
		_, consumableExt, err := extractExtensions(request.Imp[0])
		if err != nil {
			return nil, err
		}
		if consumableExt.SiteId == 0 && consumableExt.NetworkId == 0 && consumableExt.UnitId == 0 {
			return nil, []error{&errortypes.FailedToRequestBids{
				Message: "SiteId, NetworkId and UnitId are all required for site requests",
			}}
		}
		requests := []*adapters.RequestData{
			{
				Method:  "POST",
				Uri:     a.endpoint + "/sb/rtb",
				Body:    bodyBytes,
				Headers: headers,
				ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
			},
		}
		return requests, errs

	} else {
		_, consumableExt, err := extractExtensions(request.Imp[0])
		if err != nil {
			return nil, err
		}

		if consumableExt.PlacementId == "" {
			return nil, []error{&errortypes.FailedToRequestBids{
				Message: "PlacementId is required for non-site requests",
			}}
		}
		requests := []*adapters.RequestData{
			{
				Method:  "POST",
				Uri:     a.endpoint + "/rtb/bid?s=" + consumableExt.PlacementId,
				Body:    bodyBytes,
				Headers: headers,
				ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
			},
		}
		return requests, errs
	}
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
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				continue
			}
			var bidVideo *openrtb_ext.ExtBidPrebidVideo
			if bidType == openrtb_ext.BidTypeVideo {
				bidVideo = &openrtb_ext.ExtBidPrebidVideo{Duration: int(bid.Dur)}
			}
			switch bidType {
			case openrtb_ext.BidTypeAudio:
				seatBid.Bid[i].MType = openrtb2.MarkupAudio
			case openrtb_ext.BidTypeVideo:
				seatBid.Bid[i].MType = openrtb2.MarkupVideo
			case openrtb_ext.BidTypeBanner:
				seatBid.Bid[i].MType = openrtb2.MarkupBanner
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &seatBid.Bid[i],
				BidType:  bidType,
				BidVideo: bidVideo,
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.MType != 0 {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		case openrtb2.MarkupAudio:
			return openrtb_ext.BidTypeAudio, nil
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}

// Builder builds a new instance of the Consumable adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func extractExtensions(impression openrtb2.Imp) (*adapters.ExtImpBidder, *openrtb_ext.ExtImpConsumable, []error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(impression.Ext, &bidderExt); err != nil {
		return nil, nil, []error{&errortypes.BadInput{
			Message: err.Error(),
		}}
	}

	var consumableExt openrtb_ext.ExtImpConsumable
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &consumableExt); err != nil {
		return nil, nil, []error{&errortypes.BadInput{
			Message: err.Error(),
		}}
	}

	return &bidderExt, &consumableExt, nil
}
