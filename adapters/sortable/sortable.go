package sortable

import (
	"encoding/json"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type SortableAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

type impExts struct {
	Bidder openrtb.RawJSON `json:"bidder"`
}

func NewSortableBidder(client *http.Client, endpoint string) *SortableAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &SortableAdapter{
		http: a,
		URI:  endpoint,
	}
}

func (s *SortableAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	// TODO: Add other headers here?

	for i, imp := range request.Imp {
		var extStuff impExts
		err := json.Unmarshal(imp.Ext, &extStuff)
		if err != nil {
			errs = append(errs, err)
		}
		request.Imp[i].Ext = extStuff.Bidder
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     s.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func (s *SortableAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}
	return bidResponse, nil
}
