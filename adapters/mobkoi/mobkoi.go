package mobkoi

import (
	"fmt"
	"net/http"
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

	if request.User.Consent != "" {
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
		Method:  "POST",
		Uri:     uri,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			// Shadow to avoid loop var issue with Go < 1.22
			bid := bid

			macros := map[string]string{
				"${AUCTION_PRICE}":        fmt.Sprintf("%.2f", bid.Price),
				"${AUCTION_IMP_ID}":       bid.ImpID,
				"${AUCTION_CURRENCY}":     response.Cur,
				"${AUCTION_BID_ID}":       bid.ID,
				"${BIDDING_API_BASE_URL}": strings.TrimSuffix(requestData.Uri, "/bid"),
				"${CREATIVE_ID}":          bid.CrID,
				"${CAMPAIGN_ID}":          bid.CID,
				"${ORTB_ID}":              bid.ID,
			}

			bid.AdM = replaceMacros(bid.AdM, macros)
			bid.NURL = replaceMacros(bid.NURL, macros)
			bid.LURL = replaceMacros(bid.LURL, macros)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
				Seat:    "mobkoi",
			})

		}
	}
	return bidResponse, nil
}

func replaceMacros(input string, macros map[string]string) string {
	var pairs []string
	for key, value := range macros {
		pairs = append(pairs, key, value)
	}

	replacer := strings.NewReplacer(pairs...)
	return replacer.Replace(input)
}
