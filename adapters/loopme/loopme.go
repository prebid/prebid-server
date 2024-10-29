package loopme

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/v2/errortypes"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type LoopmeAdapter struct {
	Endpoint string
}

func (a *LoopmeAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	reqDatas := make([]*adapters.RequestData, 0, len(request.Imp))
	for _, imp := range request.Imp {
		_, err := parseBidderExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestCopy := *request
		requestCopy.Imp = []openrtb2.Imp{imp}
		reqJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")
		reqDatas = append(reqDatas, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.Endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		})

	}
	return reqDatas, errs
}

func (a *LoopmeAdapter) MakeBids(bidReq *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(respData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(respData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(respData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 || len(bidResp.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	resp := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	if len(bidResp.Cur) != 0 {
		resp.Currency = bidResp.Cur
	}
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := &sb.Bid[i]
			bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}
			resp.Bids = append(resp.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}
	return resp, nil
}

func parseBidderExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpLoopme, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}

	var loopmeExt openrtb_ext.ExtImpLoopme
	if err := json.Unmarshal(bidderExt.Bidder, &loopmeExt); err != nil {
		return nil, fmt.Errorf("Wrong Loopme bidder ext")
	}
	return &loopmeExt, nil
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d", bid.MType),
		}
	}
}

// Builder builds a new instance of the Loopme adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, serverCfg config.Server) (adapters.Bidder, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint is empty")
	}

	bidder := &LoopmeAdapter{
		Endpoint: cfg.Endpoint,
	}
	return bidder, nil
}
