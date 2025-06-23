package fwssp

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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	for i := 0; i < len(request.Imp); i++ {
		imp := &request.Imp[i]
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", i, err.Error()),
			}}
		}

		var impExt openrtb_ext.ImpExtFWSSP
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", i, err.Error()),
			}}
		}

		var err error
		if imp.Ext, err = jsonutil.Marshal(impExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Unable to transfer requestImpExt to Json fomat, %s", err.Error()),
			}}
		}
	}

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unable to transfer request to Json fomat, %s", err.Error()),
		}}
	}

	headers := http.Header{}
	headers.Add("Componentid", "prebid-go")

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}
	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) ||
		(response.StatusCode == http.StatusOK && len(response.Body) == 0) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	bidResponse.Currency = bidResp.Cur

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidVideo := openrtb_ext.ExtBidPrebidVideo{}
			if len(bid.Cat) > 0 {
				bidVideo.PrimaryCategory = bid.Cat[0]
			}
			if bid.Dur > 0 {
				bidVideo.Duration = int(bid.Dur)
			}
			adTypeBid := &adapters.TypedBid{
				Bid:      &bid,
				BidType:  openrtb_ext.BidTypeVideo,
				BidVideo: &bidVideo,
			}
			bidResponse.Bids = append(bidResponse.Bids, adTypeBid)
		}
	}
	return bidResponse, nil
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		config.Endpoint,
	}
	return bidder, nil
}
