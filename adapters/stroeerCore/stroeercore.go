package stroeerCore

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

type adapter struct {
	URL    string `json:"url"`
	Server config.Server
}

type response struct {
	Bids []bidResponse `json:"bids"`
}

type bidResponse struct {
	ID      string          `json:"id"`
	BidID   string          `json:"bidId"`
	CPM     float64         `json:"cpm"`
	Width   int64           `json:"width"`
	Height  int64           `json:"height"`
	Ad      string          `json:"ad"`
	CrID    string          `json:"crid"`
	Mtype   string          `json:"mtype"`
	DSA     json.RawMessage `json:"dsa"`
	ADomain []string        `json:"adomain,omitempty"`
}

type bidExt struct {
	DSA json.RawMessage `json:"dsa,omitempty"`
}

func (b *bidResponse) resolveMediaType() (mt openrtb2.MarkupType, bt openrtb_ext.BidType, err error) {
	switch b.Mtype {
	case "banner":
		return openrtb2.MarkupBanner, openrtb_ext.BidTypeBanner, nil
	case "video":
		return openrtb2.MarkupVideo, openrtb_ext.BidTypeVideo, nil
	default:
		return mt, bt, fmt.Errorf("unable to determine media type for bid with id \"%s\"", b.BidID)
	}
}

func (a *adapter) MakeBids(bidRequest *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected http status code: %d.", responseData.StatusCode),
		}}
	}

	var errors []error
	stroeerResponse := response{}

	if err := jsonutil.Unmarshal(responseData.Body, &stroeerResponse); err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(stroeerResponse.Bids))
	bidderResponse.Currency = "EUR"

	for _, bid := range stroeerResponse.Bids {
		markupType, bidType, err := bid.resolveMediaType()
		if err != nil {
			errors = append(errors, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Bid media type error: %s", err.Error()),
			})
			continue
		}

		openRtbBid := openrtb2.Bid{
			ID:      bid.ID,
			ImpID:   bid.BidID,
			W:       bid.Width,
			H:       bid.Height,
			Price:   bid.CPM,
			AdM:     bid.Ad,
			CrID:    bid.CrID,
			MType:   markupType,
			ADomain: bid.ADomain,
		}

		if bid.DSA != nil {
			dsaJson, err := json.Marshal(bidExt{bid.DSA})
			if err != nil {
				errors = append(errors, err)
			} else {
				openRtbBid.Ext = dsaJson
			}
		}

		bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
			Bid:     &openRtbBid,
			BidType: bidType,
		})
	}

	return bidderResponse, errors
}

func (a *adapter) MakeRequests(bidRequest *openrtb2.BidRequest, extraRequestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	for idx := range bidRequest.Imp {
		imp := &bidRequest.Imp[idx]
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, err)
			continue
		}

		var stroeerExt openrtb_ext.ExtImpStroeerCore
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &stroeerExt); err != nil {
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
		ImpIDs:  openrtb_ext.GetImpIDs(bidRequest.Imp),
	}}, errors
}

// Builder builds a new instance of the StroeerCore adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		URL: config.Endpoint,
	}
	return bidder, nil
}
