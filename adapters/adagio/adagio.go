package adagio

import (
	"encoding/json"
	"errors"
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

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	reqJSON, err := jsonutil.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.FailedToUnmarshal{Message: fmt.Errorf("bid response, err: %w", err).Error()}}
	}

	if len(bidResponse.SeatBid) == 0 {
		return nil, []error{errors.New("empty seatbid array")}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponse()
	if bidResponse.Cur != "" {
		bidderResponse.Currency = bidResponse.Cur
	}

	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]

			bidType, err := getBidType(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidExt, bidExtErr := getBidExt(bid.Ext)
			if bidExtErr != nil {
				errs = append(errs, &errortypes.FailedToUnmarshal{
					Message: fmt.Errorf("bid ext, err: %w", bidExtErr).Error(),
				})
				continue
			}

			var meta *openrtb_ext.ExtBidPrebidMeta
			var video *openrtb_ext.ExtBidPrebidVideo
			if bidExt.Prebid != nil {
				meta = bidExt.Prebid.Meta
				video = bidExt.Prebid.Video
			}

			typedBid := &adapters.TypedBid{
				Bid:      &bid,
				BidMeta:  meta,
				BidVideo: video,
				Seat:     openrtb_ext.BidderName(seatBid.Seat),
				BidType:  bidType,
			}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, errs
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Could not define media type for impression: %s", bid.ImpID),
	}
}

func getBidExt(ext json.RawMessage) (openrtb_ext.ExtBid, error) {
	var bidExt openrtb_ext.ExtBid
	if len(ext) == 0 {
		return bidExt, nil
	}

	err := jsonutil.Unmarshal(ext, &bidExt)
	return bidExt, err
}
