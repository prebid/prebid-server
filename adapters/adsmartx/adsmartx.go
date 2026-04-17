package adsmartx

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

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	validImps := make([]openrtb2.Imp, 0, len(request.Imp))
	var setTestMode bool

	for _, imp := range request.Imp {
		impExt, err := parseImpExt(imp.Ext)
		if err != nil {
			errs = append(errs, fmt.Errorf("impID %s: %w", imp.ID, err))
			continue
		}

		if imp.Banner == nil && imp.Video == nil {
			errs = append(errs, fmt.Errorf("impID %s: no banner or video object specified", imp.ID))
			continue
		}

		if imp.BidFloor == 0 && impExt.BidFloor > 0 {
			imp.BidFloor = impExt.BidFloor
		}

		if impExt.TestMode == 1 {
			setTestMode = true
		}

		validImps = append(validImps, imp)
	}

	if len(validImps) == 0 {
		return nil, append(errs, fmt.Errorf("no valid impressions"))
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

func parseImpExt(ext jsonutil.RawMessage) (openrtb_ext.ImpExtAdsmartx, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(ext, &bidderExt); err != nil {
		return openrtb_ext.ImpExtAdsmartx{}, err
	}
	var adsmartxExt openrtb_ext.ImpExtAdsmartx
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adsmartxExt); err != nil {
		return openrtb_ext.ImpExtAdsmartx{}, err
	}
	return adsmartxExt, nil
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

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getBidType(bid.MType)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			br.Bids = append(br.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	return br, errs
}

func getBidType(mtype openrtb2.MarkupType) (openrtb_ext.BidType, error) {
	switch mtype {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unknown bid type mtype=%d", mtype),
		}
	}
}
