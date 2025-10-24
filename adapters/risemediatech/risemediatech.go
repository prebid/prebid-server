package risemediatech

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iterutil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb2.Imp
	var setTestMode bool

	for imp := range iterutil.SlicePointerValues(request.Imp) {
		impExt, err := parseImpExt(imp.Ext)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("impID %s: %v", imp.ID, err)})
			continue
		}

		if impExt.BidFloor > 0 {
			imp.BidFloor = impExt.BidFloor
		}

		if impExt.TestMode == 1 {
			setTestMode = true
		}

		validImps = append(validImps, *imp)
	}

	if len(validImps) == 0 {
		return nil, append(errs, &errortypes.BadInput{Message: "no valid impressions"})
	}

	request.Imp = validImps
	if setTestMode {
		request.Test = 1
	}

	reqJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json;charset=utf-8")
	headers.Set("Accept", "application/json")

	return []*adapters.RequestData{
		{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(validImps),
		},
	}, errs
}

func parseImpExt(ext jsonutil.RawMessage) (openrtb_ext.ExtImpRiseMediaTech, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpRiseMediaTech{}, err
	}
	var riseExt openrtb_ext.ExtImpRiseMediaTech
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &riseExt); err != nil {
		return openrtb_ext.ExtImpRiseMediaTech{}, err
	}
	return riseExt, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(respData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(respData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(respData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	br := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid))
	if bidResp.Cur != "" {
		br.Currency = bidResp.Cur
	}

	for seatBid := range iterutil.SlicePointerValues(bidResp.SeatBid) {
		for bid := range iterutil.SlicePointerValues(seatBid.Bid) {
			bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}

			typedBid := &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			}

			br.Bids = append(br.Bids, typedBid)
		}
	}
	return br, nil
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unknown bid type mtype=%d", bid.MType)
	}
}
