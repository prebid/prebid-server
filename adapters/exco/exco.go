package exco

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type ExcoAdapter struct {
	endpoint string
}

func (a *ExcoAdapter) MakeRequests(
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
	reqJSON, err := json.Marshal(adjustedReq)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{reqData}, errs
}

func (a *ExcoAdapter) MakeBids(
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

	var appnexusResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &appnexusResponse); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponse()
	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(&bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

func Builder(
	bidderName openrtb_ext.BidderName,
	config config.Adapter,
	server config.Server,
) (adapters.Bidder, error) {
	return &ExcoAdapter{
		endpoint: config.Endpoint,
	}, nil
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case 1:
		return openrtb_ext.BidTypeBanner, nil
	case 2:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unrecognized bid_ad_type in response from appnexus: %d", bid.MType)
	}
}

func adjustRequest(
	request *openrtb2.BidRequest,
) (*openrtb2.BidRequest, error) {
	var publisherId string

	for i := range request.Imp {
		var impExt openrtb_ext.ExtImpPrebid

		if request.Imp[i].Ext != nil {
			if err := json.Unmarshal(request.Imp[i].Ext, &impExt); err != nil {
				continue
			}

			if err := json.Unmarshal(impExt.Bidder["tagId"], &request.Imp[i].TagID); err != nil {
				return request, &errortypes.BadInput{
					Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Infomation: %s", i, "Missing tagId"),
				}
			}

			if err := json.Unmarshal(impExt.Bidder["publisherId"], &publisherId); err != nil {
				return request, &errortypes.BadInput{
					Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Infomation: %s", i, "Missing publisherId"),
				}
			}
		}
	}

	if request.Site == nil {
		request.Site = &openrtb2.Site{}
	}

	if request.Site.Publisher == nil {
		request.Site.Publisher = &openrtb2.Publisher{}
	}

	request.Site.Publisher.ID = publisherId

	return request, nil
}
