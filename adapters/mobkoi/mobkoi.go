package mobkoi

import (
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

type BidderExt struct {
	Bidder openrtb_ext.ImpExtMobkoi `json:"bidder"`
}

type UserExt struct {
	Consent string `json:"consent"`
}

// Builder builds a new instance of the {bidder} adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	ext := BidderExt{}
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &ext); err != nil {
		return nil, []error{err}
	}

	if request.Imp[0].TagID == "" {
		request.Imp[0].TagID = ext.Bidder.PlacementID
	}

	uri := a.endpoint
	if ext.Bidder.AdServerBaseUrl != "" {
		uri = ext.Bidder.AdServerBaseUrl + "/bid"
	}

	if request.User != nil && request.User.Consent != "" {
		user := *request.User
		userExt, err := jsonutil.Marshal(UserExt{
			Consent: user.Consent,
		})
		if err != nil {
			return nil, []error{err}
		}
		user.Ext = userExt
		request.User = &user
	}

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    requestJSON,
		Headers: headers,
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
				Seat:    "mobkoi",
			})

		}
	}
	return bidResponse, nil
}
