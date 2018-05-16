package sharethrough

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const hbEndpoint = "https://dumb-waiter.sharethrough.com/header-bid/v1"

func NewSharethroughBidder(client *http.Client, endpoint string) *SharethroughAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}

	return &SharethroughAdapter{
		http: adapter,
		URI:  endpoint,
	}
}

// SharethroughAdapter converts the Sharethrough Adserver response into a
// prebid server compatible format
type SharethroughAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name returns the adapter name as a string
func (s SharethroughAdapter) Name() string {
	return "sharethrough"
}

type params struct {
	BidID        string `json:"bidId"`
	PlacementKey string `json:"placement_key"`
	HBVersion    string `json:"hbVersion"`
	StrVersion   string `json:"strVersion"`
	HBSource     string `json:"hbSource"`
}

func (s SharethroughAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	pKeys := make([]string, 0, len(request.Imp))
	errs := make([]error, 0, len(request.Imp))
	headers := http.Header{}
	var potentialRequests []*adapters.RequestData

	headers.Add("Content-Type", "text/plain;charset=utf-8")
	headers.Add("Accept", "application/json")

	for i := 0; i < len(request.Imp); i++ {
		pKey, err := preprocess(&request.Imp[i])
		if pKey != "" {
			pKeys = append(pKeys, pKey)
		}

		// If the preprocessing failed, the server won't be able to bid on this Imp. Delete it, and note the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
			continue
		}

		hbURI := generateHBUri(pKey, "testBidID-"+string(i))
		potentialRequests = append(potentialRequests, &adapters.RequestData{
			Method:  "GET",
			Uri:     hbURI,
			Body:    nil,
			Headers: headers,
		})
	}

	return potentialRequests, errs
}

func (s SharethroughAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&adapters.BadInputError{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if bidType, err := getMediaTypeForBid(&bid); err == nil {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			} else {
				errs = append(errs, err)
			}
		}
	}
	return bidResponse, errs
}
func preprocess(imp *openrtb.Imp) (pKey string, err error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", err
	}

	var sharethroughExt openrtb_ext.ExtImpSharethrough
	if err := json.Unmarshal(bidderExt.Bidder, &sharethroughExt); err != nil {
		return "", err
	}

	return sharethroughExt.PlacementKey, nil
}

func generateHBUri(pKey string, bidID string) string {
	v := url.Values{}
	v.Set("placement_key", pKey)
	v.Set("bidId", bidID)
	v.Set("hbVersion", "test-version")
	v.Set("hbSource", "prebid-server")

	return hbEndpoint + "?" + v.Encode()
}
