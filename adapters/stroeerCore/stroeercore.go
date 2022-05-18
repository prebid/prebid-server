package stroeerCore

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type adapter struct {
	URL string `json:"url"`
}

type response struct {
	Bids []bidResponse `json:"bids"`
}

type bidResponse struct {
	ID     string  `json:"id"`
	BidID  string  `json:"bidId"`
	CPM    float64 `json:"cpm"`
	Width  int64   `json:"width"`
	Height int64   `json:"height"`
	Ad     string  `json:"ad"`
	CrID   string  `json:"crid"`
}

func (a *adapter) MakeBids(bidRequest *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected http status code: %d.", responseData.StatusCode),
		}}
	}

	var errors []error
	stroeerResponse := response{}

	if err := json.Unmarshal(responseData.Body, &stroeerResponse); err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(stroeerResponse.Bids))
	bidderResponse.Currency = "EUR"

	for _, bid := range stroeerResponse.Bids {
		openRtbBid := openrtb2.Bid{
			ID:    bid.ID,
			ImpID: bid.BidID,
			W:     bid.Width,
			H:     bid.Height,
			Price: bid.CPM,
			AdM:   bid.Ad,
			CrID:  bid.CrID,
		}

		bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
			Bid:     &openRtbBid,
			BidType: openrtb_ext.BidTypeBanner,
		})
	}

	return bidderResponse, errors
}

func (a *adapter) MakeRequests(bidRequest *openrtb2.BidRequest, extraRequestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	for idx := range bidRequest.Imp {
		imp := &bidRequest.Imp[idx]
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, err)
			continue
		}

		var stroeerExt openrtb_ext.ExtImpStroeerCore
		if err := json.Unmarshal(bidderExt.Bidder, &stroeerExt); err != nil {
			errors = append(errors, err)
			continue
		}

		imp.TagID = stroeerExt.Sid
	}

	reqJSON, err := json.Marshal(bidRequest)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.URL,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

// Builder builds a new instance of the StroeerCore adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		URL: config.Endpoint,
	}
	return bidder, nil
}
