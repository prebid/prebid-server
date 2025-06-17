package visx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type VisxAdapter struct {
	endpoint string
}

type visxBidExtPrebidMeta struct {
	MediaType openrtb_ext.BidType `json:"mediaType"`
}

type visxBidExtPrebid struct {
	Meta visxBidExtPrebidMeta `json:"meta"`
}

type visxBidExt struct {
	Prebid visxBidExtPrebid `json:"prebid"`
}

type visxBid struct {
	ImpID   string          `json:"impid"`
	Price   float64         `json:"price"`
	UID     int             `json:"auid"`
	CrID    string          `json:"crid,omitempty"`
	AdM     string          `json:"adm,omitempty"`
	ADomain []string        `json:"adomain,omitempty"`
	DealID  string          `json:"dealid,omitempty"`
	W       uint64          `json:"w,omitempty"`
	H       uint64          `json:"h,omitempty"`
	Ext     json.RawMessage `json:"ext,omitempty"`
}

type visxSeatBid struct {
	Bid  []visxBid `json:"bid"`
	Seat string    `json:"seat,omitempty"`
}

type visxResponse struct {
	SeatBid []visxSeatBid `json:"seatbid,omitempty"`
	Cur     string        `json:"cur,omitempty"`
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *VisxAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors = make([]error, 0)

	// copy the request, because we are going to mutate it
	requestCopy := *request
	if len(requestCopy.Cur) == 0 {
		requestCopy.Cur = []string{"USD"}
	}

	reqJSON, err := json.Marshal(requestCopy)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	if request.Device != nil {
		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}

		if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
	}}, errors
}

// MakeBids unpacks the server's response into Bids.
func (a *VisxAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp visxResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := openrtb2.Bid{}
			bid.ID = internalRequest.ID
			bid.CrID = sb.Bid[i].CrID
			bid.ImpID = sb.Bid[i].ImpID
			bid.Price = sb.Bid[i].Price
			bid.AdM = sb.Bid[i].AdM
			bid.W = int64(sb.Bid[i].W)
			bid.H = int64(sb.Bid[i].H)
			bid.ADomain = sb.Bid[i].ADomain
			bid.DealID = sb.Bid[i].DealID
			bid.Ext = sb.Bid[i].Ext

			bidType, err := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp, sb.Bid[i])
			if err != nil {
				return nil, []error{err}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	return bidResponse, nil

}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp, bid visxBid) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			var ext visxBidExt
			if err := jsonutil.Unmarshal(bid.Ext, &ext); err == nil {
				if ext.Prebid.Meta.MediaType == openrtb_ext.BidTypeBanner {
					return openrtb_ext.BidTypeBanner, nil
				}
				if ext.Prebid.Meta.MediaType == openrtb_ext.BidTypeVideo {
					return openrtb_ext.BidTypeVideo, nil
				}
			}

			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}

			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}

			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unknown impression type for ID: \"%s\"", impID),
			}
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to find impression for ID: \"%s\"", impID),
	}
}

// Builder builds a new instance of the Visx adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &VisxAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
